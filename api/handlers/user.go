package handlers

import (
	"net/http"
	"time"

	"freescholar-backend/config"
	"freescholar-backend/internal/models"
	"freescholar-backend/pkg/redis"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// UserHandler handles HTTP requests related to users
type UserHandler struct {
	db          *gorm.DB
	redisClient *redis.Client
	config      *config.Config
}

// NewUserHandler creates a new user handler
func NewUserHandler(db *gorm.DB, redisClient *redis.Client, cfg *config.Config) *UserHandler {
	return &UserHandler{
		db:          db,
		redisClient: redisClient,
		config:      cfg,
	}
}

// Register handles user registration
func (h *UserHandler) Register(c *gin.Context) {
	var input models.UserRegister

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user already exists
	var existingUser models.User
	result := h.db.Where("email = ?", input.Email).First(&existingUser)
	if result.RowsAffected > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User with this email already exists"})
		return
	}

	result = h.db.Where("username = ?", input.Username).First(&existingUser)
	if result.RowsAffected > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User with this username already exists"})
		return
	}

	// Create new user
	user := models.User{
		Username:   input.Username,
		Email:      input.Email,
		Password:   input.Password, // Will be hashed by BeforeCreate hook
		IsActive:   true,
		DateJoined: time.Now(),
	}

	if err := h.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Registration successful"})
}

// Login handles user login
func (h *UserHandler) Login(c *gin.Context) {
	var input models.UserLogin

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user by email
	var user models.User
	result := h.db.Where("email = ?", input.Email).First(&user)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check password
	if !models.CheckPasswordHash(input.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Update last login time
	now := time.Now()
	h.db.Model(&user).Update("last_login", now)

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(time.Hour * 24 * 7).Unix(), // 1 week
	})

	// Sign and get the complete encoded token as a string
	tokenString, err := token.SignedString([]byte(h.config.JWT.Secret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Store token in Redis with user ID as key
	ctx := c.Request.Context()
	err = h.redisClient
}