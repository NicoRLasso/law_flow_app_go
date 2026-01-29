package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Service status constants (workflow states - must remain fixed)
const (
	ServiceStatusIntake     = "INTAKE"      // Initial request phase
	ServiceStatusInProgress = "IN_PROGRESS" // Active work
	ServiceStatusOnHold     = "ON_HOLD"     // Paused (awaiting client action)
	ServiceStatusCompleted  = "COMPLETED"   // Delivered
	ServiceStatusCancelled  = "CANCELLED"   // Cancelled
)

// Service priority constants (workflow states - must remain fixed)
const (
	ServicePriorityLow    = "LOW"
	ServicePriorityNormal = "NORMAL"
	ServicePriorityHigh   = "HIGH"
	ServicePriorityUrgent = "URGENT"
)

// ChoiceCategory keys for services (used to identify categories in the database)
const (
	ChoiceCategoryKeyServiceType = "service_type" // Category key for service types
)

// LegalService represents a non-litigation legal service
type LegalService struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Firm relationship (multi-tenancy)
	FirmID string `gorm:"type:uuid;not null;index:idx_service_firm;index:idx_service_firm_status" json:"firm_id"`
	Firm   Firm   `gorm:"foreignKey:FirmID" json:"firm,omitempty"`

	// Service identification
	ServiceNumber string `gorm:"not null;uniqueIndex" json:"service_number"` // e.g., SVC-2026-00001
	Title         string `gorm:"not null" json:"title"`
	Description   string `gorm:"type:text" json:"description"`

	// Service type (references ChoiceOption)
	ServiceTypeID *string       `gorm:"type:uuid;index" json:"service_type_id,omitempty"`
	ServiceType   *ChoiceOption `gorm:"foreignKey:ServiceTypeID" json:"service_type,omitempty"`

	// Client relationship
	ClientID string `gorm:"type:uuid;not null;index" json:"client_id"`
	Client   User   `gorm:"foreignKey:ClientID" json:"client,omitempty"`

	// Objective & Deliverables
	Objective           string  `gorm:"type:text;not null" json:"objective"` // What the client wants achieved
	ExpectedDeliverable *string `gorm:"type:text" json:"expected_deliverable,omitempty"`

	// Status and lifecycle
	Status          string     `gorm:"not null;default:INTAKE;index:idx_service_firm_status" json:"status"`
	StatusChangedAt *time.Time `gorm:"index" json:"status_changed_at,omitempty"`
	StatusChangedBy *string    `gorm:"type:uuid" json:"status_changed_by,omitempty"`

	// Time tracking
	EstimatedHours   *float64   `json:"estimated_hours,omitempty"`
	ActualHours      float64    `gorm:"default:0" json:"actual_hours"`
	EstimatedDueDate *time.Time `json:"estimated_due_date,omitempty"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`

	// Assignment
	AssignedToID *string `gorm:"type:uuid" json:"assigned_to_id,omitempty"`
	AssignedTo   *User   `gorm:"foreignKey:AssignedToID" json:"assigned_to,omitempty"`

	// Priority
	Priority string `gorm:"not null;default:NORMAL" json:"priority"`

	// Internal notes (not exposed to client)
	InternalNotes *string `gorm:"type:text" json:"-"`

	// Relationships
	StatusChanger *User              `gorm:"foreignKey:StatusChangedBy" json:"status_changer,omitempty"`
	Milestones    []ServiceMilestone `gorm:"foreignKey:ServiceID" json:"milestones,omitempty"`
	Documents     []ServiceDocument  `gorm:"foreignKey:ServiceID" json:"documents,omitempty"`
	Expenses      []ServiceExpense   `gorm:"foreignKey:ServiceID" json:"expenses,omitempty"`
	Activities    []ServiceActivity  `gorm:"foreignKey:ServiceID" json:"activities,omitempty"`
	Collaborators []User             `gorm:"many2many:service_collaborators;" json:"collaborators,omitempty"`
}

// BeforeCreate hook to generate UUID
func (s *LegalService) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name for LegalService model
func (LegalService) TableName() string {
	return "legal_services"
}

// GetServiceTypeLabel returns the service type label or empty string
func (s *LegalService) GetServiceTypeLabel() string {
	if s.ServiceType != nil {
		return s.ServiceType.Label
	}
	return ""
}

// GetServiceTypeCode returns the service type code or empty string
func (s *LegalService) GetServiceTypeCode() string {
	if s.ServiceType != nil {
		return s.ServiceType.Code
	}
	return ""
}

// IsCompleted checks if the service is completed
func (s *LegalService) IsCompleted() bool {
	return s.Status == ServiceStatusCompleted
}

// IsActive checks if the service is in progress
func (s *LegalService) IsActive() bool {
	return s.Status == ServiceStatusInProgress
}

// IsIntake checks if the service is in intake phase
func (s *LegalService) IsIntake() bool {
	return s.Status == ServiceStatusIntake
}

// IsOnHold checks if the service is on hold
func (s *LegalService) IsOnHold() bool {
	return s.Status == ServiceStatusOnHold
}

// IsCancelled checks if the service is cancelled
func (s *LegalService) IsCancelled() bool {
	return s.Status == ServiceStatusCancelled
}

// IsValidServiceStatus checks if the status is valid
func IsValidServiceStatus(status string) bool {
	validStatuses := []string{
		ServiceStatusIntake,
		ServiceStatusInProgress,
		ServiceStatusOnHold,
		ServiceStatusCompleted,
		ServiceStatusCancelled,
	}
	for _, s := range validStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// IsValidServicePriority checks if the priority is valid
func IsValidServicePriority(priority string) bool {
	validPriorities := []string{
		ServicePriorityLow,
		ServicePriorityNormal,
		ServicePriorityHigh,
		ServicePriorityUrgent,
	}
	for _, p := range validPriorities {
		if p == priority {
			return true
		}
	}
	return false
}

// GetServiceStatusDisplayName returns human-readable status name
func GetServiceStatusDisplayName(status string) string {
	names := map[string]string{
		ServiceStatusIntake:     "Intake",
		ServiceStatusInProgress: "In Progress",
		ServiceStatusOnHold:     "On Hold",
		ServiceStatusCompleted:  "Completed",
		ServiceStatusCancelled:  "Cancelled",
	}
	if name, ok := names[status]; ok {
		return name
	}
	return status
}

// GetServicePriorityDisplayName returns human-readable priority name
func GetServicePriorityDisplayName(priority string) string {
	names := map[string]string{
		ServicePriorityLow:    "Low",
		ServicePriorityNormal: "Normal",
		ServicePriorityHigh:   "High",
		ServicePriorityUrgent: "Urgent",
	}
	if name, ok := names[priority]; ok {
		return name
	}
	return priority
}
