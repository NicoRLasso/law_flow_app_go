package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditAction represents the type of operation performed
type AuditAction string

const (
	AuditActionCreate           AuditAction = "CREATE"
	AuditActionRead             AuditAction = "READ"
	AuditActionUpdate           AuditAction = "UPDATE"
	AuditActionDelete           AuditAction = "DELETE"
	AuditActionView             AuditAction = "VIEW"              // Document viewed/previewed
	AuditActionDownload         AuditAction = "DOWNLOAD"          // Document downloaded
	AuditActionVisibilityChange AuditAction = "VISIBILITY_CHANGE" // Document visibility toggled
	AuditActionLogin            AuditAction = "LOGIN"             // User logged in
	AuditActionLogout           AuditAction = "LOGOUT"            // User logged out
)

// AuditLog represents an immutable record of a data operation
type AuditLog struct {
	ID        string    `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time `gorm:"index:idx_audit_created_at" json:"created_at"`

	// Actor identification
	UserID   *string `gorm:"type:uuid;index:idx_audit_user" json:"user_id,omitempty"`
	UserName string  `gorm:"not null" json:"user_name"` // Denormalized for historical accuracy
	UserRole string  `gorm:"not null" json:"user_role"` // Denormalized

	// Firm scope
	FirmID   *string `gorm:"type:uuid;index:idx_audit_firm" json:"firm_id,omitempty"`
	FirmName string  `json:"firm_name,omitempty"` // Denormalized

	// Target resource
	ResourceType string `gorm:"not null;index:idx_audit_resource" json:"resource_type"` // e.g., "Case", "User"
	ResourceID   string `gorm:"type:uuid;not null;index:idx_audit_resource" json:"resource_id"`
	ResourceName string `json:"resource_name,omitempty"` // Human-readable identifier (e.g., case number)

	// Operation details
	Action      AuditAction `gorm:"not null;index:idx_audit_action" json:"action"`
	Description string      `gorm:"type:text" json:"description,omitempty"` // Human-readable summary

	// Change tracking (for UPDATE operations)
	OldValues string `gorm:"type:text" json:"old_values,omitempty"` // JSON encoded
	NewValues string `gorm:"type:text" json:"new_values,omitempty"` // JSON encoded

	// Request metadata (optional)
	IPAddress string `json:"ip_address,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`

	// Relationships (for reading, not for data integrity)
	User *User `gorm:"foreignKey:UserID" json:"-"`
	Firm *Firm `gorm:"foreignKey:FirmID" json:"-"`
}

// BeforeCreate generates UUID and prevents modification
func (a *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}

// BeforeUpdate prevents modification of audit logs (immutability)
func (a *AuditLog) BeforeUpdate(tx *gorm.DB) error {
	return gorm.ErrRecordNotFound // Prevent any updates
}

// BeforeDelete prevents deletion of audit logs (immutability)
func (a *AuditLog) BeforeDelete(tx *gorm.DB) error {
	return gorm.ErrRecordNotFound // Prevent any deletes
}

// TableName specifies the table name
func (AuditLog) TableName() string {
	return "audit_logs"
}
