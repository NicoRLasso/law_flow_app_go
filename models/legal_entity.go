package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LegalEntity represents a legal/judicial entity within a city (courts, tribunals, etc.)
type LegalEntity struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	CityID string `gorm:"type:uuid;not null;index" json:"city_id"`
	City   City   `gorm:"foreignKey:CityID" json:"city,omitempty"`

	Code     string `gorm:"size:15;not null;uniqueIndex" json:"code"` // Entity code (e.g., "0500131")
	Name     string `gorm:"size:200;not null" json:"name"`
	IsActive bool   `gorm:"default:true" json:"is_active"`

	// Relationships
	Specialties []LegalSpecialty `gorm:"foreignKey:EntityID" json:"specialties,omitempty"`
}

// BeforeCreate hook to generate UUID
func (e *LegalEntity) BeforeCreate(tx *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name
func (LegalEntity) TableName() string {
	return "legal_entities"
}
