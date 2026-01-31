package services

import (
	"law_flow_app_go/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupChoiceTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&models.ChoiceCategory{}, &models.ChoiceOption{}, &models.Firm{})
	return db
}

func TestChoiceRetrieval(t *testing.T) {
	db := setupChoiceTestDB()
	firmID := "firm-choice"

	db.Create(&models.Firm{ID: firmID, Name: "Choice Firm"})

	// Seed
	cat := models.ChoiceCategory{FirmID: firmID, Key: "status", Name: "Status", IsActive: true}
	db.Create(&cat)

	opt1 := models.ChoiceOption{CategoryID: cat.ID, Code: "OPEN", Label: "Open", SortOrder: 1, IsActive: true}
	opt2 := models.ChoiceOption{CategoryID: cat.ID, Code: "CLOSED", Label: "Closed", SortOrder: 2, IsActive: true}
	opt3 := models.ChoiceOption{CategoryID: cat.ID, Code: "PENDING", Label: "Pending", SortOrder: 3, IsActive: false}
	db.Create(&opt1)
	db.Create(&opt2)
	db.Create(&opt3)
	// Force IsActive to false because of GORM default:true tag
	db.Model(&opt3).Update("is_active", false)

	t.Run("GetChoiceOptions", func(t *testing.T) {
		options, err := GetChoiceOptions(db, firmID, "status")
		assert.NoError(t, err)
		assert.Len(t, options, 2)
		assert.Equal(t, "OPEN", options[0].Code)
		assert.Equal(t, "CLOSED", options[1].Code)
	})

	t.Run("GetChoiceOptionByCode", func(t *testing.T) {
		option, err := GetChoiceOptionByCode(db, firmID, "status", "OPEN")
		assert.NoError(t, err)
		assert.Equal(t, opt1.ID, option.ID)

		_, err = GetChoiceOptionByCode(db, firmID, "status", "NONEXISTENT")
		assert.Error(t, err)
	})

	t.Run("ValidateChoiceOption", func(t *testing.T) {
		assert.True(t, ValidateChoiceOption(db, firmID, "status", "OPEN"))
		assert.True(t, ValidateChoiceOption(db, firmID, "status", "CLOSED"))
		assert.False(t, ValidateChoiceOption(db, firmID, "status", "PENDING")) // Inactive
		assert.False(t, ValidateChoiceOption(db, firmID, "status", "MISSING"))
	})
}

func TestSeedDefaultChoices(t *testing.T) {
	db := setupChoiceTestDB()
	firmID := "firm-seed"

	// Test Default (Priority, Service Type, Expense Category, Currency)
	err := SeedDefaultChoices(db, firmID, "Generic")
	assert.NoError(t, err)

	var catCount int64
	db.Model(&models.ChoiceCategory{}).Where("firm_id = ?", firmID).Count(&catCount)
	assert.Equal(t, int64(4), catCount) // Priority, Service Type, Expense Category, Currency

	// Test Colombia (Previous 4 + Document Type)
	err = SeedDefaultChoices(db, firmID, "Colombia")
	assert.NoError(t, err)

	db.Model(&models.ChoiceCategory{}).Where("firm_id = ?", firmID).Count(&catCount)
	assert.Equal(t, int64(5), catCount) // 4 + Document Type

	var docTypeCat models.ChoiceCategory
	db.Where("firm_id = ? AND key = ?", firmID, "document_type").First(&docTypeCat)

	var optCount int64
	db.Model(&models.ChoiceOption{}).Where("category_id = ?", docTypeCat.ID).Count(&optCount)
	assert.Equal(t, int64(5), optCount)
}
