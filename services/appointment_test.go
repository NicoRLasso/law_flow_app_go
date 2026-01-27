package services

import (
	"law_flow_app_go/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB is reused from other tests if in the same package (services),
// but since tests are compiled per package, we can define a helper if not already exported.
// Assuming we are in `package services`, we can share `setupTestDB` if it was defined in another `_test.go` file in the same package.
// If it was defined in `addon_service_test.go`, it's available here.
// I'll assume `setupTestDB` is available or I will redefine a local version `setupAppointmentTestDB` to avoid conflicts if it's not.

func setupAppointmentTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(
		&models.User{},
		&models.Firm{},
		&models.Appointment{},
		&models.AppointmentType{},
		&models.Availability{},
		&models.BlockedDate{},
		&models.Case{},
	)
	return db
}

func TestAppointment_Create_Conflict(t *testing.T) {
	db := setupAppointmentTestDB()

	// Setup data
	firmID := "firm-1"
	lawyerID := "lawyer-1"
	clientID := "client-1"

	// Create lawyer with availability
	// Monday 09:00 - 17:00
	db.Create(&models.Availability{
		LawyerID:  lawyerID,
		DayOfWeek: 1, // Monday
		StartTime: "09:00",
		EndTime:   "17:00",
		IsActive:  true,
	})

	// Existing appointment: Monday 10:00 - 11:00
	existingStart := time.Date(2023, 10, 2, 10, 0, 0, 0, time.UTC) // Oct 2 2023 is Monday
	existingEnd := existingStart.Add(1 * time.Hour)

	db.Create(&models.Appointment{
		ID:        "apt-1",
		FirmID:    firmID,
		LawyerID:  lawyerID,
		ClientID:  &clientID,
		StartTime: existingStart,
		EndTime:   existingEnd,
		Status:    models.AppointmentStatusScheduled,
	})

	// Test Case 1: Overlapping appointment (10:30 - 11:30) -> Should fail
	newApt := &models.Appointment{
		FirmID:    firmID,
		LawyerID:  lawyerID,
		ClientID:  &clientID,
		StartTime: existingStart.Add(30 * time.Minute),
		EndTime:   existingEnd.Add(30 * time.Minute),
	}

	err := CreateAppointment(db, newApt)
	assert.Error(t, err)
	assert.Equal(t, "appointment time conflicts with an existing appointment", err.Error())

	// Test Case 2: Non-overlapping appointment (11:00 - 12:00) -> Should succeed
	// Note: 11:00 is exactly when the previous one ends.
	validApt := &models.Appointment{
		ID:        "apt-2",
		FirmID:    firmID,
		LawyerID:  lawyerID,
		ClientID:  &clientID,
		StartTime: existingEnd,
		EndTime:   existingEnd.Add(1 * time.Hour),
		Status:    models.AppointmentStatusScheduled,
	}
	err = CreateAppointment(db, validApt)
	assert.NoError(t, err)
}

func TestAppointment_Availability_Check(t *testing.T) {
	db := setupAppointmentTestDB()
	lawyerID := "lawyer-1"

	// Availability: Monday 09:00 - 12:00
	db.Create(&models.Availability{
		LawyerID:  lawyerID,
		DayOfWeek: 1, // Monday
		StartTime: "09:00",
		EndTime:   "12:00",
		IsActive:  true,
	})

	// Test Case 1: Within availability (Monday 10:00 - 11:00) -> OK
	monday := time.Date(2023, 10, 2, 10, 0, 0, 0, time.UTC)
	available, err := IsTimeSlotAvailable(db, lawyerID, monday, monday.Add(1*time.Hour))
	assert.NoError(t, err)
	assert.True(t, available)

	// Test Case 2: Outside availability (Monday 13:00 - 14:00) -> Fail
	afternoon := time.Date(2023, 10, 2, 13, 0, 0, 0, time.UTC)
	available, err = IsTimeSlotAvailable(db, lawyerID, afternoon, afternoon.Add(1*time.Hour))
	assert.NoError(t, err)
	assert.False(t, available)

	// Test Case 3: Wrong Day (Tuesday) -> Fail
	tuesday := time.Date(2023, 10, 3, 10, 0, 0, 0, time.UTC)
	available, err = IsTimeSlotAvailable(db, lawyerID, tuesday, tuesday.Add(1*time.Hour))
	assert.NoError(t, err)
	assert.False(t, available)
}

func TestAppointment_Reschedule_Conflict(t *testing.T) {
	db := setupAppointmentTestDB()
	firmID := "firm-1"
	lawyerID := "lawyer-1"
	clientID := "client-1"

	// Slot A: 10:00 - 11:00 (Existing)
	startA := time.Date(2023, 10, 2, 10, 0, 0, 0, time.UTC)
	db.Create(&models.Appointment{
		ID:        "apt-1",
		FirmID:    firmID,
		LawyerID:  lawyerID,
		ClientID:  &clientID,
		StartTime: startA,
		EndTime:   startA.Add(1 * time.Hour),
		Status:    models.AppointmentStatusScheduled,
	})

	// Slot B: 12:00 - 13:00 (Existing)
	startB := time.Date(2023, 10, 2, 12, 0, 0, 0, time.UTC)
	db.Create(&models.Appointment{
		ID:        "apt-2",
		FirmID:    firmID,
		LawyerID:  lawyerID,
		ClientID:  &clientID,
		StartTime: startB,
		EndTime:   startB.Add(1 * time.Hour),
		Status:    models.AppointmentStatusScheduled,
	})

	// Test Case: specific apt-1 cannot be rescheduled to overlap with apt-2
	// Try rescheduling apt-1 to 11:30 - 12:30 (overlaps with B which starts at 12:00)
	err := RescheduleAppointment(db, "apt-1", startB.Add(-30*time.Minute), startB.Add(30*time.Minute))
	assert.Error(t, err)
	assert.Equal(t, "new time conflicts with an existing appointment", err.Error())

	// Test Case: apt-1 CAN be rescheduled to itself (no change) or to empty slot
	// Reschedule to 09:00 - 10:00
	err = RescheduleAppointment(db, "apt-1", startA.Add(-1*time.Hour), startA)
	assert.NoError(t, err)
}
