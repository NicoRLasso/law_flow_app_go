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
	Email       string     `gorm:"uniqueIndex;not null" json:"email"`
	Password    string     `gorm:"not null" json:"-"`
	FirmID      *string    `gorm:"type:uuid;index" json:"firm_id"`     // Nullable - user may not have firm yet
	Role        string     `gorm:"not null;default:staff" json:"role"` // admin, lawyer, staff, client
	IsActive    bool       `gorm:"not null;default:true" json:"is_active"`
	LastLoginAt *time.Time `json:"last_login_at"`

	// Relationships
	Firm *Firm `gorm:"foreignKey:FirmID" json:"firm,omitempty"`
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

// TableName specifies the table name for User model
func (User) TableName() string {
	return "users"
}
