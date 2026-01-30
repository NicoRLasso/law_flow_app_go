package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CaseMilestone represents a deliverable step or significant event in a legal case
type CaseMilestone struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Parent relationships
	FirmID string `gorm:"type:uuid;not null;index" json:"firm_id"`
	Firm   Firm   `gorm:"foreignKey:FirmID" json:"firm,omitempty"`
	CaseID string `gorm:"type:uuid;not null;index:idx_case_milestone" json:"case_id"`
	Case   Case   `gorm:"foreignKey:CaseID" json:"case,omitempty"`

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
	OutputDocumentID *string       `gorm:"type:uuid" json:"output_document_id,omitempty"`
	OutputDocument   *CaseDocument `gorm:"foreignKey:OutputDocumentID" json:"output_document,omitempty"`

	// Relationships
	Completer *User `gorm:"foreignKey:CompletedBy" json:"completer,omitempty"`
}

// BeforeCreate hook to generate UUID
func (m *CaseMilestone) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name for CaseMilestone model
func (CaseMilestone) TableName() string {
	return "case_milestones"
}

// Helper methods (reusing the same logic as ServiceMilestone)
func (m *CaseMilestone) IsPending() bool {
	return m.Status == MilestoneStatusPending
}

func (m *CaseMilestone) IsInProgress() bool {
	return m.Status == MilestoneStatusInProgress
}

func (m *CaseMilestone) IsCompleted() bool {
	return m.Status == MilestoneStatusCompleted
}

func (m *CaseMilestone) IsSkipped() bool {
	return m.Status == MilestoneStatusSkipped
}
