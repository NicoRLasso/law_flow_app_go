package services

import (
	"context"
	"law_flow_app_go/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBuildTemplateDataFromCase(t *testing.T) {
	now := time.Now()
	firm := &models.Firm{
		Name:    "Firm Name",
		Address: "Firm Address",
		City:    "Firm City",
		Phone:   "555-FIRM",
	}

	client := models.User{
		ID:             "u1",
		Name:           "John Client",
		Email:          "john@client.com",
		PhoneNumber:    stringToPtr("111-222"),
		DocumentType:   &models.ChoiceOption{Label: "CC"},
		DocumentNumber: stringToPtr("998877"),
		Address:        stringToPtr("Client St 123"),
	}

	lawyer := models.User{
		Name:        "Jane Lawyer",
		Email:       "jane@firm.com",
		PhoneNumber: stringToPtr("333-444"),
	}

	caseRecord := &models.Case{
		CaseNumber:  "CASE-2026-X",
		Title:       stringToPtr("Case Title"),
		Description: "Detailed description",
		Status:      "OPEN",
		OpenedAt:    now,
		Client:      client,
		AssignedTo:  &lawyer,
		Domain:      &models.CaseDomain{Name: "Domain Name"},
		Branch:      &models.CaseBranch{Name: "Branch Name"},
		Subtypes: []models.CaseSubtype{
			{Name: "Subtype A"},
			{Name: "Subtype B"},
		},
	}

	t.Run("Full data extraction", func(t *testing.T) {
		data := BuildTemplateDataFromCase(caseRecord, firm)

		// Assert Firm
		assert.Equal(t, firm.Name, data.Firm.Name)
		assert.Equal(t, firm.Address, data.Firm.Address)

		// Assert Client
		assert.Equal(t, client.Name, data.Client.Name)
		assert.Equal(t, "CC", data.Client.DocumentType)
		assert.Equal(t, "998877", data.Client.DocumentNumber)

		// Assert Case
		assert.Equal(t, caseRecord.CaseNumber, data.Case.Number)
		assert.Equal(t, "Case Title", data.Case.Title)
		assert.Equal(t, "Domain Name", data.Case.Domain)
		assert.Equal(t, "Subtype A, Subtype B", data.Case.Subtypes)
		assert.Equal(t, now.Format("January 2, 2006"), data.Case.OpenedAt)

		// Assert Lawyer
		assert.Equal(t, lawyer.Name, data.Lawyer.Name)

		// Assert Today
		assert.Equal(t, now.Format("2006-01-02"), data.Today.Date)
	})

	t.Run("Minimal data (nils everywhere)", func(t *testing.T) {
		minimalCase := &models.Case{
			CaseNumber: "MIN-001",
			OpenedAt:   now,
		}
		data := BuildTemplateDataFromCase(minimalCase, nil)

		assert.Equal(t, "", data.Firm.Name)
		assert.Equal(t, "", data.Client.Name)
		assert.Equal(t, "MIN-001", data.Case.Number)
		assert.Equal(t, "", data.Case.Title)
		assert.Equal(t, "", data.Case.Domain)
		assert.Equal(t, "", data.Case.Subtypes)
		assert.Equal(t, "", data.Lawyer.Name)
	})
}

func TestGetVariableDictionary(t *testing.T) {
	ctx := context.Background()
	dict := GetVariableDictionary(ctx)

	assert.NotEmpty(t, dict)

	// Simply check if main categories are present
	categories := make(map[string]bool)
	for _, cat := range dict {
		categories[cat.NameKey] = true
	}

	assert.True(t, categories["templates.variables.client"])
	assert.True(t, categories["templates.variables.case"])
	assert.True(t, categories["templates.variables.firm"])
	assert.True(t, categories["templates.variables.lawyer"])
	assert.True(t, categories["templates.variables.dates"])
}

func TestSafeHelpers(t *testing.T) {
	assert.Equal(t, "", safeString(nil))
	s := "test"
	assert.Equal(t, "test", safeString(&s))

	assert.Equal(t, "", safeStringPtr(nil))
	assert.Equal(t, "test", safeStringPtr(&s))
}
