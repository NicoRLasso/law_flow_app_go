package models

import (
	"time"
)

type Session struct {
	ID        string    `gorm:"primarykey;type:varchar(36)" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	UserID    string    `gorm:"type:uuid;not null;index" json:"user_id"`
	FirmID    *string   `gorm:"type:uuid;index" json:"firm_id"` // Nullable for users without a firm (e.g., superadmin)
	Token     string    `gorm:"uniqueIndex;not null;type:varchar(128)" json:"-"`
	ExpiresAt time.Time `gorm:"not null;index" json:"expires_at"`
	IPAddress string    `gorm:"type:varchar(45)" json:"ip_address"`
	UserAgent string    `gorm:"type:text" json:"user_agent"`

	// Relationships
	User User `gorm:"foreignKey:UserID" json:"-"`
	Firm *Firm `gorm:"foreignKey:FirmID" json:"-"`
}

// TableName specifies the table name for Session model
func (Session) TableName() string {
	return "sessions"
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}
