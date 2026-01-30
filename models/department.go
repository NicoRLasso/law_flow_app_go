package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Department represents a state/department/province within a country
type Department struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	CountryID string  `gorm:"type:uuid;not null;index;uniqueIndex:idx_dept_country_code" json:"country_id"`
	Country   Country `gorm:"foreignKey:CountryID" json:"country,omitempty"`

	Code     string `gorm:"size:10;not null;uniqueIndex:idx_dept_country_code" json:"code"` // Department code (e.g., "05" for Antioquia)
	Name     string `gorm:"size:100;not null" json:"name"`
	IsActive bool   `gorm:"default:true" json:"is_active"`

	// Relationships
	Cities []City `gorm:"foreignKey:DepartmentID" json:"cities,omitempty"`
}

// BeforeCreate hook to generate UUID
func (d *Department) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name
func (Department) TableName() string {
	return "departments"
}
