package services

import (
	"fmt"
	"law_flow_app_go/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AnonymizeUser replaces a user's PII with anonymized data to comply with GDPR "Right to be Forgotten"
// This is preferred over hard deletion to maintain database referential integrity for cases/history.
func AnonymizeUser(db *gorm.DB, userID string, adminID string) error {
	var user models.User
	if err := db.First(&user, "id = ?", userID).Error; err != nil {
		return err
	}

	// 1. Create anonymized values
	// We keep the ID but scramble identifiable fields
	uniqueSuffix := uuid.New().String()[:8]
	timestamp := time.Now().UTC().Format("20060102")

	anonymizedName := fmt.Sprintf("Anonymized User %s", uniqueSuffix)
	anonymizedEmail := fmt.Sprintf("deleted_%s_%s@anonymized.invalid", timestamp, uniqueSuffix)

	// 2. Prepare updates
	updates := map[string]interface{}{
		"name":            anonymizedName,
		"email":           anonymizedEmail,
		"phone_number":    nil,
		"address":         nil,
		"document_number": nil,
		"is_active":       false,
		"password":        "", // Clear password hash
		// We could also clear roles or set to a restrictive "deleted" role if system supports it
	}

	// 3. Perform transaction
	tx := db.Begin()

	// Update user record
	if err := tx.Model(&user).Updates(updates).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to anonymize user record: %w", err)
	}

	// Soft delete the user (sets DeletedAt) so they don't show up in normal queries
	if err := tx.Delete(&user).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to soft delete user: %w", err)
	}

	// 4. Log specific GDPR audit event
	// We log this securely. The old PII is lost (as desired), but we record WHO did it.
	auditLog := models.AuditLog{
		UserID:       ptrIfNotEmpty(adminID),
		Action:       "GDPR_ANONYMIZE",
		ResourceType: "User",
		ResourceID:   userID,
		Description:  fmt.Sprintf("User anonymized by admin %s", adminID),
		// Do not store old values in audit log for GDPR compliance (or store minimal info)
		NewValues: fmt.Sprintf("{\"anonymized_email\": \"%s\"}", anonymizedEmail),
	}

	if err := tx.Create(&auditLog).Error; err != nil {
		// Non-fatal, but bad practice to fail audit
		fmt.Printf("WARNING: Failed to log GDPR audit event: %v\n", err)
	}

	return tx.Commit().Error
}
