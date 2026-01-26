package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Add-on type constants
const (
	AddOnTypeUsers     = "users"
	AddOnTypeStorage   = "storage"
	AddOnTypeCases     = "cases"
	AddOnTypeTemplates = "templates"
)

// PlanAddOn represents purchasable add-ons for extra capacity
type PlanAddOn struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Add-on identification
	Name        string `gorm:"not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`
	Type        string `gorm:"not null;index" json:"type"` // users, storage, cases, templates

	// What you get
	UnitsIncluded int   `gorm:"not null;default:0" json:"units_included"` // e.g., 5 users, 50 cases
	StorageBytes  int64 `gorm:"not null;default:0" json:"storage_bytes"`  // for storage add-ons (e.g., 5GB)

	// Pricing (in cents)
	PriceMonthly int  `gorm:"not null;default:0" json:"price_monthly"` // recurring price
	PriceOneTime int  `gorm:"not null;default:0" json:"price_one_time"` // one-time purchase (e.g., templates)
	IsRecurring  bool `gorm:"not null;default:true" json:"is_recurring"`

	// Status
	IsActive     bool `gorm:"not null;default:true" json:"is_active"`
	DisplayOrder int  `gorm:"not null;default:0" json:"display_order"`
}

// BeforeCreate hook to generate UUID
func (pa *PlanAddOn) BeforeCreate(tx *gorm.DB) error {
	if pa.ID == "" {
		pa.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name
func (PlanAddOn) TableName() string {
	return "plan_addons"
}

// FormatPrice returns the formatted price
func (pa *PlanAddOn) FormatPrice() string {
	if pa.IsRecurring {
		if pa.PriceMonthly == 0 {
			return "Free"
		}
		return fmt.Sprintf("$%d/mo", pa.PriceMonthly/100)
	}
	if pa.PriceOneTime == 0 {
		return "Free"
	}
	return fmt.Sprintf("$%d", pa.PriceOneTime/100)
}

// FormatStorageIncluded returns human-readable storage for storage add-ons
func (pa *PlanAddOn) FormatStorageIncluded() string {
	if pa.Type != AddOnTypeStorage || pa.StorageBytes == 0 {
		return ""
	}
	return FormatBytes(pa.StorageBytes)
}

// GetTypeDisplay returns human-readable type
func (pa *PlanAddOn) GetTypeDisplay() string {
	switch pa.Type {
	case AddOnTypeUsers:
		return "Users"
	case AddOnTypeStorage:
		return "Storage"
	case AddOnTypeCases:
		return "Cases"
	case AddOnTypeTemplates:
		return "Templates"
	default:
		return pa.Type
	}
}

// GetValueDisplay returns what you get from this add-on
func (pa *PlanAddOn) GetValueDisplay() string {
	switch pa.Type {
	case AddOnTypeUsers:
		return fmt.Sprintf("+%d users", pa.UnitsIncluded)
	case AddOnTypeStorage:
		return fmt.Sprintf("+%s", pa.FormatStorageIncluded())
	case AddOnTypeCases:
		return fmt.Sprintf("+%d cases", pa.UnitsIncluded)
	case AddOnTypeTemplates:
		return "Document Templates"
	default:
		return ""
	}
}
