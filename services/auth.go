package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"law_flow_app_go/models"
	"log"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	// BcryptCost is the cost factor for bcrypt hashing
	BcryptCost = 10
	// SessionTokenLength is the length of the session token in bytes (64 chars hex)
	SessionTokenLength = 32
	// DefaultSessionDuration is the default session duration (7 days)
	DefaultSessionDuration = 7 * 24 * time.Hour
)

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

// CheckPassword verifies a password against a hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// VerifyPassword verifies a password against a bcrypt hash
func VerifyPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// GenerateSessionToken generates a cryptographically secure random token
func GenerateSessionToken() (string, error) {
	bytes := make([]byte, SessionTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate session token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// CreateSession creates a new session for a user
func CreateSession(db *gorm.DB, userID, firmID string, ipAddress, userAgent string) (*models.Session, error) {
	token, err := GenerateSessionToken()
	if err != nil {
		return nil, err
	}

	session := &models.Session{
		ID:        uuid.New().String(),
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(DefaultSessionDuration),
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}

	// Only set FirmID if not empty (allows NULL for superadmin)
	if firmID != "" {
		session.FirmID = &firmID
	}

	if err := db.Create(session).Error; err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// ValidateSession validates a session token and returns the session if valid
func ValidateSession(db *gorm.DB, token string) (*models.Session, error) {
	var session models.Session

	err := db.Preload("User.Firm").Preload("Firm").
		Where("token = ?", token).
		First(&session).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to validate session: %w", err)
	}

	if session.IsExpired() {
		// Delete expired session
		db.Delete(&session)
		return nil, fmt.Errorf("session expired")
	}

	return &session, nil
}

// DeleteSession deletes a session (logout)
func DeleteSession(db *gorm.DB, token string) error {
	result := db.Where("token = ?", token).Delete(&models.Session{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete session: %w", result.Error)
	}
	return nil
}

// CleanupExpiredSessions removes all expired sessions from the database
func CleanupExpiredSessions(db *gorm.DB) error {
	result := db.Where("expires_at < ?", time.Now()).Delete(&models.Session{})
	if result.Error != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", result.Error)
	}
	if result.RowsAffected > 0 {
		fmt.Printf("Cleaned up %d expired sessions\n", result.RowsAffected)
	}
	return nil
}

// DeleteAllUserSessions deletes all sessions for a specific user
// Used when password is reset for security
func DeleteAllUserSessions(db *gorm.DB, userID string) error {
	result := db.Where("user_id = ?", userID).Delete(&models.Session{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete user sessions: %w", result.Error)
	}
	if result.RowsAffected > 0 {
		log.Printf("Deleted %d sessions for user %s (password reset)", result.RowsAffected, userID)
	}
	return nil
}

// LogSecurityEvent logs security-related events
func LogSecurityEvent(eventType, userID, details string) {
	log.Printf("[SECURITY] %s | User: %s | Details: %s", eventType, userID, details)
}
