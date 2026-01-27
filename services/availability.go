package services

import (
	"law_flow_app_go/models"
	"time"

	"gorm.io/gorm"
)

// Default working hours: Mon-Fri, 9:00-12:00 and 14:00-17:00
var defaultAvailabilitySlots = []struct {
	DayOfWeek int
	StartTime string
	EndTime   string
}{
	// Monday (1)
	{1, "09:00", "12:00"},
	{1, "14:00", "17:00"},
	// Tuesday (2)
	{2, "09:00", "12:00"},
	{2, "14:00", "17:00"},
	// Wednesday (3)
	{3, "09:00", "12:00"},
	{3, "14:00", "17:00"},
	// Thursday (4)
	{4, "09:00", "12:00"},
	{4, "14:00", "17:00"},
	// Friday (5)
	{5, "09:00", "12:00"},
	{5, "14:00", "17:00"},
}

// CreateDefaultAvailability creates the default availability slots for a lawyer
func CreateDefaultAvailability(db *gorm.DB, lawyerID string) error {
	for _, slot := range defaultAvailabilitySlots {
		availability := &models.Availability{
			LawyerID:  lawyerID,
			DayOfWeek: slot.DayOfWeek,
			StartTime: slot.StartTime,
			EndTime:   slot.EndTime,
			IsActive:  true,
		}
		if err := db.Create(availability).Error; err != nil {
			return err
		}
	}
	return nil
}

// GetLawyerAvailability fetches all availability slots for a lawyer
func GetLawyerAvailability(db *gorm.DB, lawyerID string) ([]models.Availability, error) {
	var slots []models.Availability
	err := db.Where("lawyer_id = ?", lawyerID).
		Order("day_of_week, start_time").
		Find(&slots).Error
	return slots, err
}

// GetAvailabilityByID fetches a single availability slot
func GetAvailabilityByID(db *gorm.DB, id string) (*models.Availability, error) {
	var slot models.Availability
	err := db.First(&slot, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &slot, nil
}

// CreateAvailabilitySlot adds a new availability slot
func CreateAvailabilitySlot(db *gorm.DB, slot *models.Availability) error {
	return db.Create(slot).Error
}

// UpdateAvailabilitySlot updates an existing availability slot
func UpdateAvailabilitySlot(db *gorm.DB, slot *models.Availability) error {
	return db.Save(slot).Error
}

// DeleteAvailabilitySlot removes an availability slot
func DeleteAvailabilitySlot(db *gorm.DB, id string) error {
	return db.Delete(&models.Availability{}, "id = ?", id).Error
}

// GetBlockedDates fetches blocked dates for a lawyer in a date range
func GetBlockedDates(db *gorm.DB, lawyerID string, startDate, endDate time.Time) ([]models.BlockedDate, error) {
	var blockedDates []models.BlockedDate
	// Find blocks that overlap with the requested window: (StartA < EndB) AND (EndA > StartB)
	err := db.Where("lawyer_id = ? AND start_at < ? AND end_at > ?", lawyerID, endDate, startDate).
		Order("start_at asc").
		Find(&blockedDates).Error
	return blockedDates, err
}

// GetAllBlockedDates fetches all blocked dates for a lawyer (future and recent past)
func GetAllBlockedDates(db *gorm.DB, lawyerID string) ([]models.BlockedDate, error) {
	var blockedDates []models.BlockedDate
	// Fetch blocks that end in the future (or today)
	err := db.Where("lawyer_id = ? AND end_at >= ?", lawyerID, time.Now().Truncate(24*time.Hour)).
		Order("start_at asc").
		Find(&blockedDates).Error
	return blockedDates, err
}

// GetBlockedDateByID fetches a single blocked date
func GetBlockedDateByID(db *gorm.DB, id string) (*models.BlockedDate, error) {
	var blockedDate models.BlockedDate
	err := db.First(&blockedDate, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &blockedDate, nil
}

// CreateBlockedDate adds a new blocked date
func CreateBlockedDate(db *gorm.DB, blockedDate *models.BlockedDate) error {
	return db.Create(blockedDate).Error
}

// DeleteBlockedDate removes a blocked date
func DeleteBlockedDate(db *gorm.DB, id string) error {
	return db.Delete(&models.BlockedDate{}, "id = ?", id).Error
}

// IsTimeSlotAvailable checks if a time slot is available for a lawyer
// It considers weekly availability, blocked dates, and existing appointments
func IsTimeSlotAvailable(db *gorm.DB, lawyerID string, checkStart, checkEnd time.Time) (bool, error) {
	// 1. Check if the time falls within regular availability
	dayOfWeek := int(checkStart.Weekday())
	startTimeStr := checkStart.Format("15:04")
	endTimeStr := checkEnd.Format("15:04")

	var count int64
	err := db.Model(&models.Availability{}).
		Where("lawyer_id = ? AND day_of_week = ? AND is_active = ? AND start_time <= ? AND end_time >= ?",
			lawyerID, dayOfWeek, true, startTimeStr, endTimeStr).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	if count == 0 {
		return false, nil // Not within regular working hours
	}

	// 2. Check for blocked dates
	var blockedDates []models.BlockedDate
	// Find any block overlapping with this specific time slot
	err = db.Where("lawyer_id = ? AND start_at < ? AND end_at > ?", lawyerID, checkEnd, checkStart).
		Find(&blockedDates).Error
	if err != nil {
		return false, err
	}

	for _, blocked := range blockedDates {
		if blocked.IsBlocking(checkStart, checkEnd) {
			return false, nil // Blocked
		}
	}

	// 3. Check for existing appointments
	var appointmentCount int64
	err = db.Model(&models.Appointment{}).
		Where("lawyer_id = ? AND status NOT IN (?, ?) AND start_time < ? AND end_time > ?",
			lawyerID, models.AppointmentStatusCancelled, models.AppointmentStatusNoShow, checkEnd, checkStart).
		Count(&appointmentCount).Error
	if err != nil {
		return false, err
	}
	if appointmentCount > 0 {
		return false, nil // Conflicting appointment exists
	}

	return true, nil
}

// CheckAvailabilityOverlap checks if a new or updated slot overlaps with existing slots
func CheckAvailabilityOverlap(db *gorm.DB, lawyerID string, dayOfWeek int, startTime, endTime string, excludeSlotID string) (bool, error) {
	var count int64
	query := db.Model(&models.Availability{}).
		Where("lawyer_id = ? AND day_of_week = ? AND is_active = ?", lawyerID, dayOfWeek, true).
		Where("((start_time < ? AND end_time > ?) OR (start_time >= ? AND start_time < ?) OR (end_time > ? AND end_time <= ?))",
			endTime, startTime, startTime, endTime, startTime, endTime)

	if excludeSlotID != "" {
		query = query.Where("id != ?", excludeSlotID)
	}

	err := query.Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// HasAvailabilitySlots checks if a lawyer has any availability slots configured
func HasAvailabilitySlots(db *gorm.DB, lawyerID string) (bool, error) {
	var count int64
	err := db.Model(&models.Availability{}).Where("lawyer_id = ?", lawyerID).Count(&count).Error
	return count > 0, err
}

// TODO: SyncWithGoogleCalendar - Sync blocked dates with Gmail calendar
// func SyncWithGoogleCalendar(lawyerID string) error {
//     // Future implementation: Google Calendar API integration
//     return nil
// }

// TODO: SyncWithOutlookCalendar - Sync blocked dates with Outlook calendar
// func SyncWithOutlookCalendar(lawyerID string) error {
//     // Future implementation: Microsoft Graph API integration
//     return nil
// }

// CheckBlockedDateOverlap checks if a blocked date overlaps with existing ones
func CheckBlockedDateOverlap(db *gorm.DB, lawyerID string, startAt, endAt time.Time, excludeID string) (bool, error) {
	var count int64
	query := db.Model(&models.BlockedDate{}).Where("lawyer_id = ?", lawyerID)

	if excludeID != "" {
		query = query.Where("id != ?", excludeID)
	}

	// Simple overlap check: (StartA < EndB) and (EndA > StartB)
	err := query.Where("start_at < ? AND end_at > ?", endAt, startAt).Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
