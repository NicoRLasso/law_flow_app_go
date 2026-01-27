package services

import (
	"law_flow_app_go/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAvailabilityTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&models.Availability{}, &models.BlockedDate{}, &models.Appointment{}, &models.User{})
	return db
}

func TestCreateDefaultAvailability(t *testing.T) {
	db := setupAvailabilityTestDB()
	lawyerID := "lawyer-1"

	err := CreateDefaultAvailability(db, lawyerID)
	assert.NoError(t, err)

	slots, err := GetLawyerAvailability(db, lawyerID)
	assert.NoError(t, err)
	// 5 days * 2 slots per day = 10 slots
	assert.Len(t, slots, 10)
}

func TestIsTimeSlotAvailable(t *testing.T) {
	db := setupAvailabilityTestDB()
	lawyerID := "lawyer-time"

	// 1. Setup Availability: Monday 09:00 - 12:00
	db.Create(&models.Availability{
		LawyerID:  lawyerID,
		DayOfWeek: 1, // Monday
		StartTime: "09:00",
		EndTime:   "12:00",
		IsActive:  true,
	})

	// Helper to create time on next Monday
	nextMonday := time.Now()
	for nextMonday.Weekday() != time.Monday {
		nextMonday = nextMonday.Add(24 * time.Hour)
	}
	baseDate := time.Date(nextMonday.Year(), nextMonday.Month(), nextMonday.Day(), 0, 0, 0, 0, nextMonday.Location())

	// Test Case A: Valid Slot (10:00 - 11:00)
	startA := baseDate.Add(10 * time.Hour)
	endA := baseDate.Add(11 * time.Hour)
	avail, err := IsTimeSlotAvailable(db, lawyerID, startA, endA)
	assert.NoError(t, err)
	assert.True(t, avail, "Slot should be available")

	// Test Case B: Outside valid hours (13:00 - 14:00) - No slot
	startB := baseDate.Add(13 * time.Hour)
	endB := baseDate.Add(14 * time.Hour)
	avail, err = IsTimeSlotAvailable(db, lawyerID, startB, endB)
	assert.NoError(t, err)
	assert.False(t, avail, "Slot outside hours should unavailable")

	// Test Case C: Blocked Date Conflict
	// Block 10:30 - 11:30
	db.Create(&models.BlockedDate{
		LawyerID: lawyerID,
		StartAt:  baseDate.Add(10 * time.Hour).Add(30 * time.Minute),
		EndAt:    baseDate.Add(11 * time.Hour).Add(30 * time.Minute),
	})

	avail, err = IsTimeSlotAvailable(db, lawyerID, startA, endA) // 10:00 - 11:00 overlaps with 10:30-11:30
	assert.NoError(t, err)
	assert.False(t, avail, "Slot overlapping blocked date should be unavailable")

	// Test Case D: Existing Appointment Conflict
	// Avail 09:00 - 10:00 is technically free of blocks, let's add appointment
	startD := baseDate.Add(9 * time.Hour)
	endD := baseDate.Add(10 * time.Hour)

	db.Create(&models.Appointment{
		LawyerID:  lawyerID,
		StartTime: startD,
		EndTime:   endD,
		Status:    models.AppointmentStatusScheduled,
	})

	avail, err = IsTimeSlotAvailable(db, lawyerID, startD, endD)
	assert.NoError(t, err)
	assert.False(t, avail, "Slot overlapping appointment should be unavailable")
}

func TestCheckAvailabilityOverlap(t *testing.T) {
	db := setupAvailabilityTestDB()
	lawyerID := "lawyer-overlap"

	// Slot: Mon 09:00 - 12:00
	db.Create(&models.Availability{
		LawyerID:  lawyerID,
		DayOfWeek: 1,
		StartTime: "09:00",
		EndTime:   "12:00",
		IsActive:  true,
	})

	// Overlap inside
	overlap, err := CheckAvailabilityOverlap(db, lawyerID, 1, "10:00", "11:00", "")
	assert.NoError(t, err)
	assert.True(t, overlap)

	// No Overlap
	overlap, err = CheckAvailabilityOverlap(db, lawyerID, 1, "13:00", "14:00", "")
	assert.NoError(t, err)
	assert.False(t, overlap)
}

func TestCheckBlockedDateOverlap(t *testing.T) {
	db := setupAvailabilityTestDB()
	lawyerID := "lawyer-block"

	now := time.Now()
	// Block: Today 10:00 - 12:00
	db.Create(&models.BlockedDate{
		LawyerID: lawyerID,
		StartAt:  now.Add(10 * time.Hour),
		EndAt:    now.Add(12 * time.Hour),
	})

	// Overlap
	overlap, err := CheckBlockedDateOverlap(db, lawyerID, now.Add(11*time.Hour), now.Add(13*time.Hour), "")
	assert.NoError(t, err)
	assert.True(t, overlap)

	// No Overlap
	overlap, err = CheckBlockedDateOverlap(db, lawyerID, now.Add(14*time.Hour), now.Add(15*time.Hour), "")
	assert.NoError(t, err)
	assert.False(t, overlap)
}

func TestGetBlockedDates(t *testing.T) {
	db := setupAvailabilityTestDB()
	lawyerID := "lawyer-get-blocks"

	base := time.Now().Truncate(24 * time.Hour)

	db.Create(&models.BlockedDate{
		LawyerID: lawyerID,
		StartAt:  base.Add(24 * time.Hour), // Tomorrow
		EndAt:    base.Add(25 * time.Hour),
	})
	db.Create(&models.BlockedDate{
		LawyerID: lawyerID,
		StartAt:  base.Add(48 * time.Hour), // Day after tomorrow
		EndAt:    base.Add(49 * time.Hour),
	})

	// Get all future
	dates, err := GetAllBlockedDates(db, lawyerID)
	assert.NoError(t, err)
	assert.Len(t, dates, 2)

	// Get range (tomorrow only)
	dates, err = GetBlockedDates(db, lawyerID, base.Add(23*time.Hour), base.Add(26*time.Hour))
	assert.NoError(t, err)
	assert.Len(t, dates, 1)
}
