package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ChoiceOption represents a firm-scoped choice option within a category
type ChoiceOption struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Category relationship
	CategoryID string         `gorm:"type:uuid;not null;index:idx_choice_opt_cat_order" json:"category_id"`
	Category   ChoiceCategory `gorm:"foreignKey:CategoryID" json:"category,omitempty"`

	// Option metadata
	Code     string `gorm:"not null" json:"code"`                                           // Internal value, e.g., "CC", "low"
	Label    string `gorm:"not null" json:"label"`                                          // Display text
	Order    int    `gorm:"not null;default:0;index:idx_choice_opt_cat_order" json:"order"` // For sorting options
	IsActive bool   `gorm:"not null;default:true" json:"is_active"`
	IsSystem bool   `gorm:"not null;default:false" json:"is_system"` // Prevents deletion of system options
}

// BeforeCreate hook to generate UUID
func (co *ChoiceOption) BeforeCreate(tx *gorm.DB) error {
	if co.ID == "" {
		co.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name for ChoiceOption model
func (ChoiceOption) TableName() string {
	return "choice_options"
}
