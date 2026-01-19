package services

import (
	"law_flow_app_go/models"
	"strings"
	"time"
)

// VariableCategory represents a group of template variables
type VariableCategory struct {
	Name      string     `json:"name"`
	NameKey   string     `json:"name_key"` // i18n key
	Variables []Variable `json:"variables"`
}

// Variable represents a single template variable
type Variable struct {
	Key         string `json:"key"`         // e.g., "client.name"
	Label       string `json:"label"`       // Display name
	LabelKey    string `json:"label_key"`   // i18n key
	Description string `json:"description"` // Help text
	Example     string `json:"example"`     // Example value
}

// TemplateData holds all data for template variable substitution
type TemplateData struct {
	Client ClientData `json:"client"`
	Case   CaseData   `json:"case"`
	Firm   FirmData   `json:"firm"`
	Lawyer LawyerData `json:"lawyer"`
	Today  DateData   `json:"today"`
}

// ClientData holds client-related template data
type ClientData struct {
	Name           string `json:"name"`
	Email          string `json:"email"`
	Phone          string `json:"phone"`
	DocumentType   string `json:"document_type"`
	DocumentNumber string `json:"document_number"`
	Address        string `json:"address"`
}

// CaseData holds case-related template data
type CaseData struct {
	Number      string `json:"number"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Domain      string `json:"domain"`
	Branch      string `json:"branch"`
	Subtypes    string `json:"subtypes"`
	OpenedAt    string `json:"opened_at"`
}

// FirmData holds firm-related template data
type FirmData struct {
	Name         string `json:"name"`
	Address      string `json:"address"`
	City         string `json:"city"`
	Phone        string `json:"phone"`
	BillingEmail string `json:"billing_email"`
	InfoEmail    string `json:"info_email"`
}

// LawyerData holds assigned lawyer template data
type LawyerData struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

// DateData holds current date template data
type DateData struct {
	Date     string `json:"date"`      // Short format: 2006-01-02
	DateLong string `json:"date_long"` // Long format: January 2, 2006
	Year     string `json:"year"`
}

// GetVariableDictionary returns all available template variables organized by category
func GetVariableDictionary() []VariableCategory {
	return []VariableCategory{
		{
			Name:    "Client",
			NameKey: "templates.variables.client",
			Variables: []Variable{
				{Key: "client.name", Label: "Client Name", LabelKey: "templates.variables.client_name", Example: "John Doe"},
				{Key: "client.email", Label: "Client Email", LabelKey: "templates.variables.client_email", Example: "john@example.com"},
				{Key: "client.phone", Label: "Client Phone", LabelKey: "templates.variables.client_phone", Example: "+1 555-123-4567"},
				{Key: "client.document_type", Label: "Document Type", LabelKey: "templates.variables.client_doc_type", Example: "DNI"},
				{Key: "client.document_number", Label: "Document Number", LabelKey: "templates.variables.client_doc_number", Example: "12345678"},
				{Key: "client.address", Label: "Client Address", LabelKey: "templates.variables.client_address", Example: "123 Main St, City"},
			},
		},
		{
			Name:    "Case",
			NameKey: "templates.variables.case",
			Variables: []Variable{
				{Key: "case.number", Label: "Case Number", LabelKey: "templates.variables.case_number", Example: "2026-001"},
				{Key: "case.title", Label: "Case Title", LabelKey: "templates.variables.case_title", Example: "Smith vs. Jones"},
				{Key: "case.description", Label: "Case Description", LabelKey: "templates.variables.case_description", Example: "Contract dispute..."},
				{Key: "case.status", Label: "Case Status", LabelKey: "templates.variables.case_status", Example: "Open"},
				{Key: "case.domain", Label: "Legal Domain", LabelKey: "templates.variables.case_domain", Example: "Civil"},
				{Key: "case.branch", Label: "Legal Branch", LabelKey: "templates.variables.case_branch", Example: "Family Law"},
				{Key: "case.subtypes", Label: "Case Subtypes", LabelKey: "templates.variables.case_subtypes", Example: "Divorce, Custody"},
				{Key: "case.opened_at", Label: "Opened Date", LabelKey: "templates.variables.case_opened_at", Example: "January 15, 2026"},
			},
		},
		{
			Name:    "Firm",
			NameKey: "templates.variables.firm",
			Variables: []Variable{
				{Key: "firm.name", Label: "Firm Name", LabelKey: "templates.variables.firm_name", Example: "Smith & Associates"},
				{Key: "firm.address", Label: "Firm Address", LabelKey: "templates.variables.firm_address", Example: "456 Law St"},
				{Key: "firm.city", Label: "Firm City", LabelKey: "templates.variables.firm_city", Example: "New York"},
				{Key: "firm.phone", Label: "Firm Phone", LabelKey: "templates.variables.firm_phone", Example: "+1 555-987-6543"},
				{Key: "firm.billing_email", Label: "Billing Email", LabelKey: "templates.variables.firm_billing_email", Example: "billing@firm.com"},
				{Key: "firm.info_email", Label: "Info Email", LabelKey: "templates.variables.firm_info_email", Example: "info@firm.com"},
			},
		},
		{
			Name:    "Lawyer",
			NameKey: "templates.variables.lawyer",
			Variables: []Variable{
				{Key: "lawyer.name", Label: "Lawyer Name", LabelKey: "templates.variables.lawyer_name", Example: "Jane Smith, Esq."},
				{Key: "lawyer.email", Label: "Lawyer Email", LabelKey: "templates.variables.lawyer_email", Example: "jane@firm.com"},
				{Key: "lawyer.phone", Label: "Lawyer Phone", LabelKey: "templates.variables.lawyer_phone", Example: "+1 555-111-2222"},
			},
		},
		{
			Name:    "Dates",
			NameKey: "templates.variables.dates",
			Variables: []Variable{
				{Key: "today.date", Label: "Today's Date", LabelKey: "templates.variables.today_date", Example: "2026-01-19"},
				{Key: "today.date_long", Label: "Today (Long)", LabelKey: "templates.variables.today_date_long", Example: "January 19, 2026"},
				{Key: "today.year", Label: "Current Year", LabelKey: "templates.variables.today_year", Example: "2026"},
			},
		},
	}
}

// BuildTemplateDataFromCase extracts all template data from a case and its relationships
func BuildTemplateDataFromCase(caseRecord *models.Case, firm *models.Firm) TemplateData {
	data := TemplateData{
		Today: DateData{
			Date:     time.Now().Format("2006-01-02"),
			DateLong: time.Now().Format("January 2, 2006"),
			Year:     time.Now().Format("2006"),
		},
	}

	// Client data
	if caseRecord.Client.ID != "" {
		docType := ""
		if caseRecord.Client.DocumentType != nil {
			docType = caseRecord.Client.DocumentType.Label
		}
		data.Client = ClientData{
			Name:           caseRecord.Client.Name,
			Email:          caseRecord.Client.Email,
			Phone:          safeString(caseRecord.Client.PhoneNumber),
			DocumentType:   docType,
			DocumentNumber: safeString(caseRecord.Client.DocumentNumber),
			Address:        safeString(caseRecord.Client.Address),
		}
	}

	// Case data
	data.Case = CaseData{
		Number:      caseRecord.CaseNumber,
		Title:       safeStringPtr(caseRecord.Title),
		Description: caseRecord.Description,
		Status:      caseRecord.Status,
		OpenedAt:    caseRecord.OpenedAt.Format("January 2, 2006"),
	}

	if caseRecord.Domain != nil {
		data.Case.Domain = caseRecord.Domain.Name
	}
	if caseRecord.Branch != nil {
		data.Case.Branch = caseRecord.Branch.Name
	}

	// Subtypes as comma-separated string
	if len(caseRecord.Subtypes) > 0 {
		subtypeNames := make([]string, len(caseRecord.Subtypes))
		for i, st := range caseRecord.Subtypes {
			subtypeNames[i] = st.Name
		}
		data.Case.Subtypes = strings.Join(subtypeNames, ", ")
	}

	// Firm data (fields are non-pointer strings)
	if firm != nil {
		data.Firm = FirmData{
			Name:         firm.Name,
			Address:      firm.Address,
			City:         firm.City,
			Phone:        firm.Phone,
			BillingEmail: firm.BillingEmail,
			InfoEmail:    firm.InfoEmail,
		}
	}

	// Lawyer data (assigned lawyer)
	if caseRecord.AssignedTo != nil {
		data.Lawyer = LawyerData{
			Name:  caseRecord.AssignedTo.Name,
			Email: caseRecord.AssignedTo.Email,
			Phone: safeString(caseRecord.AssignedTo.PhoneNumber),
		}
	}

	return data
}

// Helper to safely get string from pointer
func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Helper to safely get string from pointer (different name for clarity)
func safeStringPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
