package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Availability represents a lawyer's standard weekly working hours
type Availability struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	LawyerID  string `gorm:"type:uuid;index;not null" json:"lawyer_id"` // References User
	DayOfWeek int    `gorm:"not null" json:"day_of_week"`               // 0=Sunday...6=Saturday
	StartTime string `gorm:"not null" json:"start_time"`                // "09:00" or "14:00"
	EndTime   string `gorm:"not null" json:"end_time"`                  // "12:00" or "17:00"
	IsActive  bool   `gorm:"default:true" json:"is_active"`

	// Relationships
	Lawyer User `gorm:"foreignKey:LawyerID" json:"lawyer,omitempty"`
}

// BeforeCreate hook to generate UUID
func (a *Availability) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name for Availability model
func (Availability) TableName() string {
	return "availabilities"
}

// DayName returns the name of the day
func (a *Availability) DayName() string {
	days := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	if a.DayOfWeek >= 0 && a.DayOfWeek < 7 {
		return days[a.DayOfWeek]
	}
	return ""
}

// TODO: SyncWithGoogleCalendar - Future integration with Gmail calendar
// TODO: SyncWithOutlookCalendar - Future integration with Outlook calendar
// TODO: ImportExternalEvents - Import blocked dates from external calendars
