package services

import (
	"fmt"
	"law_flow_app_go/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCaseTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&models.Case{}, &models.Firm{})
	return db
}

func stringPtr(s string) *string {
	return &s
}

func TestGenerateCaseNumber(t *testing.T) {
	db := setupCaseTestDB()
	firmID := "firm-case"
	slug := "LEX"

	db.Create(&models.Firm{ID: firmID, Name: "Lex Firm", Slug: slug})

	year := time.Now().Year()

	// 1. Test First Case
	number, err := GenerateCaseNumber(db, firmID)
	assert.NoError(t, err)
	expectedFirst := fmt.Sprintf("%s-%d-00001", slug, year)
	assert.Equal(t, expectedFirst, number)

	// 2. Create the first case and test increment
	db.Create(&models.Case{
		FirmID:     firmID,
		CaseNumber: number,
		Title:      stringPtr("Case 1"),
	})

	number2, err := GenerateCaseNumber(db, firmID)
	assert.NoError(t, err)
	expectedSecond := fmt.Sprintf("%s-%d-00002", slug, year)
	assert.Equal(t, expectedSecond, number2)
}

func TestEnsureUniqueCaseNumber(t *testing.T) {
	db := setupCaseTestDB()
	firmID := "firm-unique"
	slug := "UNI"

	db.Create(&models.Firm{ID: firmID, Name: "Unique Firm", Slug: slug})

	year := time.Now().Year()
	expected := fmt.Sprintf("%s-%d-00001", slug, year)

	// Execute
	number, err := EnsureUniqueCaseNumber(db, firmID)
	assert.NoError(t, err)
	assert.Equal(t, expected, number)

	// Verify it still works after one is created
	db.Create(&models.Case{
		FirmID:     firmID,
		CaseNumber: number,
		Title:      stringPtr("First"),
	})

	number2, err := EnsureUniqueCaseNumber(db, firmID)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%s-%d-00002", slug, year), number2)
}

func TestGenerateCaseNumber_FirmNotFound(t *testing.T) {
	db := setupCaseTestDB()
	_, err := GenerateCaseNumber(db, "non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch firm")
}
