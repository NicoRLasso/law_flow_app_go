package services

import (
	"context"
	"fmt"
	"law_flow_app_go/models"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSearchTestDB() *gorm.DB {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=busy_timeout(5000)", uuid.New().String())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&models.Case{}, &models.User{}, &models.CaseParty{}, &models.CaseLog{}, &models.CaseDocument{}, &models.Firm{})
	return db
}

func TestSanitizeFTSQuery(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello*"},
		{"hello world", "hello* OR world*"},
		{"a", ""},                          // Too short
		{"legal case*", "legal* OR case*"}, // Special char removed
		{"", ""},
		{"   multiple   spaces   ", "multiple* OR spaces*"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, sanitizeFTSQuery(tt.input))
		})
	}
}

func TestSearchRoles(t *testing.T) {
	db := setupSearchTestDB()
	InitializeFTS5(db)

	firmID := "firm-search"
	client1ID := "c1"
	client2ID := "c2"
	lawyerID := "l1"
	adminID := "a1"

	db.Create(&models.Firm{ID: firmID, Name: "Search Firm"})
	db.Create(&models.User{ID: client1ID, FirmID: &firmID, Role: "client", Name: "Client One"})
	db.Create(&models.User{ID: client2ID, FirmID: &firmID, Role: "client", Name: "Client Two"})
	db.Create(&models.User{ID: lawyerID, FirmID: &firmID, Role: "lawyer", Name: "Lawyer One"})

	// Case 1: Client 1's case
	db.Create(&models.Case{
		ID: "case1", FirmID: firmID, ClientID: client1ID, CaseNumber: "NUM-001",
		Title: stringToPtr("Divorce Case"), Description: "Description of divorce",
	})

	// Case 2: Client 2's case, Lawyer is collaborator
	c2 := &models.Case{
		ID: "case2", FirmID: firmID, ClientID: client2ID, CaseNumber: "NUM-002",
		Title: stringToPtr("Corporate Case"), Description: "Description of corporate",
	}
	db.Create(c2)
	db.Model(c2).Association("Collaborators").Append(&models.User{ID: lawyerID})

	searchSvc := NewSearchService(db)
	ctx := context.Background()

	t.Run("Client 1 only sees their own case", func(t *testing.T) {
		results, err := searchSvc.SearchWithRoleFilter(ctx, firmID, client1ID, "client", "Case", 10)
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "case1", results[0].CaseID)
	})

	t.Run("Lawyer sees cases where they collaborate", func(t *testing.T) {
		results, err := searchSvc.SearchWithRoleFilter(ctx, firmID, lawyerID, "lawyer", "Corporate", 10)
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "case2", results[0].CaseID)
	})

	t.Run("Admin sees all cases in firm", func(t *testing.T) {
		results, err := searchSvc.SearchWithRoleFilter(ctx, firmID, adminID, "admin", "Case", 10)
		assert.NoError(t, err)
		assert.Len(t, results, 2)
	})
}

func TestDetermineMatchSource(t *testing.T) {
	assert.Equal(t, "document", determineMatchSource("This is a <mark>contract.pdf</mark> snippet"))
	assert.Equal(t, "case", determineMatchSource("This is a <mark>divorce</mark> title"))
}

func TestProcessSnippet(t *testing.T) {
	expected := "Check out &amp; <mark>this</mark> snippet"
	assert.Equal(t, expected, processSnippet("Check out & <mark>this</mark> snippet"))
}
