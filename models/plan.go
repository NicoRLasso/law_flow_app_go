package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Plan tier constants
const (
	PlanTierTrial        = "trial"
	PlanTierStarter      = "starter"
	PlanTierProfessional = "professional"
	PlanTierEnterprise   = "enterprise"
)

// Plan represents a subscription tier with defined limits
type Plan struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Plan identification
	Name        string `gorm:"not null;uniqueIndex" json:"name"`
	Tier        string `gorm:"not null;index" json:"tier"`
	Description string `gorm:"type:text" json:"description"`

	// Pricing (in cents, for future Stripe integration)
	PriceMonthly  int    `gorm:"not null;default:0" json:"price_monthly"`
	PriceYearly   int    `gorm:"not null;default:0" json:"price_yearly"`
	StripePriceID string `json:"stripe_price_id,omitempty"`

	// Limit Pillars (-1 = unlimited)
	MaxUsers         int   `gorm:"not null" json:"max_users"`
	MaxStorageBytes  int64 `gorm:"not null" json:"max_storage_bytes"`
	MaxCases         int   `gorm:"not null" json:"max_cases"`
	TemplatesEnabled bool  `gorm:"not null;default:false" json:"templates_enabled"`

	// Trial specific
	TrialDays   int  `gorm:"not null;default:0" json:"trial_days"`
	IsTrialPlan bool `gorm:"not null;default:false" json:"is_trial_plan"`

	// Status
	IsActive     bool `gorm:"not null;default:true" json:"is_active"`
	IsDefault    bool `gorm:"not null;default:false" json:"is_default"`
	DisplayOrder int  `gorm:"not null;default:0" json:"display_order"`
}

// BeforeCreate hook to generate UUID
func (p *Plan) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name
func (Plan) TableName() string {
	return "plans"
}

// IsUnlimitedUsers checks if users are unlimited
func (p *Plan) IsUnlimitedUsers() bool {
	return p.MaxUsers == -1
}

// IsUnlimitedStorage checks if storage is unlimited
func (p *Plan) IsUnlimitedStorage() bool {
	return p.MaxStorageBytes == -1
}

// IsUnlimitedCases checks if cases are unlimited
func (p *Plan) IsUnlimitedCases() bool {
	return p.MaxCases == -1
}

// FormatStorageLimit returns human-readable storage limit
func (p *Plan) FormatStorageLimit() string {
	if p.MaxStorageBytes == -1 {
		return "Unlimited"
	}
	gb := float64(p.MaxStorageBytes) / (1024 * 1024 * 1024)
	if gb >= 1 {
		return fmt.Sprintf("%.0f GB", gb)
	}
	mb := float64(p.MaxStorageBytes) / (1024 * 1024)
	return fmt.Sprintf("%.0f MB", mb)
}

// FormatPriceMonthly returns formatted monthly price
func (p *Plan) FormatPriceMonthly() string {
	if p.PriceMonthly == 0 {
		return "Free"
	}
	return fmt.Sprintf("$%d/mo", p.PriceMonthly/100)
}

// FormatPriceYearly returns formatted yearly price
func (p *Plan) FormatPriceYearly() string {
	if p.PriceYearly == 0 {
		return "Free"
	}
	return fmt.Sprintf("$%d/yr", p.PriceYearly/100)
}
