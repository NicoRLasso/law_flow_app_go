package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Activity type constants
const (
	ActivityTypeNote      = "NOTE"
	ActivityTypeCall      = "CALL"
	ActivityTypeMeeting   = "MEETING"
	ActivityTypeEmail     = "EMAIL"
	ActivityTypeTimeEntry = "TIME_ENTRY"
)

// ServiceActivity tracks interactions and time entries for a legal service
type ServiceActivity struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Scoping
	FirmID    string       `gorm:"type:uuid;not null;index" json:"firm_id"`
	Firm      Firm         `gorm:"foreignKey:FirmID" json:"firm,omitempty"`
	ServiceID string       `gorm:"type:uuid;not null;index:idx_activity_service" json:"service_id"`
	Service   LegalService `gorm:"foreignKey:ServiceID" json:"service,omitempty"`

	// Activity details
	ActivityType string `gorm:"not null" json:"activity_type"`
	Title        string `gorm:"not null" json:"title"`
	Content      string `gorm:"type:text" json:"content"`

	// Time tracking (primarily for TIME_ENTRY type)
	Duration   *int       `json:"duration,omitempty"`    // Duration in minutes
	OccurredAt *time.Time `json:"occurred_at,omitempty"` // When the activity occurred
	IsBillable bool       `gorm:"default:true" json:"is_billable"`

	// Contact info (for CALL/MEETING types)
	ContactName  *string `json:"contact_name,omitempty"`
	ContactPhone *string `json:"contact_phone,omitempty"`
	ContactEmail *string `json:"contact_email,omitempty"`

	// Document reference (optional)
	DocumentID *string          `gorm:"type:uuid" json:"document_id,omitempty"`
	Document   *ServiceDocument `gorm:"foreignKey:DocumentID" json:"document,omitempty"`

	// Creator
	CreatedByID string `gorm:"type:uuid;not null;index" json:"created_by_id"`
	CreatedBy   User   `gorm:"foreignKey:CreatedByID" json:"created_by,omitempty"`
}

// BeforeCreate hook to generate UUID
func (a *ServiceActivity) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name for ServiceActivity model
func (ServiceActivity) TableName() string {
	return "service_activities"
}

// IsNote checks if the activity is a note
func (a *ServiceActivity) IsNote() bool {
	return a.ActivityType == ActivityTypeNote
}

// IsCall checks if the activity is a call
func (a *ServiceActivity) IsCall() bool {
	return a.ActivityType == ActivityTypeCall
}

// IsMeeting checks if the activity is a meeting
func (a *ServiceActivity) IsMeeting() bool {
	return a.ActivityType == ActivityTypeMeeting
}

// IsEmail checks if the activity is an email
func (a *ServiceActivity) IsEmail() bool {
	return a.ActivityType == ActivityTypeEmail
}

// IsTimeEntry checks if the activity is a time entry
func (a *ServiceActivity) IsTimeEntry() bool {
	return a.ActivityType == ActivityTypeTimeEntry
}

// GetDurationHours returns the duration in hours (for display)
func (a *ServiceActivity) GetDurationHours() float64 {
	if a.Duration == nil {
		return 0
	}
	return float64(*a.Duration) / 60.0
}

// IsValidActivityType checks if the activity type is valid
func IsValidActivityType(activityType string) bool {
	validTypes := []string{
		ActivityTypeNote,
		ActivityTypeCall,
		ActivityTypeMeeting,
		ActivityTypeEmail,
		ActivityTypeTimeEntry,
	}
	for _, t := range validTypes {
		if t == activityType {
			return true
		}
	}
	return false
}

// GetActivityTypeDisplayName returns human-readable activity type name
func GetActivityTypeDisplayName(activityType string) string {
	names := map[string]string{
		ActivityTypeNote:      "Note",
		ActivityTypeCall:      "Phone Call",
		ActivityTypeMeeting:   "Meeting",
		ActivityTypeEmail:     "Email",
		ActivityTypeTimeEntry: "Time Entry",
	}
	if name, ok := names[activityType]; ok {
		return name
	}
	return activityType
}

// GetActivityTypeIcon returns an icon identifier for the activity type
func GetActivityTypeIcon(activityType string) string {
	icons := map[string]string{
		ActivityTypeNote:      "document-text",
		ActivityTypeCall:      "phone",
		ActivityTypeMeeting:   "users",
		ActivityTypeEmail:     "mail",
		ActivityTypeTimeEntry: "clock",
	}
	if icon, ok := icons[activityType]; ok {
		return icon
	}
	return "document"
}
