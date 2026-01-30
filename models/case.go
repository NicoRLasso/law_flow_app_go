package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Case status constants
const (
	CaseStatusOpen   = "OPEN"
	CaseStatusOnHold = "ON_HOLD"
	CaseStatusClosed = "CLOSED"
)

// Client role constants (role of client in the case)
const (
	ClientRoleDemandante = "DEMANDANTE" // Plaintiff - initiates the legal action
	ClientRoleDemandado  = "DEMANDADO"  // Defendant - responds to the legal action
)

// Case represents a legal case
type Case struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Firm relationship
	FirmID string `gorm:"type:uuid;not null;index:idx_case_firm_opened;index:idx_case_firm_status;uniqueIndex:idx_firm_filing_number" json:"firm_id"`
	Firm   Firm   `gorm:"foreignKey:FirmID" json:"firm,omitempty"`

	// Client relationship (User with role 'client')
	ClientID string `gorm:"type:uuid;not null;index" json:"client_id"`
	Client   User   `gorm:"foreignKey:ClientID" json:"client,omitempty"`

	// Case identification
	CaseNumber   string  `gorm:"not null;uniqueIndex" json:"case_number"`
	Title        *string `json:"title,omitempty"` // Brief case title for identification
	CaseType     string  `gorm:"not null" json:"case_type"`
	Description  string  `gorm:"type:text;not null" json:"description"`
	FilingNumber *string `gorm:"size:100;uniqueIndex:idx_firm_filing_number" json:"filing_number,omitempty"` // External filing number from court/process

	// Client's role in the case (demandante/demandado)
	ClientRole *string `gorm:"size:20" json:"client_role,omitempty"`

	// Status and lifecycle
	Status          string     `gorm:"not null;default:OPEN;index:idx_case_firm_status" json:"status"`
	OpenedAt        time.Time  `gorm:"not null;index:idx_case_firm_opened" json:"opened_at"`
	ClosedAt        *time.Time `json:"closed_at,omitempty"`
	StatusChangedAt *time.Time `gorm:"index:idx_case_firm_status_changed" json:"status_changed_at,omitempty"`
	StatusChangedBy *string    `gorm:"type:uuid" json:"status_changed_by,omitempty"`

	// Assignment
	AssignedToID *string `gorm:"type:uuid" json:"assigned_to_id,omitempty"`
	AssignedTo   *User   `gorm:"foreignKey:AssignedToID" json:"assigned_to,omitempty"`

	// Classification (Module B)
	DomainID *string     `gorm:"type:uuid;index:idx_case_firm_domain_branch" json:"domain_id,omitempty"`
	Domain   *CaseDomain `gorm:"foreignKey:DomainID" json:"domain,omitempty"`

	BranchID *string     `gorm:"type:uuid;index:idx_case_firm_domain_branch;index:idx_case_branch_status" json:"branch_id,omitempty"`
	Branch   *CaseBranch `gorm:"foreignKey:BranchID" json:"branch,omitempty"`

	ClassifiedAt *time.Time `json:"classified_at,omitempty"`
	ClassifiedBy *string    `gorm:"type:uuid" json:"classified_by,omitempty"`

	// Soft delete tracking
	IsDeleted  bool       `gorm:"not null;default:false" json:"is_deleted"`
	DeletedAt2 *time.Time `json:"deleted_at_custom,omitempty"` // Custom deleted timestamp (separate from GORM's DeletedAt)
	DeletedBy  *string    `gorm:"type:uuid" json:"deleted_by,omitempty"`

	// Historical case tracking (for migrating paper cases)
	IsHistorical         bool       `gorm:"not null;default:false;index" json:"is_historical"`
	OriginalFilingDate   *time.Time `json:"original_filing_date,omitempty"`
	HistoricalCaseNumber *string    `json:"historical_case_number,omitempty"` // Original paper case reference
	MigrationNotes       *string    `gorm:"type:text" json:"migration_notes,omitempty"`
	MigratedAt           *time.Time `json:"migrated_at,omitempty"`
	MigratedBy           *string    `gorm:"type:uuid" json:"migrated_by,omitempty"`

	// Relationships
	StatusChanger *User          `gorm:"foreignKey:StatusChangedBy" json:"status_changer,omitempty"`
	Classifier    *User          `gorm:"foreignKey:ClassifiedBy" json:"classifier,omitempty"`
	Deleter       *User          `gorm:"foreignKey:DeletedBy" json:"deleter,omitempty"`
	Migrator      *User          `gorm:"foreignKey:MigratedBy" json:"migrator,omitempty"`
	Subtypes      []CaseSubtype  `gorm:"many2many:case_subtypes_junction;" json:"subtypes,omitempty"`
	Documents     []CaseDocument `gorm:"foreignKey:CaseID" json:"documents,omitempty"`
	Milestones    []CaseMilestone `gorm:"foreignKey:CaseID" json:"milestones,omitempty"`
	Collaborators []User         `gorm:"many2many:case_collaborators;" json:"collaborators,omitempty"`
	OpposingParty *CaseParty     `gorm:"foreignKey:CaseID" json:"opposing_party,omitempty"`
}

// BeforeCreate hook to generate UUID and set OpenedAt
func (c *Case) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	if c.OpenedAt.IsZero() {
		c.OpenedAt = time.Now()
	}
	return nil
}

// TableName specifies the table name for Case model
func (Case) TableName() string {
	return "cases"
}

// GetBranchDisplayName returns human-readable branch name
func (c *Case) GetBranchDisplayName() string {
	if c.Branch != nil {
		return c.Branch.Name
	}
	return ""
}

// GetDomainDisplayName returns human-readable domain name
func (c *Case) GetDomainDisplayName() string {
	if c.Domain != nil {
		return c.Domain.Name
	}
	return ""
}

// IsOpen checks if the case is open
func (c *Case) IsOpen() bool {
	return c.Status == CaseStatusOpen
}

// IsClosed checks if the case is closed
func (c *Case) IsClosed() bool {
	return c.Status == CaseStatusClosed
}

// IsOnHold checks if the case is on hold
func (c *Case) IsOnHold() bool {
	return c.Status == CaseStatusOnHold
}

// IsValidStatus checks if the status is valid
func IsValidCaseStatus(status string) bool {
	validStatuses := []string{
		CaseStatusOpen,
		CaseStatusOnHold,
		CaseStatusClosed,
	}
	for _, s := range validStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// IsValidClientRole checks if the client role is valid
func IsValidClientRole(role string) bool {
	return role == ClientRoleDemandante || role == ClientRoleDemandado
}
