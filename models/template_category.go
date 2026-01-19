package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TemplateCategory represents a category for organizing document templates
type TemplateCategory struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Firm relationship (multi-tenant scoping)
	FirmID string `gorm:"type:uuid;not null;index" json:"firm_id"`
	Firm   Firm   `gorm:"foreignKey:FirmID" json:"firm,omitempty"`

	// Category details
	Name        string  `gorm:"not null" json:"name"`
	Description *string `gorm:"type:text" json:"description,omitempty"`

	// Ordering
	SortOrder int `gorm:"not null;default:0" json:"sort_order"`

	// Status
	IsActive bool `gorm:"not null;default:true" json:"is_active"`

	// Relationships
	Templates []DocumentTemplate `gorm:"foreignKey:CategoryID" json:"templates,omitempty"`
}

// BeforeCreate hook to generate UUID
func (c *TemplateCategory) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name for TemplateCategory model
func (TemplateCategory) TableName() string {
	return "template_categories"
}
