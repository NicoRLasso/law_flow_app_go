package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Colombian document types
const (
	DocumentTypeCC        = "CC"        // Cédula de Ciudadanía
	DocumentTypeCE        = "CE"        // Cédula de Extranjería
	DocumentTypePasaporte = "Pasaporte" // Passport
	DocumentTypeNIT       = "NIT"       // Company ID
	DocumentTypeTI        = "TI"        // Tarjeta de Identidad
)

// Priority levels
const (
	PriorityLow    = "low"
	PriorityMedium = "medium"
	PriorityHigh   = "high"
	PriorityUrgent = "urgent"
)

// Request status
const (
	StatusPending  = "pending"
	StatusAccepted = "accepted"
	StatusRejected = "rejected"
)

type CaseRequest struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Firm relationship
	FirmID string `gorm:"type:uuid;not null;index" json:"firm_id"`
	Firm   Firm   `gorm:"foreignKey:FirmID" json:"firm,omitempty"`

	// Requester information
	Name           string  `gorm:"not null" json:"name"`
	Email          string  `gorm:"not null" json:"email"`
	Phone          string  `gorm:"not null" json:"phone"`
	DocumentType   string  `gorm:"not null" json:"document_type"`               // Legacy: stores code (CC, CE, etc.)
	DocumentTypeID *string `gorm:"type:uuid" json:"document_type_id,omitempty"` // Foreign key to ChoiceOption
	DocumentNumber string  `gorm:"not null" json:"document_number"`

	// Case details
	Description   string `gorm:"type:text;not null" json:"description"`
	Priority      string `gorm:"not null;default:medium;index" json:"priority"`
	Status        string `gorm:"not null;default:pending;index" json:"status"`
	RejectionNote string `gorm:"type:text" json:"rejection_note,omitempty"`

	// File metadata
	FileName         string `json:"file_name,omitempty"`
	FileOriginalName string `json:"file_original_name,omitempty"`
	FilePath         string `json:"-"` // Not exposed in JSON for security
	FileSize         int64  `json:"file_size,omitempty"`

	// Audit fields
	IPAddress    string     `json:"ip_address,omitempty"`
	UserAgent    string     `gorm:"type:text" json:"user_agent,omitempty"`
	ReviewedByID *string    `gorm:"type:uuid" json:"reviewed_by_id,omitempty"`
	ReviewedBy   *User      `gorm:"foreignKey:ReviewedByID" json:"reviewed_by,omitempty"`
	ReviewedAt   *time.Time `json:"reviewed_at,omitempty"`

	// Relationships
	DocumentTypeOption *ChoiceOption `gorm:"foreignKey:DocumentTypeID" json:"document_type_option,omitempty"`
}

// BeforeCreate hook to generate UUID
func (cr *CaseRequest) BeforeCreate(tx *gorm.DB) error {
	if cr.ID == "" {
		cr.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name for CaseRequest model
func (CaseRequest) TableName() string {
	return "case_requests"
}

// IsValidDocumentType checks if the document type is valid
func IsValidDocumentType(docType string) bool {
	validTypes := []string{
		DocumentTypeCC,
		DocumentTypeCE,
		DocumentTypePasaporte,
		DocumentTypeNIT,
		DocumentTypeTI,
	}
	for _, t := range validTypes {
		if t == docType {
			return true
		}
	}
	return false
}

// IsValidPriority checks if the priority is valid
func IsValidPriority(priority string) bool {
	validPriorities := []string{
		PriorityLow,
		PriorityMedium,
		PriorityHigh,
		PriorityUrgent,
	}
	for _, p := range validPriorities {
		if p == priority {
			return true
		}
	}
	return false
}

// IsValidStatus checks if the status is valid
func IsValidStatus(status string) bool {
	validStatuses := []string{
		StatusPending,
		StatusAccepted,
		StatusRejected,
	}
	for _, s := range validStatuses {
		if s == status {
			return true
		}
	}
	return false
}
