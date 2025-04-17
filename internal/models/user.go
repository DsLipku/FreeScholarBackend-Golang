package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User model represents a user in the system
type User struct {
	gorm.Model
	Username        string     `json:"username" gorm:"uniqueIndex;size:255;not null"`
	Email           string     `json:"email" gorm:"uniqueIndex;size:255;not null"`
	Password        string     `json:"-" gorm:"size:255;not null"`
	IsActive        bool       `json:"is_active" gorm:"default:true"`
	IsAdmin         bool       `json:"is_admin" gorm:"default:false"`
	DateJoined      time.Time  `json:"date_joined" gorm:"not null;default:CURRENT_TIMESTAMP"`
	LastLogin       *time.Time `json:"last_login" gorm:"default:null"`
	ProfileImageURL string     `json:"profile_image_url" gorm:"size:255;default:''"`
	Biography       string     `json:"biography" gorm:"type:text"`
	Institution     string     `json:"institution" gorm:"size:255"`
}

// UserRegister is the data structure for user registration
type UserRegister struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// UserLogin is the data structure for user login
type UserLogin struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// UserProfile is the data structure for user profile updates
type UserProfile struct {
	Username        string `json:"username"`
	Email           string `json:"email"`
	ProfileImageURL string `json:"profile_image_url"`
	Biography       string `json:"biography"`
	Institution     string `json:"institution"`
}

// BeforeCreate hook is called before creating the user
func (u *User) BeforeCreate(tx *gorm.DB) error {
	// Hash password before storing
	hashedPwd, err := HashPassword(u.Password)
	if err != nil {
		return err
	}
	u.Password = hashedPwd
	return nil
}

// HashPassword generates a hashed version of the password
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// CheckPasswordHash compares a password with a hash to check if they match
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}