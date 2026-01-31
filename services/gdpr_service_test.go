package services

import (
	"law_flow_app_go/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupGDPRTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&models.User{}, &models.AuditLog{}, &models.Firm{})
	return db
}

func TestAnonymizeUser(t *testing.T) {
	db := setupGDPRTestDB()
	userID := "user-to-forget"
	adminID := "admin-1"
	firmID := "firm-1"

	db.Create(&models.Firm{ID: firmID, Name: "F1"})
	db.Create(&models.User{
		ID:             userID,
		FirmID:         &firmID,
		Name:           "Secret Name",
		Email:          "secret@example.com",
		PhoneNumber:    stringPtr("123"),
		DocumentNumber: stringPtr("ABC"),
		IsActive:       true,
	})

	err := AnonymizeUser(db, userID, adminID)
	assert.NoError(t, err)

	// Verify User
	var user models.User
	// Use Unscoped to find soft-deleted user
	err = db.Unscoped().First(&user, "id = ?", userID).Error
	assert.NoError(t, err)
	assert.NotEqual(t, "Secret Name", user.Name)
	assert.NotEqual(t, "secret@example.com", user.Email)
	assert.Nil(t, user.PhoneNumber)
	assert.Nil(t, user.DocumentNumber)
	assert.False(t, user.IsActive)
	assert.NotNil(t, user.DeletedAt)

	// Verify Audit Log
	var audit models.AuditLog
	err = db.First(&audit, "resource_id = ? AND action = ?", userID, "GDPR_ANONYMIZE").Error
	assert.NoError(t, err)
	assert.Equal(t, adminID, *audit.UserID)
}
