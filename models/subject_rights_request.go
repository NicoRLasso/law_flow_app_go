package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SubjectRequestType represents the type of data subject right being exercised (ARCO)
type SubjectRequestType string

const (
	SubjectRequestTypeAccess      SubjectRequestType = "ACCESS"      // Acceso - Right to access personal data
	SubjectRequestTypeRectify     SubjectRequestType = "RECTIFY"     // Rectificación - Right to correct data
	SubjectRequestTypeCancel      SubjectRequestType = "CANCEL"      // Cancelación - Right to delete data
	SubjectRequestTypeOpposition  SubjectRequestType = "OPPOSITION"  // Oposición - Right to object to processing
	SubjectRequestTypePortability SubjectRequestType = "PORTABILITY" // Portabilidad - Right to export data
)

// SubjectRequestStatus represents the status of a data subject request
type SubjectRequestStatus string

const (
	SubjectRequestStatusPending   SubjectRequestStatus = "PENDING"
	SubjectRequestStatusInReview  SubjectRequestStatus = "IN_REVIEW"
	SubjectRequestStatusApproved  SubjectRequestStatus = "APPROVED"
	SubjectRequestStatusDenied    SubjectRequestStatus = "DENIED"
	SubjectRequestStatusCompleted SubjectRequestStatus = "COMPLETED"
)

// SubjectRightsRequest represents a request from a data subject to exercise their rights under Law 1581.
type SubjectRightsRequest struct {
	ID        string    `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time `gorm:"index:idx_srr_created_at" json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Subject identification
	UserID    string `gorm:"type:uuid;not null;index:idx_srr_user" json:"user_id"`
	UserEmail string `gorm:"not null" json:"user_email"` // Denormalized

	// Firm scope
	FirmID *string `gorm:"type:uuid;index:idx_srr_firm" json:"firm_id,omitempty"`

	// Request details
	RequestType   SubjectRequestType   `gorm:"not null;index:idx_srr_type" json:"request_type"`
	Status        SubjectRequestStatus `gorm:"not null;default:PENDING;index:idx_srr_status" json:"status"`
	Justification string               `gorm:"type:text;not null" json:"justification"` // User's reason for request
	Response      string               `gorm:"type:text" json:"response,omitempty"`     // Firm's response

	// Resolution
	ResolvedAt   *time.Time `json:"resolved_at,omitempty"`
	ResolvedByID *string    `gorm:"type:uuid" json:"resolved_by_id,omitempty"`

	// Request metadata
	IPAddress string `gorm:"not null" json:"ip_address"`
	UserAgent string `gorm:"not null" json:"user_agent"`

	// Relationships
	User       *User `gorm:"foreignKey:UserID" json:"-"`
	Firm       *Firm `gorm:"foreignKey:FirmID" json:"-"`
	ResolvedBy *User `gorm:"foreignKey:ResolvedByID" json:"-"`
}

// BeforeCreate generates UUID
func (s *SubjectRightsRequest) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name
func (SubjectRightsRequest) TableName() string {
	return "subject_rights_requests"
}
