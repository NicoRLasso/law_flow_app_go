package services

import (
	"law_flow_app_go/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SetupAppointmentTestDB initializes a fresh in-memory DB for these tests
func SetupAppointmentTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Migrate schemas
	err = db.AutoMigrate(&models.AppointmentType{})
	assert.NoError(t, err)

	return db
}

func TestGetAppointmentTypes(t *testing.T) {
	db := SetupAppointmentTestDB(t)
	firmID := "firm-1"

	// Seed
	t1 := models.AppointmentType{FirmID: firmID, Name: "B", Order: 2, IsActive: true}
	t2 := models.AppointmentType{FirmID: firmID, Name: "A", Order: 1, IsActive: true}
	t3 := models.AppointmentType{FirmID: "other-firm", Name: "C", Order: 1, IsActive: true}
	db.Create(&t1)
	db.Create(&t2)
	db.Create(&t3)

	// Test
	types, err := GetAppointmentTypes(db, firmID)
	assert.NoError(t, err)
	assert.Len(t, types, 2)
	assert.Equal(t, "A", types[0].Name) // Ordered by Order asc
	assert.Equal(t, "B", types[1].Name)
}

func TestGetActiveAppointmentTypes(t *testing.T) {
	db := SetupAppointmentTestDB(t)
	firmID := "firm-1"

	// Seed
	active := models.AppointmentType{FirmID: firmID, Name: "Active", IsActive: true}
	inactive := models.AppointmentType{FirmID: firmID, Name: "Inactive", IsActive: false}
	db.Create(&active)
	// Create then Update to ensure IsActive=false is persisted (GORM zero-value check)
	db.Create(&inactive)
	db.Model(&inactive).Update("IsActive", false)

	// Test
	types, err := GetActiveAppointmentTypes(db, firmID)
	assert.NoError(t, err)
	assert.Len(t, types, 1)
	assert.Equal(t, "Active", types[0].Name)
}

func TestCreateAppointmentType(t *testing.T) {
	db := SetupAppointmentTestDB(t)
	firmID := "firm-1"

	aptType := &models.AppointmentType{
		FirmID:          firmID,
		Name:            "New Type",
		DurationMinutes: 30,
	}

	err := CreateAppointmentType(db, aptType)
	assert.NoError(t, err)
	assert.NotEmpty(t, aptType.ID)
}

func TestUpdateAppointmentType(t *testing.T) {
	db := SetupAppointmentTestDB(t)

	aptType := models.AppointmentType{Name: "Old Name", IsActive: true}
	db.Create(&aptType)

	updates := map[string]interface{}{
		"name": "New Name",
	}

	err := UpdateAppointmentType(db, aptType.ID, updates)
	assert.NoError(t, err)

	var updated models.AppointmentType
	db.First(&updated, "id = ?", aptType.ID)
	assert.Equal(t, "New Name", updated.Name)
}

func TestDeleteAppointmentType(t *testing.T) {
	db := SetupAppointmentTestDB(t)

	aptType := models.AppointmentType{Name: "To Delete", IsActive: true}
	db.Create(&aptType)

	err := DeleteAppointmentType(db, aptType.ID)
	assert.NoError(t, err)

	var count int64
	db.Model(&models.AppointmentType{}).Where("id = ?", aptType.ID).Count(&count)
	assert.Equal(t, int64(0), count) // Should be soft deleted (hidden from default query)

	// Verify GORM default scope hides it
	err = db.First(&models.AppointmentType{}, "id = ?", aptType.ID).Error
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestEnsureDefaultAppointmentTypes(t *testing.T) {
	db := SetupAppointmentTestDB(t)
	firmID := "firm-1"

	// 1. Ensure creation when empty
	err := EnsureDefaultAppointmentTypes(db, firmID)
	assert.NoError(t, err)

	var count int64
	db.Model(&models.AppointmentType{}).Where("firm_id = ?", firmID).Count(&count)
	assert.Greater(t, count, int64(0))

	// 2. Ensure no duplicates if exists
	beforeCount := count
	err = EnsureDefaultAppointmentTypes(db, firmID)
	assert.NoError(t, err)

	db.Model(&models.AppointmentType{}).Where("firm_id = ?", firmID).Count(&count)
	assert.Equal(t, beforeCount, count)
}
