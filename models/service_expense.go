package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Expense status constants (workflow states - must remain fixed)
const (
	ExpenseStatusPending  = "PENDING"
	ExpenseStatusApproved = "APPROVED"
	ExpenseStatusPaid     = "PAID"
	ExpenseStatusRejected = "REJECTED"
)

// ChoiceCategory key for expense categories
const (
	ChoiceCategoryKeyExpenseCategory = "expense_category" // Category key for expense categories
)

// ServiceExpense tracks reimbursable costs for a legal service
type ServiceExpense struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Scoping
	FirmID    string       `gorm:"type:uuid;not null;index" json:"firm_id"`
	Firm      Firm         `gorm:"foreignKey:FirmID" json:"firm,omitempty"`
	ServiceID string       `gorm:"type:uuid;not null;index:idx_expense_service" json:"service_id"`
	Service   LegalService `gorm:"foreignKey:ServiceID" json:"service,omitempty"`

	// Expense category (references ChoiceOption)
	ExpenseCategoryID *string       `gorm:"column:category_id;type:uuid;index" json:"category_id,omitempty"`
	Category          *ChoiceOption `gorm:"foreignKey:ExpenseCategoryID" json:"category,omitempty"`

	// Expense details
	Description string  `gorm:"not null" json:"description"`
	Amount      float64 `gorm:"not null" json:"amount"`
	Currency    string  `gorm:"not null;default:USD" json:"currency"`

	// Date tracking
	IncurredAt time.Time `gorm:"not null" json:"incurred_at"`

	// Status
	Status     string     `gorm:"not null;default:PENDING;index" json:"status"`
	ApprovedAt *time.Time `json:"approved_at,omitempty"`
	ApprovedBy *string    `gorm:"type:uuid" json:"approved_by,omitempty"`

	// Receipt attachment (optional)
	ReceiptDocumentID *string          `gorm:"type:uuid" json:"receipt_document_id,omitempty"`
	ReceiptDocument   *ServiceDocument `gorm:"foreignKey:ReceiptDocumentID" json:"receipt_document,omitempty"`

	// Who recorded this expense
	RecordedByID string `gorm:"type:uuid;not null" json:"recorded_by_id"`
	RecordedBy   User   `gorm:"foreignKey:RecordedByID" json:"recorded_by,omitempty"`

	// Relationships
	Approver *User `gorm:"foreignKey:ApprovedBy" json:"approver,omitempty"`
}

// BeforeCreate hook to generate UUID
func (e *ServiceExpense) BeforeCreate(tx *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name for ServiceExpense model
func (ServiceExpense) TableName() string {
	return "service_expenses"
}

// GetCategoryLabel returns the category label or empty string
func (e *ServiceExpense) GetCategoryLabel() string {
	if e.Category != nil {
		return e.Category.Label
	}
	return ""
}

// GetCategoryCode returns the category code or empty string
func (e *ServiceExpense) GetCategoryCode() string {
	if e.Category != nil {
		return e.Category.Code
	}
	return ""
}

// IsPending checks if the expense is pending approval
func (e *ServiceExpense) IsPending() bool {
	return e.Status == ExpenseStatusPending
}

// IsApproved checks if the expense is approved
func (e *ServiceExpense) IsApproved() bool {
	return e.Status == ExpenseStatusApproved
}

// IsPaid checks if the expense has been paid/reimbursed
func (e *ServiceExpense) IsPaid() bool {
	return e.Status == ExpenseStatusPaid
}

// IsRejected checks if the expense was rejected
func (e *ServiceExpense) IsRejected() bool {
	return e.Status == ExpenseStatusRejected
}

// IsValidExpenseStatus checks if the status is valid
func IsValidExpenseStatus(status string) bool {
	validStatuses := []string{
		ExpenseStatusPending,
		ExpenseStatusApproved,
		ExpenseStatusPaid,
		ExpenseStatusRejected,
	}
	for _, s := range validStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// GetExpenseStatusDisplayName returns human-readable status name
func GetExpenseStatusDisplayName(status string) string {
	names := map[string]string{
		ExpenseStatusPending:  "Pending Approval",
		ExpenseStatusApproved: "Approved",
		ExpenseStatusPaid:     "Paid",
		ExpenseStatusRejected: "Rejected",
	}
	if name, ok := names[status]; ok {
		return name
	}
	return status
}
