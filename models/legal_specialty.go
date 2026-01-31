package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LegalSpecialty represents a legal specialty within an entity (civil, penal, laboral, etc.)
type LegalSpecialty struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	EntityID string      `gorm:"type:uuid;not null;index" json:"entity_id"`
	Entity   LegalEntity `gorm:"foreignKey:EntityID" json:"entity,omitempty"`

	Code     string `gorm:"size:15;not null;uniqueIndex" json:"code"` // Specialty code (e.g., "050013103")
	Name     string `gorm:"size:150;not null" json:"name"`
	IsActive bool   `json:"is_active"`

	// Relationships
	CourtOffices []CourtOffice `gorm:"foreignKey:SpecialtyID" json:"court_offices,omitempty"`
}

// BeforeCreate hook to generate UUID
func (s *LegalSpecialty) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name
func (LegalSpecialty) TableName() string {
	return "legal_specialties"
}
