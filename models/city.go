package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// City represents a city/municipality within a department
type City struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	DepartmentID string     `gorm:"type:uuid;not null;index;uniqueIndex:idx_city_dept_code" json:"department_id"`
	Department   Department `gorm:"foreignKey:DepartmentID" json:"department,omitempty"`

	Code     string `gorm:"size:10;not null;uniqueIndex:idx_city_dept_code" json:"code"` // City code (e.g., "05001" for Medellín)
	Name     string `gorm:"size:100;not null" json:"name"`
	IsActive bool   `json:"is_active"`

	// Relationships
	Entities []LegalEntity `gorm:"foreignKey:CityID" json:"entities,omitempty"`
}

// BeforeCreate hook to generate UUID
func (c *City) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name
func (City) TableName() string {
	return "cities"
}
