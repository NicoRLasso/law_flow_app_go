package services

import (
	"errors"
	"law_flow_app_go/db"
	"law_flow_app_go/models"
	"time"
)

// CreateAppointment creates a new appointment after checking for conflicts
func CreateAppointment(apt *models.Appointment) error {
	// Check for conflicts before creating
	hasConflict, err := CheckAppointmentConflict(apt.LawyerID, apt.StartTime, apt.EndTime, "")
	if err != nil {
		return err
	}
	if hasConflict {
		return errors.New("appointment time conflicts with an existing appointment")
	}

	// Verify the slot is within availability and not blocked
	isAvailable, err := IsTimeSlotAvailable(apt.LawyerID, apt.StartTime, apt.EndTime)
	if err != nil {
		return err
	}
	if !isAvailable {
		return errors.New("selected time is not within lawyer's availability")
	}

	return db.DB.Create(apt).Error
}

// GetAppointmentByID fetches a single appointment with relationships
func GetAppointmentByID(id string) (*models.Appointment, error) {
	var apt models.Appointment
	err := db.DB.Preload("Lawyer").Preload("Client").Preload("Case").Preload("AppointmentType").
		First(&apt, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &apt, nil
}

// GetLawyerAppointments fetches appointments for a lawyer within a date range
func GetLawyerAppointments(lawyerID string, startDate, endDate time.Time) ([]models.Appointment, error) {
	var appointments []models.Appointment
	err := db.DB.Preload("Client").
		Where("lawyer_id = ? AND start_time >= ? AND end_time <= ?", lawyerID, startDate, endDate).
		Where("status NOT IN (?)", []string{models.AppointmentStatusCancelled}).
		Order("start_time asc").
		Find(&appointments).Error
	return appointments, err
}

// GetFirmAppointments fetches all appointments for a firm within a date range
func GetFirmAppointments(firmID string, startDate, endDate time.Time) ([]models.Appointment, error) {
	var appointments []models.Appointment
	err := db.DB.Preload("Lawyer").Preload("Client").
		Where("firm_id = ? AND start_time >= ? AND end_time <= ?", firmID, startDate, endDate).
		Order("start_time asc").
		Find(&appointments).Error
	return appointments, err
}

// GetClientAppointments fetches appointments for a client
func GetClientAppointments(clientID string) ([]models.Appointment, error) {
	var appointments []models.Appointment
	err := db.DB.Preload("Lawyer").
		Where("client_id = ?", clientID).
		Order("start_time desc").
		Find(&appointments).Error
	return appointments, err
}

// UpdateAppointmentStatus updates the status of an appointment
func UpdateAppointmentStatus(id, status string) error {
	if !models.IsValidAppointmentStatus(status) {
		return errors.New("invalid appointment status")
	}
	return db.DB.Model(&models.Appointment{}).Where("id = ?", id).Update("status", status).Error
}

// CancelAppointment cancels an appointment
func CancelAppointment(id string) error {
	apt, err := GetAppointmentByID(id)
	if err != nil {
		return err
	}
	if !apt.IsCancellable() {
		return errors.New("appointment cannot be cancelled")
	}
	return UpdateAppointmentStatus(id, models.AppointmentStatusCancelled)
}

// RescheduleAppointment reschedules an appointment to a new time
func RescheduleAppointment(id string, newStart, newEnd time.Time) error {
	apt, err := GetAppointmentByID(id)
	if err != nil {
		return err
	}
	if !apt.IsEditable() {
		return errors.New("appointment cannot be rescheduled")
	}

	// Check for conflicts (excluding this appointment)
	hasConflict, err := CheckAppointmentConflict(apt.LawyerID, newStart, newEnd, id)
	if err != nil {
		return err
	}
	if hasConflict {
		return errors.New("new time conflicts with an existing appointment")
	}

	return db.DB.Model(&models.Appointment{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"start_time": newStart,
			"end_time":   newEnd,
		}).Error
}

// CheckAppointmentConflict checks if a time slot conflicts with existing appointments
func CheckAppointmentConflict(lawyerID string, startTime, endTime time.Time, excludeID string) (bool, error) {
	var count int64
	query := db.DB.Model(&models.Appointment{}).
		Where("lawyer_id = ?", lawyerID).
		Where("status NOT IN (?)", []string{models.AppointmentStatusCancelled, models.AppointmentStatusNoShow}).
		Where("start_time < ? AND end_time > ?", endTime, startTime) // Overlap check

	if excludeID != "" {
		query = query.Where("id != ?", excludeID)
	}

	err := query.Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetAvailableSlots generates available time slots for a lawyer on a specific date
func GetAvailableSlots(lawyerID string, date time.Time, slotDurationMinutes int, firmTimezone string) ([]models.TimeSlot, error) {
	// Load timezone
	loc, err := time.LoadLocation(firmTimezone)
	if err != nil {
		loc = time.UTC
	}

	// Get the start and end of the day in the firm's timezone
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, loc)
	dayEnd := dayStart.Add(24 * time.Hour)

	// Get the day of week (0=Sunday, 6=Saturday)
	dayOfWeek := int(dayStart.Weekday())

	// 1. Get availability slots for this day
	var availabilities []models.Availability
	err = db.DB.Where("lawyer_id = ? AND day_of_week = ? AND is_active = ?", lawyerID, dayOfWeek, true).
		Order("start_time").
		Find(&availabilities).Error
	if err != nil {
		return nil, err
	}

	if len(availabilities) == 0 {
		return []models.TimeSlot{}, nil // No availability for this day
	}

	// 2. Get blocked dates overlapping with this day
	blockedDates, err := GetBlockedDates(lawyerID, dayStart, dayEnd)
	if err != nil {
		return nil, err
	}

	// 3. Get existing appointments for this day (excluding cancelled)
	existingAppointments, err := GetLawyerAppointments(lawyerID, dayStart, dayEnd)
	if err != nil {
		return nil, err
	}

	// 4. Generate all possible slots within availability windows
	var availableSlots []models.TimeSlot
	slotDuration := time.Duration(slotDurationMinutes) * time.Minute

	for _, avail := range availabilities {
		// Parse availability times (format "HH:MM")
		availStart, _ := time.Parse("15:04", avail.StartTime)
		availEnd, _ := time.Parse("15:04", avail.EndTime)

		// Create full datetime for this availability window
		windowStart := time.Date(dayStart.Year(), dayStart.Month(), dayStart.Day(),
			availStart.Hour(), availStart.Minute(), 0, 0, loc)
		windowEnd := time.Date(dayStart.Year(), dayStart.Month(), dayStart.Day(),
			availEnd.Hour(), availEnd.Minute(), 0, 0, loc)

		// Generate slots within this window
		for slotStart := windowStart; slotStart.Add(slotDuration).Before(windowEnd) || slotStart.Add(slotDuration).Equal(windowEnd); slotStart = slotStart.Add(slotDuration) {
			slotEnd := slotStart.Add(slotDuration)

			// Convert to UTC for comparison
			slotStartUTC := slotStart.UTC()
			slotEndUTC := slotEnd.UTC()

			// Check if slot is blocked
			isBlocked := false
			for _, blocked := range blockedDates {
				if blocked.IsBlocking(slotStartUTC, slotEndUTC) {
					isBlocked = true
					break
				}
			}
			if isBlocked {
				continue
			}

			// Check if slot conflicts with existing appointment
			hasConflict := false
			for _, apt := range existingAppointments {
				// Overlap check: (StartA < EndB) AND (EndA > StartB)
				if apt.StartTime.Before(slotEndUTC) && apt.EndTime.After(slotStartUTC) {
					hasConflict = true
					break
				}
			}
			if hasConflict {
				continue
			}

			// Slot is available - store in UTC
			availableSlots = append(availableSlots, models.TimeSlot{
				StartTime: slotStartUTC,
				EndTime:   slotEndUTC,
			})
		}
	}

	return availableSlots, nil
}

// GetFirmLawyersWithAvailability fetches lawyers from a firm who have availability set up
func GetFirmLawyersWithAvailability(firmID string) ([]models.User, error) {
	var lawyers []models.User

	// Get lawyers with role 'lawyer' or 'admin' who have at least one active availability slot
	err := db.DB.
		Where("firm_id = ? AND role IN (?) AND is_active = ?", firmID, []string{"lawyer", "admin"}, true).
		Where("id IN (SELECT DISTINCT lawyer_id FROM availabilities WHERE is_active = ?))", true).
		Find(&lawyers).Error

	return lawyers, err
}
