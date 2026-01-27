package services

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"law_flow_app_go/models"
	"log"
	"time"

	"gorm.io/gorm"
)

const (
	// ResetTokenLength is the length of the reset token in bytes
	ResetTokenLength = 32
	// ResetTokenExpiration is how long a reset token is valid
	ResetTokenExpiration = 24 * time.Hour
)

// GenerateResetToken creates a password reset token for a user
func GenerateResetToken(db *gorm.DB, userEmail string) (*models.PasswordResetToken, error) {
	// Find user by email
	var user models.User
	if err := db.Where("email = ?", userEmail).First(&user).Error; err != nil {
		// Don't reveal if email exists or not (security best practice)
		// Return nil without error to prevent email enumeration
		log.Printf("Password reset requested for non-existent email: %s", userEmail)
		return nil, nil
	}

	// Check if user is active
	if !user.IsActive {
		log.Printf("Password reset requested for inactive user: %s", userEmail)
		return nil, nil
	}

	// Delete any existing tokens for this user
	db.Where("user_id = ?", user.ID).Delete(&models.PasswordResetToken{})

	// Generate cryptographically secure random token
	tokenBytes := make([]byte, ResetTokenLength)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random token: %v", err)
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Create reset token record
	resetToken := &models.PasswordResetToken{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(ResetTokenExpiration),
	}

	if err := db.Create(resetToken).Error; err != nil {
		return nil, fmt.Errorf("failed to create reset token: %v", err)
	}

	// Log security event
	LogSecurityEvent("PASSWORD_RESET_REQUESTED", user.ID, fmt.Sprintf("Password reset requested for email: %s", userEmail))

	return resetToken, nil
}

// ValidateResetToken validates a password reset token and returns the associated user
func ValidateResetToken(db *gorm.DB, token string) (*models.User, error) {
	var resetToken models.PasswordResetToken

	// Find token with user preloaded
	if err := db.Preload("User").Where("token = ?", token).First(&resetToken).Error; err != nil {
		return nil, fmt.Errorf("invalid or expired token")
	}

	// Check if token is expired
	if resetToken.IsExpired() {
		// Delete expired token
		db.Delete(&resetToken)
		return nil, fmt.Errorf("token has expired")
	}

	// Check if user is still active
	if resetToken.User == nil || !resetToken.User.IsActive {
		return nil, fmt.Errorf("user account is not active")
	}

	return resetToken.User, nil
}

// ResetPassword resets a user's password using a valid token
func ResetPassword(db *gorm.DB, token string, newPassword string) error {
	// Validate password strength using comprehensive policy
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}

	// Validate token and get user
	user, err := ValidateResetToken(db, token)
	if err != nil {
		LogSecurityEvent("PASSWORD_RESET_FAILED", "", fmt.Sprintf("Failed password reset attempt with token: %s", token[:10]))
		return err
	}

	// Hash new password
	hashedPassword, err := HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %v", err)
	}

	// Start transaction
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update user password
	if err := tx.Model(&user).Update("password", hashedPassword).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update password: %v", err)
	}

	// Delete the used token
	if err := tx.Where("token = ?", token).Delete(&models.PasswordResetToken{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete token: %v", err)
	}

	// Invalidate all user sessions (force re-login on all devices)
	if err := DeleteAllUserSessions(tx, user.ID); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to invalidate sessions: %v", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	// Log security event
	LogSecurityEvent("PASSWORD_RESET_COMPLETED", user.ID, "Password successfully reset")

	return nil
}

// CleanupExpiredTokens deletes all expired password reset tokens
func CleanupExpiredTokens(db *gorm.DB) error {
	result := db.Where("expires_at < ?", time.Now()).Delete(&models.PasswordResetToken{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected > 0 {
		log.Printf("Cleaned up %d expired password reset tokens", result.RowsAffected)
	}

	return nil
}
