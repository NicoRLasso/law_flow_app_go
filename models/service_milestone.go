package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Milestone status constants
const (
	MilestoneStatusPending    = "PENDING"
	MilestoneStatusInProgress = "IN_PROGRESS"
	MilestoneStatusCompleted  = "COMPLETED"
	MilestoneStatusSkipped    = "SKIPPED"
)

// ServiceMilestone represents a deliverable step in the service workflow
type ServiceMilestone struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Parent relationships
	FirmID    string       `gorm:"type:uuid;not null;index" json:"firm_id"`
	Firm      Firm         `gorm:"foreignKey:FirmID" json:"firm,omitempty"`
	ServiceID string       `gorm:"type:uuid;not null;index:idx_milestone_service" json:"service_id"`
	Service   LegalService `gorm:"foreignKey:ServiceID" json:"service,omitempty"`

	// Milestone details
	Title       string  `gorm:"not null" json:"title"`
	Description *string `gorm:"type:text" json:"description,omitempty"`
	SortOrder   int     `gorm:"not null;default:0" json:"sort_order"`

	// Status tracking
	Status      string     `gorm:"not null;default:PENDING" json:"status"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CompletedBy *string    `gorm:"type:uuid" json:"completed_by,omitempty"`

	// Document reference (optional output from this milestone)
	OutputDocumentID *string          `gorm:"type:uuid" json:"output_document_id,omitempty"`
	OutputDocument   *ServiceDocument `gorm:"foreignKey:OutputDocumentID" json:"output_document,omitempty"`

	// Relationships
	Completer *User `gorm:"foreignKey:CompletedBy" json:"completer,omitempty"`
}

// BeforeCreate hook to generate UUID
func (m *ServiceMilestone) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name for ServiceMilestone model
func (ServiceMilestone) TableName() string {
	return "service_milestones"
}

// IsPending checks if the milestone is pending
func (m *ServiceMilestone) IsPending() bool {
	return m.Status == MilestoneStatusPending
}

// IsInProgress checks if the milestone is in progress
func (m *ServiceMilestone) IsInProgress() bool {
	return m.Status == MilestoneStatusInProgress
}

// IsCompleted checks if the milestone is completed
func (m *ServiceMilestone) IsCompleted() bool {
	return m.Status == MilestoneStatusCompleted
}

// IsSkipped checks if the milestone was skipped
func (m *ServiceMilestone) IsSkipped() bool {
	return m.Status == MilestoneStatusSkipped
}

// IsValidMilestoneStatus checks if the status is valid
func IsValidMilestoneStatus(status string) bool {
	validStatuses := []string{
		MilestoneStatusPending,
		MilestoneStatusInProgress,
		MilestoneStatusCompleted,
		MilestoneStatusSkipped,
	}
	for _, s := range validStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// GetMilestoneStatusDisplayName returns human-readable status name
func GetMilestoneStatusDisplayName(status string) string {
	names := map[string]string{
		MilestoneStatusPending:    "Pending",
		MilestoneStatusInProgress: "In Progress",
		MilestoneStatusCompleted:  "Completed",
		MilestoneStatusSkipped:    "Skipped",
	}
	if name, ok := names[status]; ok {
		return name
	}
	return status
}
