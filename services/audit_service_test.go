package services

import (
	"encoding/json"
	"law_flow_app_go/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Reusing setupTestDB from other tests if available, but declaring here for standalone correctness
func setupAuditTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&models.AuditLog{}, &models.User{})
	return db
}

func TestLogAuditEvent(t *testing.T) {
	db := setupAuditTestDB()

	// Create a dummy user
	user := models.User{
		Name:  "Test Auditor",
		Email: "auditor@lexlegal.com",
		Role:  "admin",
	}
	db.Create(&user)

	ctx := AuditContext{
		UserID:   user.ID,
		UserName: user.Name,
		UserRole: user.Role,
		FirmID:   "firm-123",
		FirmName: "Test Firm",
	}

	oldVals := map[string]interface{}{"status": "pending"}
	newVals := map[string]interface{}{"status": "active"}

	// Determine expected action based on your model definition
	// Assuming AuditActionUpdate is a constant or type alias
	action := models.AuditActionUpdate

	LogAuditEvent(db, ctx, action, "Case", "case-123", "Case #123", "Updated status", oldVals, newVals)

	// Since LogAuditEvent is async (go func), we need to wait a bit
	time.Sleep(100 * time.Millisecond)

	var log models.AuditLog
	result := db.First(&log, "resource_id = ?", "case-123")
	assert.NoError(t, result.Error)
	assert.Equal(t, user.ID, *log.UserID)
	assert.Equal(t, "firm-123", *log.FirmID)
	assert.Equal(t, "Case", log.ResourceType)
	assert.Equal(t, "Updated status", log.Description)

	// Check JSON fields
	var savedOld, savedNew map[string]interface{}
	json.Unmarshal([]byte(log.OldValues), &savedOld)
	json.Unmarshal([]byte(log.NewValues), &savedNew)

	assert.Equal(t, "pending", savedOld["status"])
	assert.Equal(t, "active", savedNew["status"])
}

func TestLogSecurityEvent(t *testing.T) {
	db := setupAuditTestDB()

	userID := "user-security-123"

	LogSecurityEvent(db, "LOGIN_FAILED", userID, "Invalid password")

	// Wait for async
	time.Sleep(100 * time.Millisecond)

	var log models.AuditLog
	result := db.Where("action = ?", "SECURITY").First(&log)
	assert.NoError(t, result.Error)
	assert.Equal(t, userID, *log.UserID)
	assert.Equal(t, "SECURITY_EVENT", log.ResourceType)
	assert.Equal(t, "LOGIN_FAILED", log.ResourceID)
	assert.Equal(t, "Invalid password", log.Description)
}

func TestGetResourceAuditHistory(t *testing.T) {
	db := setupAuditTestDB()

	// Seed some logs
	db.Create(&models.AuditLog{
		ResourceType: "Case",
		ResourceID:   "case-ABC",
		Action:       models.AuditActionCreate,
		CreatedAt:    time.Now().Add(-2 * time.Hour),
	})
	db.Create(&models.AuditLog{
		ResourceType: "Case",
		ResourceID:   "case-ABC",
		Action:       models.AuditActionUpdate,
		CreatedAt:    time.Now().Add(-1 * time.Hour),
	})
	db.Create(&models.AuditLog{
		ResourceType: "Other",
		ResourceID:   "other-123",
		Action:       models.AuditActionCreate,
	})

	logs, err := GetResourceAuditHistory(db, "Case", "case-ABC")
	assert.NoError(t, err)
	assert.Len(t, logs, 2)
	assert.Equal(t, models.AuditActionUpdate, logs[0].Action) // Should be ordered by desc time
}

// Assuming GetFirmAuditLogs exists and takes filtering params
func TestGetFirmAuditLogs(t *testing.T) {
	db := setupAuditTestDB()

	firmID := "firm-test-1"
	otherFirmID := "firm-test-2"

	db.Create(&models.AuditLog{
		FirmID:    &firmID,
		Action:    models.AuditActionCreate,
		CreatedAt: time.Now(),
	})
	db.Create(&models.AuditLog{
		FirmID:    &otherFirmID,
		Action:    models.AuditActionCreate,
		CreatedAt: time.Now(),
	})

	filter := AuditLogFilters{}

	logs, total, err := GetFirmAuditLogs(db, firmID, filter, 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, logs, 1)
	assert.Equal(t, firmID, *logs[0].FirmID)
}
