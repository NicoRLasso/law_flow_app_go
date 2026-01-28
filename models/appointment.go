package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Appointment status constants
const (
	AppointmentStatusScheduled = "SCHEDULED"
	AppointmentStatusConfirmed = "CONFIRMED"
	AppointmentStatusCancelled = "CANCELLED"
	AppointmentStatusCompleted = "COMPLETED"
	AppointmentStatusNoShow    = "NO_SHOW"
)

// Appointment represents a scheduled appointment between a lawyer and a client
type Appointment struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Firm relationship
	FirmID string `gorm:"type:uuid;index;not null" json:"firm_id"`
	Firm   Firm   `gorm:"foreignKey:FirmID" json:"firm,omitempty"`

	// Appointment Type
	AppointmentTypeID *string          `gorm:"type:uuid;index" json:"appointment_type_id,omitempty"`
	AppointmentType   *AppointmentType `gorm:"foreignKey:AppointmentTypeID" json:"appointment_type,omitempty"`

	// Lawyer relationship
	LawyerID string `gorm:"type:uuid;index;not null" json:"lawyer_id"`
	Lawyer   User   `gorm:"foreignKey:LawyerID" json:"lawyer,omitempty"`

	// Client relationship (User with role 'client')
	ClientID *string `gorm:"type:uuid;index" json:"client_id,omitempty"`
	Client   *User   `gorm:"foreignKey:ClientID" json:"client,omitempty"`

	// Client Info Snapshot (preserved even if client data changes)
	ClientName  string  `gorm:"size:200;not null" json:"client_name"`
	ClientEmail string  `gorm:"size:255;not null;index" json:"client_email"`
	ClientPhone *string `gorm:"size:20" json:"client_phone,omitempty"`

	// Schedule
	ScheduledDate   time.Time `gorm:"type:date;index;not null" json:"scheduled_date"` // Date only for queries
	StartTime       time.Time `gorm:"not null;index" json:"start_time"`               // Full datetime (UTC)
	EndTime         time.Time `gorm:"not null;index" json:"end_time"`                 // Full datetime (UTC)
	DurationMinutes int       `gorm:"not null" json:"duration_minutes"`

	// Status
	Status             string     `gorm:"size:20;default:'SCHEDULED';index" json:"status"`
	CancellationReason *string    `gorm:"type:text" json:"cancellation_reason,omitempty"`
	CancelledAt        *time.Time `json:"cancelled_at,omitempty"`
	CancelledByID      *string    `gorm:"type:uuid" json:"cancelled_by_id,omitempty"`
	CancelledBy        *User      `gorm:"foreignKey:CancelledByID" json:"cancelled_by,omitempty"`

	// Public Access Token (for reschedule/cancel via email link)
	BookingToken string `gorm:"type:uuid;uniqueIndex;not null" json:"booking_token"`

	// Notes
	Notes         *string `gorm:"type:text" json:"notes,omitempty"`          // Visible to client
	InternalNotes *string `gorm:"type:text" json:"internal_notes,omitempty"` // Staff only

	// Meeting URL (for video calls)
	MeetingURL *string `gorm:"size:500" json:"meeting_url,omitempty"`

	// Reminder System
	ReminderSentAt *time.Time `json:"reminder_sent_at,omitempty"`

	// Optional links
	CaseID *string `gorm:"type:uuid;index" json:"case_id,omitempty"`
	Case   *Case   `gorm:"foreignKey:CaseID" json:"case,omitempty"`
}

// BeforeCreate hook to generate UUID and BookingToken
func (a *Appointment) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	if a.BookingToken == "" {
		a.BookingToken = uuid.New().String()
	}
	// Calculate duration if not set
	if a.DurationMinutes == 0 && !a.EndTime.IsZero() && !a.StartTime.IsZero() {
		a.DurationMinutes = int(a.EndTime.Sub(a.StartTime).Minutes())
	}
	// Set ScheduledDate from StartTime if not set
	if a.ScheduledDate.IsZero() && !a.StartTime.IsZero() {
		a.ScheduledDate = time.Date(a.StartTime.Year(), a.StartTime.Month(), a.StartTime.Day(), 0, 0, 0, 0, time.UTC)
	}
	return nil
}

// TableName specifies the table name for Appointment model
func (Appointment) TableName() string {
	return "appointments"
}

// IsValidAppointmentStatus checks if the status is valid
func IsValidAppointmentStatus(status string) bool {
	validStatuses := []string{
		AppointmentStatusScheduled,
		AppointmentStatusConfirmed,
		AppointmentStatusCancelled,
		AppointmentStatusCompleted,
		AppointmentStatusNoShow,
	}
	for _, s := range validStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// IsCancellable checks if the appointment can be cancelled
func (a *Appointment) IsCancellable() bool {
	return a.Status == AppointmentStatusScheduled || a.Status == AppointmentStatusConfirmed
}

// IsEditable checks if the appointment can be modified
func (a *Appointment) IsEditable() bool {
	return a.Status == AppointmentStatusScheduled || a.Status == AppointmentStatusConfirmed
}

// Duration returns the duration of the appointment in minutes
func (a *Appointment) Duration() int {
	if a.DurationMinutes > 0 {
		return a.DurationMinutes
	}
	return int(a.EndTime.Sub(a.StartTime).Minutes())
}

// TimeSlot represents an available time slot for booking
type TimeSlot struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}
