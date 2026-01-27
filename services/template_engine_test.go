package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderTemplate(t *testing.T) {
	data := TemplateData{
		Client: ClientData{
			Name: "John Doe",
		},
		Case: CaseData{
			Number: "2026-001",
			Title:  "Test Case",
		},
		Firm: FirmData{
			Name: "Law Flow Firm",
		},
		Lawyer: LawyerData{
			Name: "Jane Smith, Esq.",
		},
		Today: DateData{
			Date: "2026-01-27",
		},
	}

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "Single variable",
			content:  "Hello {{client.name}}",
			expected: "Hello John Doe",
		},
		{
			name:     "Multiple variables",
			content:  "Case {{case.number}}: {{case.title}}",
			expected: "Case 2026-001: Test Case",
		},
		{
			name:     "Variables with whitespace",
			content:  "{{  firm.name  }}",
			expected: "Law Flow Firm",
		},
		{
			name:     "Missing variable",
			content:  "Missing: {{client.missing_field}}",
			expected: "Missing: ",
		},
		{
			name:     "Invalid category",
			content:  "Invalid: {{unknown.name}}",
			expected: "Invalid: ",
		},
		{
			name:     "Malformed tag",
			content:  "Malformed: {{client.name",
			expected: "Malformed: {{client.name",
		},
		{
			name:     "Today data",
			content:  "Date: {{today.date}}",
			expected: "Date: 2026-01-27",
		},
		{
			name:     "Lawyer data",
			content:  "Lawyer: {{lawyer.name}}",
			expected: "Lawyer: Jane Smith, Esq.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderTemplate(tt.content, data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTemplateEngine_WrapHTMLForPDF(t *testing.T) {
	content := "<h1>Doc Title</h1><p>Body content</p>"
	wrapped := WrapHTMLForPDF(content)

	assert.Contains(t, wrapped, "<!DOCTYPE html>")
	assert.Contains(t, wrapped, "font-family: \"Times New Roman\"")
	assert.Contains(t, wrapped, content)
	assert.Contains(t, wrapped, "</body>")
}

func TestGetValueByKey(t *testing.T) {
	data := TemplateData{
		Client: ClientData{Name: "Client Name"},
		Case:   CaseData{Number: "CASE-123"},
		Firm:   FirmData{Name: "Firm Name"},
		Lawyer: LawyerData{Name: "Lawyer Name"},
		Today:  DateData{Date: "2026-02-02"},
	}

	assert.Equal(t, "Client Name", getValueByKey("client.name", data))
	assert.Equal(t, "CASE-123", getValueByKey("case.number", data))
	assert.Equal(t, "Firm Name", getValueByKey("firm.name", data))
	assert.Equal(t, "Lawyer Name", getValueByKey("lawyer.name", data))
	assert.Equal(t, "2026-02-02", getValueByKey("today.date", data))

	// Edge cases
	assert.Equal(t, "", getValueByKey("client", data))         // Incomplete key
	assert.Equal(t, "", getValueByKey("unknown.field", data))  // Unknown category
	assert.Equal(t, "", getValueByKey("client.unknown", data)) // Unknown field
}
