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
	err = h.redisClient.Set(ctx, 
		"user_tokens:"+tokenString, 
		user.ID, 
		time.Hour*24*7, // 1 week
	).Err()
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store token"})
		return
	}

	// Return token to client
	c.JSON(http.StatusOK, gin.H{
		"token": tokenString,
		"user": gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"email":     user.Email,
			"lastLogin": now,
		},
	})
}

// Logout handles user logout
func (h *UserHandler) Logout(c *gin.Context) {
	// Get token from authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization header is required"})
		return
	}

	// Check if the header has the Bearer format
	parts := authHeader[7:] // Remove "Bearer " prefix
	
	// Add token to blacklist
	ctx := c.Request.Context()
	err := h.redisClient.Set(ctx, 
		"blacklist:"+parts, 
		true, 
		time.Hour*24*7, // Same as token expiration
	).Err()
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logout successful"})
}

// GetProfile returns the current user's profile
func (h *UserHandler) GetProfile(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Fetch user from database
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Get scholar profile if exists
	var scholarProfile models.ScholarProfile
	h.db.Where("user_id = ?", user.ID).First(&scholarProfile)

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":              user.ID,
			"username":        user.Username,
			"email":           user.Email,
			"dateJoined":      user.DateJoined,
			"lastLogin":       user.LastLogin,
			"profileImageURL": user.ProfileImageURL,
			"biography":       user.Biography,
			"institution":     user.Institution,
		},
		"scholarProfile": gin.H{
			"researchArea": scholarProfile.ResearchArea,
			"citations":    scholarProfile.Citations,
			"hIndex":       scholarProfile.HIndex,
			"i10Index":     scholarProfile.I10Index,
		},
	})
}

// UpdateProfile updates the current user's profile
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Bind input data
	var input models.UserProfile
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch user from database
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update user fields
	updateData := map[string]interface{}{
		"biography":        input.Biography,
		"institution":      input.Institution,
		"profile_image_url": input.ProfileImageURL,
	}

	// Only update username if provided and different
	if input.Username != "" && input.Username != user.Username {
		// Check if username is already taken
		var existingUser models.User
		result := h.db.Where("username = ? AND id != ?", input.Username, user.ID).First(&existingUser)
		if result.RowsAffected > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username already taken"})
			return
		}
		updateData["username"] = input.Username
	}

	// Only update email if provided and different
	if input.Email != "" && input.Email != user.Email {
		// Check if email is already taken
		var existingUser models.User
		result := h.db.Where("email = ? AND id != ?", input.Email, user.ID).First(&existingUser)
		if result.RowsAffected > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email already taken"})
			return
		}
		updateData["email"] = input.Email
	}

	// Update user in database
	if err := h.db.Model(&user).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
}

// RequestPasswordReset initiates the password reset process
func (h *UserHandler) RequestPasswordReset(c *gin.Context) {
	var input struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user by email
	var user models.User
	result := h.db.Where("email = ?", input.Email).First(&user)
	if result.RowsAffected == 0 {
		// Don't reveal that the email doesn't exist
		c.JSON(http.StatusOK, gin.H{"message": "If your email is registered, you will receive a password reset link"})
		return
	}

	// Generate reset token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(time.Hour * 24).Unix(), // 24 hours
		"type": "password_reset",
	})

	// Sign token
	tokenString, err := token.SignedString([]byte(h.config.JWT.Secret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate reset token"})
		return
	}

	// Store token in Redis
	ctx := c.Request.Context()
	err = h.redisClient.Set(ctx, 
		"password_reset:"+tokenString, 
		user.ID, 
		time.Hour*24, // 24 hours
	).Err()
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process request"})
		return
	}

	// TODO: Send email with reset link
	// For now, just return the token in the response
	c.JSON(http.StatusOK, gin.H{
		"message": "If your email is registered, you will receive a password reset link",
		"token": tokenString, // In production, this would be sent via email
	})
}

// ResetPassword resets a user's password using a reset token
func (h *UserHandler) ResetPassword(c *gin.Context) {
	token := c.Param("token")
	
	var input struct {
		Password string `json:"password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate the token
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.config.JWT.Secret), nil
	})

	if err != nil || !parsedToken.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired token"})
		return
	}

	// Check token type
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token"})
		return
	}

	if claims["type"] != "password_reset" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token type"})
		return
	}

	// Check if token exists in Redis
	ctx := c.Request.Context()
	_, err = h.redisClient.Get(ctx, "password_reset:"+token).Result()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired token"})
		return
	}

	// Get user ID from token
	userID, ok := claims["sub"].(float64)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token"})
		return
	}

	// Find user
	var user models.User
	if err := h.db.First(&user, uint(userID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Hash new password
	hashedPassword, err := models.HashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process request"})
		return
	}

	// Update password
	if err := h.db.Model(&user).Update("password", hashedPassword).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	// Delete token from Redis
	h.redisClient.Del(ctx, "password_reset:"+token)

	c.JSON(http.StatusOK, gin.H{"message": "Password has been reset successfully"})
}