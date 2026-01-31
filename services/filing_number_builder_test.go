package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildFilingNumber(t *testing.T) {
	input := FilingNumberInput{
		CourtOfficeCode: "110014003001",
		Year:            2026,
		ProcessCode:     "123",
		ResourceCode:    "1",
	}

	result := BuildFilingNumber(input)
	assert.Equal(t, 23, len(result))
	assert.Equal(t, "11001400300120260012301", result)
}

func TestParseFilingNumber(t *testing.T) {
	t.Run("Valid Number", func(t *testing.T) {
		num := "11001400300120260012301"
		comp, err := ParseFilingNumber(num)
		assert.NoError(t, err)
		assert.Equal(t, "11", comp.DepartmentCode)
		assert.Equal(t, "001", comp.CityCode)
		assert.Equal(t, "40", comp.EntityCode)
		assert.Equal(t, "03", comp.SpecialtyCode)
		assert.Equal(t, "001", comp.CourtOfficeCode)
		assert.Equal(t, "2026", comp.Year)
		assert.Equal(t, "00123", comp.ProcessCode)
		assert.Equal(t, "01", comp.ResourceCode)
	})

	t.Run("Invalid Length", func(t *testing.T) {
		_, err := ParseFilingNumber("123")
		assert.Error(t, err)
	})
}

func TestValidateFilingNumberInput(t *testing.T) {
	t.Run("Valid Input", func(t *testing.T) {
		input := FilingNumberInput{
			CourtOfficeCode: "110014003001",
			Year:            2026,
			ProcessCode:     "123",
			ResourceCode:    "01",
		}
		errors := ValidateFilingNumberInput(input)
		assert.Empty(t, errors)
	})

	t.Run("Invalid Input", func(t *testing.T) {
		input := FilingNumberInput{
			Year:         1800,
			ProcessCode:  "ABC",
			ResourceCode: "999",
		}
		errors := ValidateFilingNumberInput(input)
		assert.NotEmpty(t, errors)
		assert.Contains(t, errors, "Court office code is required")
		assert.Contains(t, errors, "Year must be between 1900 and 2100")
		assert.Contains(t, errors, "Process code must contain only digits")
		assert.Contains(t, errors, "Resource code must be at most 2 digits")
	})
}
