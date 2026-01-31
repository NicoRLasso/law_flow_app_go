package services

import (
	"law_flow_app_go/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAvailabilityTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(
		&models.Firm{},
		&models.User{},
		&models.Availability{},
		&models.BlockedDate{},
		&models.Appointment{},
	)
	assert.NoError(t, err)

	return db
}

func TestAvailabilityService(t *testing.T) {
	db := setupAvailabilityTestDB(t)
	firmID := "firm-avail"
	lawyerID := "lawyer-avail"
	db.Create(&models.Firm{ID: firmID})
	db.Create(&models.User{ID: lawyerID, FirmID: &firmID})

	t.Run("CreateDefaultAvailability", func(t *testing.T) {
		err := CreateDefaultAvailability(db, lawyerID)
		assert.NoError(t, err)

		slots, err := GetLawyerAvailability(db, lawyerID)
		assert.NoError(t, err)
		assert.NotEmpty(t, slots)
	})

	t.Run("IsTimeSlotAvailable", func(t *testing.T) {
		// Monday June 1st 2026 is a Monday (Weekday 1)
		monday := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
		sunday := time.Date(2026, 5, 31, 9, 0, 0, 0, time.UTC)

		available, err := IsTimeSlotAvailable(db, lawyerID, monday, monday.Add(time.Hour))
		assert.NoError(t, err)
		assert.True(t, available)

		notAvailable, err := IsTimeSlotAvailable(db, lawyerID, sunday, sunday.Add(time.Hour))
		assert.NoError(t, err)
		assert.False(t, notAvailable)
	})

	t.Run("Blocked Dates", func(t *testing.T) {
		now := time.Now()
		db.Create(&models.BlockedDate{
			LawyerID: lawyerID,
			StartAt:  now,
			EndAt:    now.Add(time.Hour * 24),
			Reason:   "Vacation",
		})

		blocked, err := GetBlockedDates(db, lawyerID, now, now.Add(time.Hour))
		assert.NoError(t, err)
		assert.Len(t, blocked, 1)

		allBlocked, err := GetAllBlockedDates(db, lawyerID)
		assert.NoError(t, err)
		assert.Len(t, allBlocked, 1)
	})

	t.Run("Availability Overlap", func(t *testing.T) {
		// Existing slot: 09:00 - 10:00 (from default)
		overlap, err := CheckAvailabilityOverlap(db, lawyerID, 1, "09:30", "10:30", "")
		assert.NoError(t, err)
		assert.True(t, overlap)

		noOverlap, err := CheckAvailabilityOverlap(db, lawyerID, 1, "12:00", "13:00", "")
		assert.NoError(t, err)
		assert.False(t, noOverlap)
	})
}
