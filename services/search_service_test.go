package services

import (
	"context"
	"law_flow_app_go/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSearchTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(
		&models.Firm{},
		&models.User{},
		&models.Case{},
		&models.LegalService{},
		&models.CaseDocument{},
		&models.ServiceDocument{},
		&models.CaseLog{},
		&models.ServiceMilestone{},
		&models.CaseParty{},
	)
	assert.NoError(t, err)

	err = InitializeFTS5(db)
	assert.NoError(t, err)

	return db
}

func TestSearchService(t *testing.T) {
	db := setupSearchTestDB(t)
	s := NewSearchService(db)
	ctx := context.Background()
	firmID := "firm-search"
	db.Create(&models.Firm{ID: firmID, Name: "Search B.V.", Slug: "search"})

	// Create a user/client
	client := models.User{ID: "client-1", Name: "John Searcher", FirmID: &firmID}
	db.Create(&client)

	t.Run("Search cases", func(t *testing.T) {
		db.Create(&models.Case{
			ID:         "case-1",
			FirmID:     firmID,
			CaseNumber: "CASE-001",
			Title:      stringToPtr("Divorce of Smith"),
			ClientID:   client.ID,
			OpenedAt:   time.Now(),
		})

		// Search
		results, err := s.Search(ctx, firmID, "Smith", 10)
		assert.NoError(t, err)
		assert.NotEmpty(t, results)
		assert.Equal(t, "case", results[0].Type)
		assert.Contains(t, results[0].CaseTitle, "Smith")
	})

	t.Run("Search services", func(t *testing.T) {
		db.Create(&models.LegalService{
			ID:            "service-1",
			FirmID:        firmID,
			ServiceNumber: "SVC-001",
			Title:         "Contract Review",
			ClientID:      client.ID,
		})

		results, err := s.Search(ctx, firmID, "Contract", 10)
		assert.NoError(t, err)
		if assert.NotEmpty(t, results) {
			assert.Equal(t, "service", results[0].Type)
			assert.Contains(t, results[0].ServiceTitle, "Contract")
		}
	})

	t.Run("Search with role filter - client", func(t *testing.T) {
		results, err := s.SearchWithRoleFilter(ctx, firmID, client.ID, "client", "Smith", 10)
		assert.NoError(t, err)
		if assert.NotEmpty(t, results) {
			assert.Equal(t, "case", results[0].Type)
		}
	})

	t.Run("Search with role filter - lawyer (forbidden case)", func(t *testing.T) {
		lawyerID := "lawyer-1"
		db.Create(&models.User{ID: lawyerID, Name: "Lawyer Lara", FirmID: &firmID})

		results, err := s.SearchWithRoleFilter(ctx, firmID, lawyerID, "lawyer", "Smith", 10)
		assert.NoError(t, err)
		assert.Empty(t, results) // Lara is not assigned to Smith's case
	})
}
