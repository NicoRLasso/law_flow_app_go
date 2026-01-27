package jobs

import (
	"law_flow_app_go/config"
	"law_flow_app_go/models"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupRemindersTestDB() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	db.AutoMigrate(&models.Firm{}, &models.User{}, &models.Appointment{}, &models.AppointmentType{})
	return db
}

func TestSendAppointmentReminders(t *testing.T) {
	db := setupRemindersTestDB()
	cfg := &config.Config{
		AppURL:        "http://test.com",
		EmailTestMode: true, // This ensures SendEmail logs to console instead of trying to send
	}

	firm := models.Firm{ID: uuid.New().String(), Name: "Test Firm"}
	db.Create(&firm)

	lawyer := models.User{ID: uuid.New().String(), Name: "Jane Lawyer", Role: "lawyer", FirmID: &firm.ID}
	db.Create(&lawyer)

	client := models.User{ID: uuid.New().String(), Name: "John Client", Role: "client", FirmID: &firm.ID, Language: "en"}
	db.Create(&client)

	now := time.Now().UTC()

	// 1. Appointment tomorrow (Should be reminded)
	apt1 := models.Appointment{
		ID:           uuid.New().String(),
		FirmID:       firm.ID,
		LawyerID:     lawyer.ID,
		ClientID:     &client.ID,
		ClientEmail:  "john@client.com",
		ClientName:   "John Client",
		StartTime:    now.Add(25 * time.Hour),
		EndTime:      now.Add(26 * time.Hour),
		Status:       models.AppointmentStatusScheduled,
		BookingToken: "token1",
	}
	db.Create(&apt1)

	// 2. Appointment already reminded
	apt2 := apt1
	apt2.ID = uuid.New().String()
	apt2.BookingToken = "token2"
	remindedAt := now.Add(-1 * time.Hour)
	apt2.ReminderSentAt = &remindedAt
	db.Create(&apt2)

	// 3. Appointment too far in future (3 days)
	apt3 := apt1
	apt3.ID = uuid.New().String()
	apt3.BookingToken = "token3"
	apt3.StartTime = now.Add(72 * time.Hour)
	db.Create(&apt3)

	// 4. Cancelled appointment
	apt4 := apt1
	apt4.ID = uuid.New().String()
	apt4.BookingToken = "token4"
	apt4.Status = models.AppointmentStatusCancelled
	db.Create(&apt4)

	SendAppointmentReminders(db, cfg)

	// Verify ReminderSentAt was updated for apt1
	var updatedApt1 models.Appointment
	db.First(&updatedApt1, "id = ?", apt1.ID)
	assert.NotNil(t, updatedApt1.ReminderSentAt)
	assert.True(t, updatedApt1.ReminderSentAt.After(now))

	// Verify others were NOT updated (beyond their initial state)
	var updatedApt2 models.Appointment
	db.First(&updatedApt2, "id = ?", apt2.ID)
	assert.True(t, updatedApt2.ReminderSentAt.Equal(remindedAt))

	var updatedApt3 models.Appointment
	db.First(&updatedApt3, "id = ?", apt3.ID)
	assert.Nil(t, updatedApt3.ReminderSentAt)

	var updatedApt4 models.Appointment
	db.First(&updatedApt4, "id = ?", apt4.ID)
	assert.Nil(t, updatedApt4.ReminderSentAt)
}
