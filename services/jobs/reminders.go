package jobs

import (
	"law_flow_app_go/config"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"log"
	"time"

	"gorm.io/gorm"
)

// SendAppointmentReminders checks for appointments tomorrow and sends reminders
func SendAppointmentReminders(database *gorm.DB, cfg *config.Config) {
	log.Println("Starting appointment reminder job...")

	// Calculate time range for appointments starting tomorrow (next 24-48 hours window)
	now := time.Now().UTC()
	tomorrowStart := now.Add(24 * time.Hour)
	tomorrowEnd := now.Add(48 * time.Hour) // Broad window to catch anything for "tomorrow"

	var appointments []models.Appointment

	// Find appointments:
	// 1. Scheduled or Confirmed
	// 2. StartTime between tomorrowStart and tomorrowEnd
	// 3. ReminderSentAt is NULL
	err := database.Preload("Lawyer").Preload("Firm").Preload("AppointmentType").Preload("Client").
		Where("status IN (?)", []string{models.AppointmentStatusScheduled, models.AppointmentStatusConfirmed}).
		Where("start_time >= ? AND start_time <= ?", tomorrowStart, tomorrowEnd).
		Where("reminder_sent_at IS NULL").
		Find(&appointments).Error

	if err != nil {
		log.Printf("Error fetching appointments for reminders: %v", err)
		return
	}

	log.Printf("Found %d appointments to remind", len(appointments))

	for _, apt := range appointments {
		// Send email
		meetingURL := ""
		if apt.MeetingURL != nil {
			meetingURL = *apt.MeetingURL
		}

		// Use booking token link for management
		manageLink := cfg.AppURL + "/appointment/" + apt.BookingToken

		// Determine language
		lang := "es"
		if apt.ClientID != nil && apt.Client != nil && apt.Client.Language != "" {
			lang = apt.Client.Language
		}

		email := services.BuildAppointmentReminderEmail(apt.ClientEmail, services.AppointmentReminderEmailData{
			ClientName: apt.ClientName,
			FirmName:   apt.Firm.Name,
			Date:       apt.StartTime.Format("Monday, January 2, 2006"),
			Time:       apt.StartTime.Format("3:04 PM"),
			Duration:   apt.Duration(),
			LawyerName: apt.Lawyer.Name,
			MeetingURL: meetingURL,
			ManageLink: manageLink,
		}, lang)

		if err := services.SendEmail(cfg, email); err != nil {
			log.Printf("Failed to send reminder for appointment %s: %v", apt.ID, err)
			continue
		}

		// Update ReminderSentAt
		now := time.Now().UTC()
		database.Model(&apt).Update("reminder_sent_at", now)
		log.Printf("Sent reminder for appointment %s", apt.ID)
	}

	log.Println("Appointment reminder job completed")
}
