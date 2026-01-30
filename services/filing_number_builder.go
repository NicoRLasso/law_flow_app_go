package services

import (
	"fmt"
	"strings"
	"time"
)

// FilingNumberInput contains the input data for building a filing number
type FilingNumberInput struct {
	CourtOfficeCode string // Full 12-digit code (dept+city+entity+specialty+office)
	Year            int    // 4 digits (YYYY)
	ProcessCode     string // 5 digits (user input)
	ResourceCode    string // 2 digits (user input)
}

// FilingNumberComponents contains the parsed components of a filing number
// Colombian filing number format (23 digits total):
// DD + CCC + EE + SS + OOO + YYYY + PPPPP + RR = 2+3+2+2+3+4+5+2 = 23
type FilingNumberComponents struct {
	DepartmentCode  string // 2 digits (positions 0-1)
	CityCode        string // 3 digits (positions 2-4)
	EntityCode      string // 2 digits (positions 5-6)
	SpecialtyCode   string // 2 digits (positions 7-8)
	CourtOfficeCode string // 3 digits (positions 9-11)
	Year            string // 4 digits (positions 12-15)
	ProcessCode     string // 5 digits (positions 16-20)
	ResourceCode    string // 2 digits (positions 21-22)
}

// BuildFilingNumber constructs a 23-digit filing number from the input
// Format: {court_office_code(12)}{year(4)}{process(5)}{resource(2)} = 23 digits
// The court office code contains: dept(2)+city(3)+entity(2)+specialty(2)+office(3) = 12 digits
func BuildFilingNumber(input FilingNumberInput) string {
	code := strings.TrimSpace(input.CourtOfficeCode)

	// Ensure proper formatting
	year := input.Year
	if year == 0 {
		year = time.Now().Year()
	}

	// Pad process code to 5 digits
	processCode := strings.TrimSpace(input.ProcessCode)
	processCode = fmt.Sprintf("%05s", processCode)
	if len(processCode) > 5 {
		processCode = processCode[:5]
	}

	// Pad resource code to 2 digits
	resourceCode := strings.TrimSpace(input.ResourceCode)
	resourceCode = fmt.Sprintf("%02s", resourceCode)
	if len(resourceCode) > 2 {
		resourceCode = resourceCode[:2]
	}

	return fmt.Sprintf("%s%04d%s%s", code, year, processCode, resourceCode)
}

// ParseFilingNumber parses a filing number string into its components
// Colombian filing number format (23 digits total):
// - Department: 2 digits (positions 0-1)
// - City: 3 digits (positions 2-4)
// - Entity: 2 digits (positions 5-6)
// - Specialty: 2 digits (positions 7-8)
// - Court Office: 3 digits (positions 9-11)
// - Year: 4 digits (positions 12-15)
// - Process Code: 5 digits (positions 16-20)
// - Resource Code: 2 digits (positions 21-22)
func ParseFilingNumber(filingNumber string) (*FilingNumberComponents, error) {
	filingNumber = strings.TrimSpace(filingNumber)

	// Exact length: 23 characters
	// Format: DD + CCC + EE + SS + OOO + YYYY + PPPPP + RR = 2+3+2+2+3+4+5+2 = 23
	if len(filingNumber) != 23 {
		return nil, fmt.Errorf("filing number must be exactly 23 characters, got %d", len(filingNumber))
	}

	// Parse fixed positions
	components := &FilingNumberComponents{
		DepartmentCode:  filingNumber[0:2],   // 2 digits
		CityCode:        filingNumber[2:5],   // 3 digits
		EntityCode:      filingNumber[5:7],   // 2 digits
		SpecialtyCode:   filingNumber[7:9],   // 2 digits
		CourtOfficeCode: filingNumber[9:12],  // 3 digits
		Year:            filingNumber[12:16], // 4 digits
		ProcessCode:     filingNumber[16:21], // 5 digits
		ResourceCode:    filingNumber[21:23], // 2 digits
	}

	return components, nil
}

// ValidateFilingNumberInput validates the input for building a filing number
func ValidateFilingNumberInput(input FilingNumberInput) []string {
	var errors []string

	if strings.TrimSpace(input.CourtOfficeCode) == "" {
		errors = append(errors, "Court office code is required")
	}

	if input.Year < 1900 || input.Year > 2100 {
		errors = append(errors, "Year must be between 1900 and 2100")
	}

	processCode := strings.TrimSpace(input.ProcessCode)
	if processCode == "" {
		errors = append(errors, "Process code is required")
	} else if len(processCode) > 5 {
		errors = append(errors, "Process code must be at most 5 digits")
	} else {
		for _, c := range processCode {
			if c < '0' || c > '9' {
				errors = append(errors, "Process code must contain only digits")
				break
			}
		}
	}

	resourceCode := strings.TrimSpace(input.ResourceCode)
	if resourceCode == "" {
		errors = append(errors, "Resource code is required")
	} else if len(resourceCode) > 2 {
		errors = append(errors, "Resource code must be at most 2 digits")
	} else {
		for _, c := range resourceCode {
			if c < '0' || c > '9' {
				errors = append(errors, "Resource code must contain only digits")
				break
			}
		}
	}

	return errors
}
