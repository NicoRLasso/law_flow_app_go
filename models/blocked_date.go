package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BlockedDate represents a date/time range when a lawyer is unavailable
type BlockedDate struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	LawyerID  string    `gorm:"type:uuid;index;not null" json:"lawyer_id"`
	StartAt   time.Time `gorm:"not null;index" json:"start_at"`
	EndAt     time.Time `gorm:"not null;index" json:"end_at"`
	Reason    string    `json:"reason"` // "Vacation", "Holiday", "Personal", "Other"
	IsFullDay bool      `gorm:"default:false" json:"is_full_day"`

	// Relationships
	Lawyer User `gorm:"foreignKey:LawyerID" json:"lawyer,omitempty"`
}

// BeforeCreate hook to generate UUID
func (b *BlockedDate) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name for BlockedDate model
func (BlockedDate) TableName() string {
	return "blocked_dates"
}

// IsBlocking checks if this blocked date blocks a given time range
func (b *BlockedDate) IsBlocking(checkStart, checkEnd time.Time) bool {
	// Simple range overlap check: (StartA < EndB) and (EndA > StartB)
	return b.StartAt.Before(checkEnd) && b.EndAt.After(checkStart)
}
