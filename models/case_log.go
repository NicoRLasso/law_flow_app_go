package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CaseLog struct {
	ID           string `gorm:"type:uuid;primaryKey"`
	FirmID       string `gorm:"index"` // Multi-tenancy
	CaseID       string `gorm:"index"`
	EntryType    string // note, document, call, meeting
	Title        string
	Content      string  `gorm:"type:text"`
	DocumentID   *string        `gorm:"index"` // Optional reference to a document
	Document     *CaseDocument `gorm:"foreignKey:DocumentID"`
	ContactName  *string
	ContactPhone *string
	OccurredAt   *time.Time
	Duration     *int   // Duration in minutes
	CreatedByID  string `gorm:"index"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// BeforeCreate hook to generate UUID
func (log *CaseLog) BeforeCreate(tx *gorm.DB) error {
	if log.ID == "" {
		log.ID = uuid.New().String()
	}
	return nil
}
