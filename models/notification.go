package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

// Notification types
const (
	NotificationTypeJudicialUpdate = "JUDICIAL_UPDATE"
	NotificationTypeCaseUpdate     = "CASE_UPDATE"
	NotificationTypeSystem         = "SYSTEM"
)

type Notification struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Targeting
	FirmID string  `gorm:"type:uuid;not null;index" json:"firm_id"`
	UserID *string `gorm:"type:uuid;index" json:"user_id,omitempty"` // null = all firm users

	// Context
	CaseID                  *string `gorm:"type:uuid" json:"case_id,omitempty"`
	JudicialProcessActionID *string `gorm:"type:uuid" json:"judicial_process_action_id,omitempty"`

	// Content
	Type    string `gorm:"not null" json:"type"`
	Title   string `gorm:"not null" json:"title"`
	Message string `gorm:"type:text" json:"message"`
	LinkURL string `json:"link_url,omitempty"` // e.g., "/cases/{case_id}"

	// Read tracking
	ReadAt *time.Time `json:"read_at,omitempty"`

	// Relationships
	Firm Firm  `gorm:"foreignKey:FirmID" json:"-"`
	User *User `gorm:"foreignKey:UserID" json:"-"`
	Case *Case `gorm:"foreignKey:CaseID" json:"case,omitempty"`
}

func (n *Notification) BeforeCreate(tx *gorm.DB) error {
	if n.ID == "" {
		n.ID = uuid.New().String()
	}
	return nil
}

func (Notification) TableName() string {
	return "notifications"
}

func (n *Notification) IsRead() bool {
	return n.ReadAt != nil
}
