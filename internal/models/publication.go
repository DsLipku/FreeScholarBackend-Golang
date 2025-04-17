package models

import (
	"time"

	"gorm.io/gorm"
)

// Publication represents a scholarly publication
type Publication struct {
	gorm.Model
	Title           string    `json:"title" gorm:"index;size:512;not null"`
	Abstract        string    `json:"abstract" gorm:"type:text"`
	DOI             string    `json:"doi" gorm:"uniqueIndex;size:255"`
	PublicationDate time.Time `json:"publication_date" gorm:"index"`
	Journal         string    `json:"journal" gorm:"index;size:255"`
	Volume          string    `json:"volume" gorm:"size:50"`
	Issue           string    `json:"issue" gorm:"size:50"`
	Pages           string    `json:"pages" gorm:"size:50"`
	Publisher       string    `json:"publisher" gorm:"size:255"`
	CitationCount   int       `json:"citation_count" gorm:"default:0"`
	URL             string    `json:"url" gorm:"size:512"`
	PDFPath         string    `json:"pdf_path" gorm:"size:512"`
	
	// Relationships
	Authors         []Author         `json:"authors" gorm:"many2many:publication_authors;"`
	Keywords        []Keyword        `json:"keywords" gorm:"many2many:publication_keywords;"`
}

// Author represents an author of publications
type Author struct {
	gorm.Model
	Name         string        `json:"name" gorm:"index;size:255;not null"`
	Institution  string        `json:"institution" gorm:"size:255"`
	Email        string        `json:"email" gorm:"size:255"`
	WebsiteURL   string        `json:"website_url" gorm:"size:512"`
	Biography    string        `json:"biography" gorm:"type:text"`
	Publications []Publication `json:"publications" gorm:"many2many:publication_authors;"`
}

// Keyword represents a keyword associated with publications
type Keyword struct {
	gorm.Model
	Name         string        `json:"name" gorm:"uniqueIndex;size:100;not null"`
	Publications []Publication `json:"publications" gorm:"many2many:publication_keywords;"`
}

// PublicationAuthor represents the relationship between publications and authors with ordering
type PublicationAuthor struct {
	gorm.Model
	PublicationID uint `json:"publication_id" gorm:"index;not null"`
	AuthorID      uint `json:"author_id" gorm:"index;not null"`
	Order         int  `json:"order" gorm:"not null;default:0"`
}

// ScholarProfile represents a scholar's profile
type ScholarProfile struct {
	gorm.Model
	UserID       uint   `json:"user_id" gorm:"uniqueIndex;not null"`
	User         User   `json:"user" gorm:"foreignKey:UserID"`
	ResearchArea string `json:"research_area" gorm:"type:text"`
	Citations    int    `json:"citations" gorm:"default:0"`
	HIndex       int    `json:"h_index" gorm:"default:0"`
	I10Index     int    `json:"i10_index" gorm:"default:0"`
}

// Relation represents a relationship between users (following/follower)
type Relation struct {
	gorm.Model
	FollowerID  uint `json:"follower_id" gorm:"index;not null"`
	FollowingID uint `json:"following_id" gorm:"index;not null"`
	Follower    User `json:"follower" gorm:"foreignKey:FollowerID"`
	Following   User `json:"following" gorm:"foreignKey:FollowingID"`
}

// Message represents a message between users
type Message struct {
	gorm.Model
	SenderID    uint      `json:"sender_id" gorm:"index;not null"`
	ReceiverID  uint      `json:"receiver_id" gorm:"index;not null"`
	Content     string    `json:"content" gorm:"type:text;not null"`
	IsRead      bool      `json:"is_read" gorm:"default:false"`
	ReadAt      time.Time `json:"read_at" gorm:"default:null"`
	Sender      User      `json:"sender" gorm:"foreignKey:SenderID"`
	Receiver    User      `json:"receiver" gorm:"foreignKey:ReceiverID"`
}

// File represents an uploaded file
type File struct {
	gorm.Model
	FileName    string `json:"file_name" gorm:"size:255;not null"`
	FilePath    string `json:"file_path" gorm:"size:512;not null"`
	FileSize    int64  `json:"file_size" gorm:"not null"`
	ContentType string `json:"content_type" gorm:"size:100;not null"`
	UploaderID  uint   `json:"uploader_id" gorm:"index"`
	Uploader    User   `json:"uploader" gorm:"foreignKey:UploaderID"`
}

// SearchHistory represents a user's search history
type SearchHistory struct {
	gorm.Model
	UserID    uint   `json:"user_id" gorm:"index"`
	User      User   `json:"user" gorm:"foreignKey:UserID"`
	Query     string `json:"query" gorm:"size:512;not null"`
	Category  string `json:"category" gorm:"size:50"`
	Count     int    `json:"count" gorm:"default:1"`
	IsSaved   bool   `json:"is_saved" gorm:"default:false"`
}

// Serialization represents a serialization of data
type Serialization struct {
	gorm.Model
	UserID      uint   `json:"user_id" gorm:"index"`
	User        User   `json:"user" gorm:"foreignKey:UserID"`
	Title       string `json:"title" gorm:"size:255;not null"`
	Description string `json:"description" gorm:"type:text"`
	Content     string `json:"content" gorm:"type:longtext;not null"`
	Format      string `json:"format" gorm:"size:50;not null"`
}

// PublicationSearch is the model for searching publications in Elasticsearch
type PublicationSearch struct {
	ID              uint      `json:"id"`
	Title           string    `json:"title"`
	Abstract        string    `json:"abstract"`
	Authors         []string  `json:"authors"`
	Keywords        []string  `json:"keywords"`
	DOI             string    `json:"doi"`
	PublicationDate time.Time `json:"publication_date"`
	Journal         string    `json:"journal"`
	CitationCount   int       `json:"citation_count"`
}