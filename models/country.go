package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Country represents a country in the system
type Country struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Code     string `gorm:"size:3;uniqueIndex;not null" json:"code"` // ISO 3166-1 alpha-3 (COL, USA, etc.)
	Name     string `gorm:"size:100;not null" json:"name"`
	IsActive bool   `json:"is_active"`

	// Relationships
	Departments []Department `gorm:"foreignKey:CountryID" json:"departments,omitempty"`
}

// BeforeCreate hook to generate UUID
func (c *Country) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name
func (Country) TableName() string {
	return "countries"
}
