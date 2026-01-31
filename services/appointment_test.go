package services

import (
	"law_flow_app_go/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAppointmentTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(
		&models.Firm{},
		&models.User{},
		&models.Availability{},
		&models.BlockedDate{},
		&models.Appointment{},
		&models.AppointmentType{},
	)
	assert.NoError(t, err)

	return db
}

func TestAppointmentService(t *testing.T) {
	db := setupAppointmentTestDB(t)
	firmID := "firm-apt"
	lawyerID := "lawyer-apt"
	clientID := "client-apt"
	db.Create(&models.Firm{ID: firmID})
	db.Create(&models.User{ID: lawyerID, FirmID: &firmID, Role: "lawyer"})
	db.Create(&models.User{ID: clientID, FirmID: &firmID, Role: "client"})

	// Setup availability
	CreateDefaultAvailability(db, lawyerID)

	t.Run("CreateAppointment", func(t *testing.T) {
		// Monday June 1st 2026, 10:00-11:00
		start := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
		end := start.Add(time.Hour)

		apt := &models.Appointment{
			FirmID:      firmID,
			LawyerID:    lawyerID,
			ClientID:    &clientID,
			ClientName:  "Test Client",
			ClientEmail: "client@test.com",
			StartTime:   start,
			EndTime:     end,
			Status:      models.AppointmentStatusScheduled,
		}

		err := CreateAppointment(db, apt)
		assert.NoError(t, err)

		// Check conflict
		conflictApt := &models.Appointment{
			FirmID:      firmID,
			LawyerID:    lawyerID,
			ClientID:    &clientID,
			ClientName:  "Conflict",
			ClientEmail: "conflict@test.com",
			StartTime:   start.Add(30 * time.Minute),
			EndTime:     end.Add(30 * time.Minute),
		}
		err = CreateAppointment(db, conflictApt)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "conflicts")
	})

	t.Run("GetAvailableSlots", func(t *testing.T) {
		date := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
		slots, err := GetAvailableSlots(db, lawyerID, date, 60, "UTC")
		assert.NoError(t, err)
		assert.NotEmpty(t, slots)

		// The 10:00-11:00 slot should be missing (taken by first test)
		for _, s := range slots {
			assert.NotEqual(t, "10:00", s.StartTime.Format("15:04"))
		}
	})

	t.Run("Cancel/Reschedule", func(t *testing.T) {
		var apt models.Appointment
		db.First(&apt)

		// Reschedule
		newStart := time.Date(2026, 6, 1, 14, 0, 0, 0, time.UTC)
		newEnd := newStart.Add(time.Hour)
		err := RescheduleAppointment(db, apt.ID, newStart, newEnd)
		assert.NoError(t, err)

		// Cancel
		err = CancelAppointment(db, apt.ID)
		assert.NoError(t, err)

		db.First(&apt, "id = ?", apt.ID)
		assert.Equal(t, models.AppointmentStatusCancelled, apt.Status)
	})
}

func TestAppointmentTypeService(t *testing.T) {
	db := setupAppointmentTestDB(t)
	firmID := "firm-apt-type"
	db.Create(&models.Firm{ID: firmID})

	t.Run("CRUD", func(t *testing.T) {
		at := &models.AppointmentType{Name: "Consultation", FirmID: firmID, IsActive: true}
		err := CreateAppointmentType(db, at)
		assert.NoError(t, err)

		types, err := GetAppointmentTypes(db, firmID)
		assert.NoError(t, err)
		assert.Len(t, types, 1)

		err = UpdateAppointmentType(db, at.ID, map[string]interface{}{"is_active": false})
		assert.NoError(t, err)

		active, _ := GetActiveAppointmentTypes(db, firmID)
		assert.Empty(t, active)

		err = DeleteAppointmentType(db, at.ID)
		assert.NoError(t, err)
	})
}
