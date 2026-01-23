package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SupportTicket represents a support request from a user
type SupportTicket struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UserID  string `gorm:"type:uuid;not null;index" json:"user_id"`
	Subject string `gorm:"not null" json:"subject"`
	Message string `gorm:"not null" json:"message"`
	Status  string `gorm:"not null;default:open" json:"status"` // open, in_progress, resolved, closed

	// Response fields
	Response      *string    `json:"response,omitempty"`
	RespondedByID *string    `gorm:"type:uuid" json:"responded_by_id,omitempty"`
	RespondedAt   *time.Time `json:"responded_at,omitempty"`

	// Relationships
	User        *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	RespondedBy *User `gorm:"foreignKey:RespondedByID" json:"responded_by,omitempty"`
}

// BeforeCreate hook to generate UUID
func (t *SupportTicket) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name for SupportTicket model
func (SupportTicket) TableName() string {
	return "support_tickets"
}
