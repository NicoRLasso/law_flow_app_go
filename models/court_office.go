package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CourtOffice represents a court office (despacho judicial) within a specialty
type CourtOffice struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	SpecialtyID string         `gorm:"type:uuid;not null;index" json:"specialty_id"`
	Specialty   LegalSpecialty `gorm:"foreignKey:SpecialtyID" json:"specialty,omitempty"`

	Code     string `gorm:"size:20;not null;uniqueIndex" json:"code"` // Court office code (e.g., "050013103001")
	Name     string `gorm:"size:250;not null" json:"name"`
	IsActive bool   `json:"is_active"`
}

// BeforeCreate hook to generate UUID
func (co *CourtOffice) BeforeCreate(tx *gorm.DB) error {
	if co.ID == "" {
		co.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name
func (CourtOffice) TableName() string {
	return "court_offices"
}
