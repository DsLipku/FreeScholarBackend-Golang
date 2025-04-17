package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"freescholar-backend/config"
	"freescholar-backend/internal/models"
	"freescholar-backend/pkg/elasticsearch"

	"github.com/gin-gonic/gin"
	"github.com/olivere/elastic/v7"
	"gorm.io/gorm"
)

// PublicationHandler handles HTTP requests related to publications
type PublicationHandler struct {
	db       *gorm.DB
	esClient *elasticsearch.Client
	config   *config.Config
}

// NewPublicationHandler creates a new publication handler
func NewPublicationHandler(db *gorm.DB, esClient *elasticsearch.Client, cfg *config.Config) *PublicationHandler {
	return &PublicationHandler{
		db:       db,
		esClient: esClient,
		config:   cfg,
	}
}

// PublicationInput represents input for creating/updating publications
type PublicationInput struct {
	Title           string    `json:"title" binding:"required"`
	Abstract        string    `json:"abstract"`
	DOI             string    `json:"doi"`
	PublicationDate string    `json:"publication_date"` // Format: YYYY-MM-DD
	Journal         string    `json:"journal"`
	Volume          string    `json:"volume"`
	Issue           string    `json:"issue"`
	Pages           string    `json:"pages"`
	Publisher       string    `json:"publisher"`
	URL             string    `json:"url"`
	Keywords        []string  `json:"keywords"`
	Authors         []uint    `json:"authors"` // Author IDs
}

// GetPublications handles fetching multiple publications with filtering and pagination
func (h *PublicationHandler) GetPublications(c *gin.Context) {
	// Parse query parameters
	query := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	
	// Ensure reasonable pagination values
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	
	offset := (page - 1) * limit

	// If search query is provided, use Elasticsearch
	if query != "" {
		// Create search query for Elasticsearch
		esQuery := elastic.NewMultiMatchQuery(query, 
			"title^3", // Boost title relevance
			"abstract^2",
			"authors",
			"keywords",
			"journal",
		).Type("best_fields").Fuzziness("AUTO")
		
		searchResult, err := h.esClient.Search().
			Index("publications").
			Query(esQuery).
			From(offset).
			Size(limit).
			Sort("_score", false).  // Sort by relevance
			Sort("publication_date", false). // Then by date (newest first)
			Do(context.Background())
			
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Search error"})
			return
		}

		// Process search results
		var publications []models.PublicationSearch
		var total = searchResult.TotalHits()
		
		for _, hit := range searchResult.Hits.Hits {
			var publication models.PublicationSearch
			err := json.Unmarshal(hit.Source, &publication)
			if err != nil {
				continue
			}
			publications = append(publications, publication)
		}

		c.JSON(http.StatusOK, gin.H{
			"publications": publications,
			"total":        total,
			"page":         page,
			"limit":        limit,
			"pages":        (total + int64(limit) - 1) / int64(limit),
		})
		return
	}

	// Otherwise, use database query
	var publications []models.Publication
	var total int64

	db := h.db.Model(&models.Publication{})
	
	// Filter by journal if provided
	if journal := c.Query("journal"); journal != "" {
		db = db.Where("journal LIKE ?", "%"+journal+"%")
	}
	
	// Filter by date range if provided
	if fromDate := c.Query("from_date"); fromDate != "" {
		db = db.Where("publication_date >= ?", fromDate)
	}
	if toDate := c.Query("to_date"); toDate != "" {
		db = db.Where("publication_date <= ?", toDate)
	}
	
	// Get total count
	db.Count(&total)
	
	// Get paginated results with preloaded relationships
	err := db.Preload("Authors").Preload("Keywords").
		Offset(offset).
		Limit(limit).
		Order("publication_date DESC").
		Find(&publications).Error
		
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch publications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"publications": publications,
		"total":        total,
		"page":         page,
		"limit":        limit,
		"pages":        (total + int64(limit) - 1) / int64(limit),
	})
}

// GetPublication handles fetching a single publication by ID
func (h *PublicationHandler) GetPublication(c *gin.Context) {
	id := c.Param("id")
	
	var publication models.Publication
	err := h.db.Preload("Authors").Preload("Keywords").First(&publication, id).Error
	
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Publication not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"publication": publication})
}

// CreatePublication handles creating a new publication
func (h *PublicationHandler) CreatePublication(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var input PublicationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse publication date
	pubDate, err := time.Parse("2006-01-02", input.PublicationDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid publication date format. Use YYYY-MM-DD"})
		return
	}

	// Start a transaction
	tx := h.db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// Create publication
	publication := models.Publication{
		Title:           input.Title,
		Abstract:        input.Abstract,
		DOI:             input.DOI,
		PublicationDate: pubDate,
		Journal:         input.Journal,
		Volume:          input.Volume,
		Issue:           input.Issue,
		Pages:           input.Pages,
		Publisher:       input.Publisher,
		URL:             input.URL,
	}

	if err := tx.Create(&publication).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create publication"})
		return
	}

	// Process keywords
	for _, keyword := range input.Keywords {
		var existingKeyword models.Keyword
		result := tx.Where("name = ?", keyword).First(&existingKeyword)
		
		if result.RowsAffected == 0 {
			// Create new keyword if it doesn't exist
			existingKeyword = models.Keyword{Name: keyword}
			if err := tx.Create(&existingKeyword).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create keyword"})
				return
			}
		}
		
		// Associate keyword with publication
		if err := tx.Model(&publication).Association("Keywords").Append(&existingKeyword); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to associate keyword"})
			return
		}
	}

	// Process authors
	for i, authorID := range input.Authors {
		var author models.Author
		if err := tx.First(&author, authorID).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Author not found: " + strconv.Itoa(int(authorID))})
			return
		}
		
		// Create publication-author relationship with order
		pubAuthor := models.PublicationAuthor{
			PublicationID: publication.ID,
			AuthorID:      author.ID,
			Order:         i,
		}
		
		if err := tx.Create(&pubAuthor).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to associate author"})
			return
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Index in Elasticsearch
	go h.indexPublication(publication)

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Publication created successfully",
		"publication": publication,
	})
}

// Completing the UpdatePublication method that was cut off
func (h *PublicationHandler) UpdatePublication(c *gin.Context) {
	id := c.Param("id")
	
	// Get user ID from context (set by auth middleware)
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Check if publication exists
	var publication models.Publication
	if err := h.db.First(&publication, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Publication not found"})
		return
	}

	var input PublicationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse publication date
	var pubDate time.Time
	var err error
	if input.PublicationDate != "" {
		pubDate, err = time.Parse("2006-01-02", input.PublicationDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid publication date format. Use YYYY-MM-DD"})
			return
		}
	}

	// Start a transaction
	tx := h.db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// Update publication fields
	updates := map[string]interface{}{
		"title":           input.Title,
		"abstract":        input.Abstract,
		"doi":             input.DOI,
		"journal":         input.Journal,
		"volume":          input.Volume,
		"issue":           input.Issue,
		"pages":           input.Pages,
		"publisher":       input.Publisher,
		"url":             input.URL,
	}
	
	// Only update publication date if provided
	if !pubDate.IsZero() {
		updates["publication_date"] = pubDate
	}

	if err := tx.Model(&publication).Updates(updates).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update publication"})
		return
	}

	// Update keywords (clear and re-add)
	if len(input.Keywords) > 0 {
		// Remove existing keywords association
		if err := tx.Model(&publication).Association("Keywords").Clear(); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear keywords"})
			return
		}

		// Add new keywords
		for _, keyword := range input.Keywords {
			var existingKeyword models.Keyword
			result := tx.Where("name = ?", keyword).First(&existingKeyword)
			
			if result.RowsAffected == 0 {
				// Create new keyword if it doesn't exist
				existingKeyword = models.Keyword{Name: keyword}
				if err := tx.Create(&existingKeyword).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create keyword"})
					return
				}
			}
			
			// Associate keyword with publication
			if err := tx.Model(&publication).Association("Keywords").Append(&existingKeyword); err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to associate keyword"})
				return
			}
		}
	}

	// Update authors (if provided)
	if len(input.Authors) > 0 {
		// Delete existing publication-author relationships
		if err := tx.Where("publication_id = ?", publication.ID).Delete(&models.PublicationAuthor{}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear author associations"})
			return
		}

		// Create new publication-author relationships
		for i, authorID := range input.Authors {
			var author models.Author
			if err := tx.First(&author, authorID).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"error": "Author not found: " + strconv.Itoa(int(authorID))})
				return
			}
			
			// Create publication-author relationship with order
			pubAuthor := models.PublicationAuthor{
				PublicationID: publication.ID,
				AuthorID:      author.ID,
				Order:         i,
			}
			
			if err := tx.Create(&pubAuthor).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to associate author"})
				return
			}
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Re-fetch the publication with updated relationships
	h.db.Preload("Authors").Preload("Keywords").First(&publication, publication.ID)

	// Update in Elasticsearch
	go h.indexPublication(publication)

	c.JSON(http.StatusOK, gin.H{
		"message":     "Publication updated successfully",
		"publication": publication,
	})
}

// DeletePublication handles deleting a publication
func (h *PublicationHandler) DeletePublication(c *gin.Context) {
	id := c.Param("id")
	
	// Get user ID from context (set by auth middleware)
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Check if publication exists
	var publication models.Publication
	if err := h.db.First(&publication, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Publication not found"})
		return
	}

	// Start a transaction
	tx := h.db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// Delete all publication-author relationships
	if err := tx.Where("publication_id = ?", publication.ID).Delete(&models.PublicationAuthor{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete author associations"})
		return
	}

	// Remove all keyword associations (uses many-to-many relationship)
	if err := tx.Model(&publication).Association("Keywords").Clear(); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear keyword associations"})
		return
	}

	// Delete the publication
	if err := tx.Delete(&publication).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete publication"})
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Delete from Elasticsearch
	go func() {
		ctx := context.Background()
		_, err := h.esClient.Delete().
			Index("publications").
			Id(id).
			Do(ctx)
		
		if err != nil {
			// Log the error but don't fail the response
			// since MySQL deletion was successful
			log.Printf("Error deleting publication from Elasticsearch: %v", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "Publication deleted successfully",
	})
}

// indexPublication indexes a publication in Elasticsearch
func (h *PublicationHandler) indexPublication(publication models.Publication) {
	// Create a search model of the publication
	var authors []string
	for _, author := range publication.Authors {
		authors = append(authors, author.Name)
	}

	var keywords []string
	for _, keyword := range publication.Keywords {
		keywords = append(keywords, keyword.Name)
	}

	pubSearch := models.PublicationSearch{
		ID:              publication.ID,
		Title:           publication.Title,
		Abstract:        publication.Abstract,
		Authors:         authors,
		Keywords:        keywords,
		DOI:             publication.DOI,
		PublicationDate: publication.PublicationDate,
		Journal:         publication.Journal,
		CitationCount:   publication.CitationCount,
	}

	// Index document in Elasticsearch
	ctx := context.Background()
	id := strconv.Itoa(int(publication.ID))
	
	_, err := h.esClient.Index().
		Index("publications").
		Id(id).
		BodyJson(pubSearch).
		Do(ctx)
		
	if err != nil {
		// Log error but don't stop execution
		log.Printf("Failed to index publication in Elasticsearch: %v", err)
	}
}