package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Subscription status constants
const (
	SubscriptionStatusTrialing = "trialing"
	SubscriptionStatusActive   = "active"
	SubscriptionStatusPastDue  = "past_due"
	SubscriptionStatusCanceled = "canceled"
	SubscriptionStatusExpired  = "expired"
)

// FirmSubscription links a firm to a plan with status tracking
type FirmSubscription struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	FirmID string `gorm:"type:uuid;not null;uniqueIndex" json:"firm_id"`
	Firm   Firm   `gorm:"foreignKey:FirmID" json:"firm,omitempty"`

	PlanID string `gorm:"type:uuid;not null;index" json:"plan_id"`
	Plan   Plan   `gorm:"foreignKey:PlanID" json:"plan,omitempty"`

	// Subscription lifecycle
	Status             string     `gorm:"not null;default:trialing;index" json:"status"`
	StartedAt          time.Time  `gorm:"not null" json:"started_at"`
	TrialEndsAt        *time.Time `json:"trial_ends_at,omitempty"`
	CurrentPeriodStart *time.Time `json:"current_period_start,omitempty"`
	CurrentPeriodEnd   *time.Time `json:"current_period_end,omitempty"`
	CanceledAt         *time.Time `json:"canceled_at,omitempty"`

	// Stripe integration (future)
	StripeCustomerID     string `json:"stripe_customer_id,omitempty"`
	StripeSubscriptionID string `json:"stripe_subscription_id,omitempty"`

	// Audit trail
	LastPlanChangeAt *time.Time `json:"last_plan_change_at,omitempty"`
	PreviousPlanID   *string    `gorm:"type:uuid" json:"previous_plan_id,omitempty"`
	ChangedByUserID  *string    `gorm:"type:uuid" json:"changed_by_user_id,omitempty"`
}

// BeforeCreate hook to generate UUID
func (fs *FirmSubscription) BeforeCreate(tx *gorm.DB) error {
	if fs.ID == "" {
		fs.ID = uuid.New().String()
	}
	if fs.StartedAt.IsZero() {
		fs.StartedAt = time.Now()
	}
	return nil
}

// TableName specifies the table name
func (FirmSubscription) TableName() string {
	return "firm_subscriptions"
}

// IsTrialing checks if subscription is in trial period
func (fs *FirmSubscription) IsTrialing() bool {
	return fs.Status == SubscriptionStatusTrialing
}

// IsActive checks if subscription is active (paid or valid trial)
func (fs *FirmSubscription) IsActive() bool {
	return fs.Status == SubscriptionStatusActive || fs.Status == SubscriptionStatusTrialing
}

// IsExpired checks if subscription/trial has expired
func (fs *FirmSubscription) IsExpired() bool {
	return fs.Status == SubscriptionStatusExpired || fs.Status == SubscriptionStatusCanceled
}

// IsPastDue checks if subscription is past due
func (fs *FirmSubscription) IsPastDue() bool {
	return fs.Status == SubscriptionStatusPastDue
}

// TrialDaysRemaining returns days left in trial
func (fs *FirmSubscription) TrialDaysRemaining() int {
	if fs.TrialEndsAt == nil || !fs.IsTrialing() {
		return 0
	}
	days := int(time.Until(*fs.TrialEndsAt).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

// ShouldShowTrialWarning returns true if trial ends within 7 days
func (fs *FirmSubscription) ShouldShowTrialWarning() bool {
	if !fs.IsTrialing() || fs.TrialEndsAt == nil {
		return false
	}
	return fs.TrialDaysRemaining() <= 7
}

// HasTrialExpired checks if the trial period has passed
func (fs *FirmSubscription) HasTrialExpired() bool {
	if !fs.IsTrialing() || fs.TrialEndsAt == nil {
		return false
	}
	return time.Now().After(*fs.TrialEndsAt)
}

// GetStatusDisplay returns a human-readable status
func (fs *FirmSubscription) GetStatusDisplay() string {
	switch fs.Status {
	case SubscriptionStatusTrialing:
		return "Trial"
	case SubscriptionStatusActive:
		return "Active"
	case SubscriptionStatusPastDue:
		return "Past Due"
	case SubscriptionStatusCanceled:
		return "Canceled"
	case SubscriptionStatusExpired:
		return "Expired"
	default:
		return fs.Status
	}
}

// GetStatusBadgeClass returns the CSS class for the status badge
func (fs *FirmSubscription) GetStatusBadgeClass() string {
	switch fs.Status {
	case SubscriptionStatusTrialing:
		return "badge-info"
	case SubscriptionStatusActive:
		return "badge-success"
	case SubscriptionStatusPastDue:
		return "badge-warning"
	case SubscriptionStatusCanceled, SubscriptionStatusExpired:
		return "badge-error"
	default:
		return "badge-ghost"
	}
}
