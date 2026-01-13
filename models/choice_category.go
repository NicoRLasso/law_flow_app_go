package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ChoiceCategory represents a firm-scoped configurable choice category
type ChoiceCategory struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Firm relationship
	FirmID string `gorm:"type:uuid;not null;index:idx_choice_cat_firm_country;index:idx_choice_cat_firm_key" json:"firm_id"`
	Firm   Firm   `gorm:"foreignKey:FirmID" json:"firm,omitempty"`

	// Category metadata
	Country  string `gorm:"not null;index:idx_choice_cat_firm_country" json:"country"` // Full country name matching Firm.Country
	Key      string `gorm:"not null;index:idx_choice_cat_firm_key" json:"key"`         // e.g., "document_type", "priority"
	Name     string `gorm:"not null" json:"name"`                                      // Human-readable name
	Order    int    `gorm:"not null;default:0" json:"order"`                           // For sorting categories
	IsActive bool   `gorm:"not null;default:true" json:"is_active"`
	IsSystem bool   `gorm:"not null;default:false" json:"is_system"` // Prevents deletion of system categories

	// Relationships
	Options []ChoiceOption `gorm:"foreignKey:CategoryID" json:"options,omitempty"`
}

// BeforeCreate hook to generate UUID
func (cc *ChoiceCategory) BeforeCreate(tx *gorm.DB) error {
	if cc.ID == "" {
		cc.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name for ChoiceCategory model
func (ChoiceCategory) TableName() string {
	return "choice_categories"
}
