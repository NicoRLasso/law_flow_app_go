package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Name        string     `gorm:"not null" json:"name"`
	Email       string     `gorm:"not null;uniqueIndex:idx_user_firm_email_role" json:"email"` // Unique per firm+role combination
	Password    string     `gorm:"not null" json:"-"`
	FirmID      *string    `gorm:"type:uuid;index;uniqueIndex:idx_user_firm_email_role" json:"firm_id"`     // Nullable - user may not have firm yet
	Role        string     `gorm:"not null;default:staff;uniqueIndex:idx_user_firm_email_role" json:"role"` // superadmin, admin, lawyer, staff, client
	IsActive    bool       `gorm:"not null;default:true" json:"is_active"`
	Language    string     `gorm:"not null;default:'es'" json:"language"` // en, es
	LastLoginAt *time.Time `json:"last_login_at"`

	// Security / Lockout
	FailedLoginAttempts int        `gorm:"default:0" json:"-"`
	LockoutUntil        *time.Time `json:"-"`
	LockoutCount        int        `gorm:"default:0" json:"-"` // Tracks lockout occurrences for exponential backoff

	// Optional personal information
	Address        *string `json:"address,omitempty"`
	PhoneNumber    *string `json:"phone_number,omitempty"`
	DocumentTypeID *string `gorm:"type:uuid" json:"document_type_id,omitempty"` // Foreign key to ChoiceOption
	DocumentNumber *string `json:"document_number,omitempty"`

	// Relationships
	Firm         *Firm         `gorm:"foreignKey:FirmID" json:"firm,omitempty"`
	DocumentType *ChoiceOption `gorm:"foreignKey:DocumentTypeID" json:"document_type,omitempty"`
}

// BeforeCreate hook to generate UUID
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return nil
}

// HasFirm checks if the user has a firm assigned
func (u *User) HasFirm() bool {
	return u.FirmID != nil && *u.FirmID != ""
}

// IsSuperadmin checks if the user is a superadmin
func (u *User) IsSuperadmin() bool {
	return u.Role == "superadmin"
}

// TableName specifies the table name for User model
func (User) TableName() string {
	return "users"
}
