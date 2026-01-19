package services

import (
	"regexp"
	"strings"
)

// variableRegex matches {{variable.path}} patterns
var variableRegex = regexp.MustCompile(`\{\{([a-zA-Z0-9_.]+)\}\}`)

// RenderTemplate replaces {{variable}} placeholders with actual values from TemplateData
func RenderTemplate(content string, data TemplateData) string {
	return variableRegex.ReplaceAllStringFunc(content, func(match string) string {
		// Extract variable key from {{key}}
		key := strings.TrimPrefix(strings.TrimSuffix(match, "}}"), "{{")

		// Look up the value
		value := getValueByKey(key, data)
		if value == "" {
			// Return the original placeholder if no value found
			return match
		}
		return value
	})
}

// getValueByKey retrieves a value from TemplateData using a dot-notation key
func getValueByKey(key string, data TemplateData) string {
	parts := strings.SplitN(key, ".", 2)
	if len(parts) != 2 {
		return ""
	}

	category := parts[0]
	field := parts[1]

	switch category {
	case "client":
		return getClientValue(field, data.Client)
	case "case":
		return getCaseValue(field, data.Case)
	case "firm":
		return getFirmValue(field, data.Firm)
	case "lawyer":
		return getLawyerValue(field, data.Lawyer)
	case "today":
		return getTodayValue(field, data.Today)
	default:
		return ""
	}
}

func getClientValue(field string, client ClientData) string {
	switch field {
	case "name":
		return client.Name
	case "email":
		return client.Email
	case "phone":
		return client.Phone
	case "document_type":
		return client.DocumentType
	case "document_number":
		return client.DocumentNumber
	case "address":
		return client.Address
	default:
		return ""
	}
}

func getCaseValue(field string, caseData CaseData) string {
	switch field {
	case "number":
		return caseData.Number
	case "title":
		return caseData.Title
	case "description":
		return caseData.Description
	case "status":
		return caseData.Status
	case "domain":
		return caseData.Domain
	case "branch":
		return caseData.Branch
	case "subtypes":
		return caseData.Subtypes
	case "opened_at":
		return caseData.OpenedAt
	default:
		return ""
	}
}

func getFirmValue(field string, firm FirmData) string {
	switch field {
	case "name":
		return firm.Name
	case "address":
		return firm.Address
	case "city":
		return firm.City
	case "phone":
		return firm.Phone
	case "billing_email":
		return firm.BillingEmail
	case "info_email":
		return firm.InfoEmail
	default:
		return ""
	}
}

func getLawyerValue(field string, lawyer LawyerData) string {
	switch field {
	case "name":
		return lawyer.Name
	case "email":
		return lawyer.Email
	case "phone":
		return lawyer.Phone
	default:
		return ""
	}
}

func getTodayValue(field string, today DateData) string {
	switch field {
	case "date":
		return today.Date
	case "date_long":
		return today.DateLong
	case "year":
		return today.Year
	default:
		return ""
	}
}

// WrapHTMLForPDF wraps HTML content with legal document styles for PDF generation
func WrapHTMLForPDF(content string) string {
	return `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        @page {
            margin: 1in;
        }
        body {
            font-family: "Times New Roman", Times, serif;
            font-size: 12pt;
            line-height: 1.5;
            color: #000;
            text-align: justify;
        }
        h1 {
            font-size: 16pt;
            font-weight: bold;
            text-align: center;
            margin-bottom: 24pt;
        }
        h2 {
            font-size: 14pt;
            font-weight: bold;
            margin-top: 18pt;
            margin-bottom: 12pt;
        }
        h3 {
            font-size: 12pt;
            font-weight: bold;
            margin-top: 12pt;
            margin-bottom: 6pt;
        }
        p {
            margin-bottom: 12pt;
            text-indent: 0.5in;
        }
        p:first-of-type {
            text-indent: 0;
        }
        ul, ol {
            margin-left: 0.5in;
            margin-bottom: 12pt;
        }
        li {
            margin-bottom: 6pt;
        }
        .signature-block {
            margin-top: 48pt;
        }
        .signature-line {
            border-top: 1px solid #000;
            width: 3in;
            margin-top: 36pt;
            padding-top: 6pt;
        }
        .date-line {
            margin-top: 24pt;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin-bottom: 12pt;
        }
        th, td {
            border: 1px solid #000;
            padding: 6pt;
            text-align: left;
        }
        th {
            background-color: #f0f0f0;
            font-weight: bold;
        }
    </style>
</head>
<body>
` + content + `
</body>
</html>`
}
