package services

import (
	"law_flow_app_go/models"
	"log"

	"gorm.io/gorm"
)

// GetChoiceOptions fetches active choice options for a specific category
func GetChoiceOptions(db *gorm.DB, firmID string, categoryKey string) ([]models.ChoiceOption, error) {
	var options []models.ChoiceOption

	err := db.
		Joins("JOIN choice_categories ON choice_categories.id = choice_options.category_id").
		Where("choice_categories.firm_id = ?", firmID).
		Where("choice_categories.key = ?", categoryKey).
		Where("choice_categories.is_active = ?", true).
		Where("choice_options.is_active = ?", true).
		Order("choice_options.sort_order ASC").
		Find(&options).Error

	return options, err
}

// GetChoiceOptionByCode fetches a specific choice option by its firm, category key and code
func GetChoiceOptionByCode(db *gorm.DB, firmID string, categoryKey string, code string) (models.ChoiceOption, error) {
	var option models.ChoiceOption

	err := db.
		Joins("JOIN choice_categories ON choice_categories.id = choice_options.category_id").
		Where("choice_categories.firm_id = ?", firmID).
		Where("choice_categories.key = ?", categoryKey).
		Where("choice_options.code = ?", code).
		First(&option).Error

	return option, err
}

// ValidateChoiceOption validates that a choice option exists for a firm and category
func ValidateChoiceOption(db *gorm.DB, firmID string, categoryKey string, code string) bool {
	var count int64

	db.Model(&models.ChoiceOption{}).
		Joins("JOIN choice_categories ON choice_categories.id = choice_options.category_id").
		Where("choice_categories.firm_id = ?", firmID).
		Where("choice_categories.key = ?", categoryKey).
		Where("choice_categories.is_active = ?", true).
		Where("choice_options.code = ?", code).
		Where("choice_options.is_active = ?", true).
		Count(&count)

	return count > 0
}

// SeedDefaultChoices seeds default choice categories and options for a firm based on country
func SeedDefaultChoices(db *gorm.DB, firmID string, country string) error {
	// Seed priority category (for all countries)
	if err := seedPriorityChoices(db, firmID, country); err != nil {
		log.Printf("Error seeding priority choices for firm %s: %v", firmID, err)
		return err
	}

	// Seed country-specific choices
	switch country {
	case "Colombia":
		if err := seedColombianDocumentTypes(db, firmID, country); err != nil {
			log.Printf("Error seeding Colombian document types for firm %s: %v", firmID, err)
			return err
		}
	default:
		log.Printf("No country-specific choices to seed for country: %s", country)
	}

	return nil
}

// seedPriorityChoices seeds priority level options (applicable to all countries)
func seedPriorityChoices(db *gorm.DB, firmID string, country string) error {
	// Check if already exists
	var existing models.ChoiceCategory
	if err := db.Where("firm_id = ? AND key = ?", firmID, "priority").First(&existing).Error; err == nil {
		return nil // Already seeded
	}

	// Create priority category
	category := models.ChoiceCategory{
		FirmID:   firmID,
		Country:  country,
		Key:      "priority",
		Name:     "Priority Level",
		Order:    1,
		IsActive: true,
		IsSystem: true,
	}

	if err := db.Create(&category).Error; err != nil {
		return err
	}

	// Create priority options
	priorities := []models.ChoiceOption{
		{CategoryID: category.ID, Code: "low", Label: "Low", SortOrder: 1, IsActive: true, IsSystem: true},
		{CategoryID: category.ID, Code: "medium", Label: "Medium", SortOrder: 2, IsActive: true, IsSystem: true},
		{CategoryID: category.ID, Code: "high", Label: "High", SortOrder: 3, IsActive: true, IsSystem: true},
		{CategoryID: category.ID, Code: "urgent", Label: "Urgent", SortOrder: 4, IsActive: true, IsSystem: true},
	}

	for _, priority := range priorities {
		if err := db.Create(&priority).Error; err != nil {
			return err
		}
	}

	return nil
}

// seedColombianDocumentTypes seeds Colombian document type options
func seedColombianDocumentTypes(db *gorm.DB, firmID string, country string) error {
	// Check if already exists
	var existing models.ChoiceCategory
	if err := db.Where("firm_id = ? AND key = ?", firmID, "document_type").First(&existing).Error; err == nil {
		return nil // Already seeded
	}

	// Create document type category
	category := models.ChoiceCategory{
		FirmID:   firmID,
		Country:  country,
		Key:      "document_type",
		Name:     "Document Type",
		Order:    0,
		IsActive: true,
		IsSystem: true,
	}

	if err := db.Create(&category).Error; err != nil {
		return err
	}

	// Create Colombian document type options
	documentTypes := []models.ChoiceOption{
		{CategoryID: category.ID, Code: "CC", Label: "Cédula de Ciudadanía (CC)", SortOrder: 1, IsActive: true, IsSystem: true},
		{CategoryID: category.ID, Code: "CE", Label: "Cédula de Extranjería (CE)", SortOrder: 2, IsActive: true, IsSystem: true},
		{CategoryID: category.ID, Code: "Pasaporte", Label: "Pasaporte", SortOrder: 3, IsActive: true, IsSystem: true},
		{CategoryID: category.ID, Code: "NIT", Label: "NIT (Company)", SortOrder: 4, IsActive: true, IsSystem: true},
		{CategoryID: category.ID, Code: "TI", Label: "Tarjeta de Identidad (TI)", SortOrder: 5, IsActive: true, IsSystem: true},
	}

	for _, docType := range documentTypes {
		if err := db.Create(&docType).Error; err != nil {
			return err
		}
	}

	return nil
}
