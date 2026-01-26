package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FirmAddOn tracks add-ons purchased by a firm
type FirmAddOn struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	FirmID  string    `gorm:"type:uuid;not null;index" json:"firm_id"`
	Firm    Firm      `gorm:"foreignKey:FirmID" json:"firm,omitempty"`
	AddOnID string    `gorm:"type:uuid;not null;index" json:"addon_id"`
	AddOn   PlanAddOn `gorm:"foreignKey:AddOnID" json:"addon,omitempty"`

	// Quantity (for recurring add-ons that can be stacked)
	Quantity int `gorm:"not null;default:1" json:"quantity"`

	// For recurring add-ons
	StartedAt *time.Time `json:"started_at,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// For one-time purchases (templates)
	PurchasedAt *time.Time `json:"purchased_at,omitempty"`
	IsPermanent bool       `gorm:"not null;default:false" json:"is_permanent"`

	// Status
	IsActive bool `gorm:"not null;default:true" json:"is_active"`

	// Audit
	PurchasedByUserID *string `gorm:"type:uuid" json:"purchased_by_user_id,omitempty"`
}

// BeforeCreate hook to generate UUID
func (fa *FirmAddOn) BeforeCreate(tx *gorm.DB) error {
	if fa.ID == "" {
		fa.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name
func (FirmAddOn) TableName() string {
	return "firm_addons"
}

// IsExpired checks if the add-on has expired
func (fa *FirmAddOn) IsExpired() bool {
	if fa.IsPermanent {
		return false
	}
	if fa.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*fa.ExpiresAt)
}

// IsValid checks if the add-on is active and not expired
func (fa *FirmAddOn) IsValid() bool {
	return fa.IsActive && !fa.IsExpired()
}

// GetEffectiveUsers returns the number of users this add-on provides
func (fa *FirmAddOn) GetEffectiveUsers() int {
	if !fa.IsValid() || fa.AddOn.Type != AddOnTypeUsers {
		return 0
	}
	return fa.AddOn.UnitsIncluded * fa.Quantity
}

// GetEffectiveStorage returns the storage bytes this add-on provides
func (fa *FirmAddOn) GetEffectiveStorage() int64 {
	if !fa.IsValid() || fa.AddOn.Type != AddOnTypeStorage {
		return 0
	}
	return fa.AddOn.StorageBytes * int64(fa.Quantity)
}

// GetEffectiveCases returns the number of cases this add-on provides
func (fa *FirmAddOn) GetEffectiveCases() int {
	if !fa.IsValid() || fa.AddOn.Type != AddOnTypeCases {
		return 0
	}
	return fa.AddOn.UnitsIncluded * fa.Quantity
}

// ProvidesTemplates checks if this add-on provides templates access
func (fa *FirmAddOn) ProvidesTemplates() bool {
	return fa.IsValid() && fa.AddOn.Type == AddOnTypeTemplates
}
