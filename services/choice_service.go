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

	// Seed service types (for all countries, with country-specific options)
	if err := seedServiceTypes(db, firmID, country); err != nil {
		log.Printf("Error seeding service types for firm %s: %v", firmID, err)
		return err
	}

	// Seed expense categories (for all countries, with country-specific options)
	if err := seedExpenseCategories(db, firmID, country); err != nil {
		log.Printf("Error seeding expense categories for firm %s: %v", firmID, err)
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

	// Seed currency choices (for all countries)
	if err := SeedCurrencyChoices(db, firmID, country); err != nil {
		log.Printf("Error seeding currency choices for firm %s: %v", firmID, err)
		return err
	}

	return nil
}

// SeedCurrencyChoices seeds currency options (exported for use by handlers)
func SeedCurrencyChoices(db *gorm.DB, firmID string, country string) error {
	// Check if already exists
	var existing models.ChoiceCategory
	if err := db.Where("firm_id = ? AND key = ?", firmID, models.ChoiceCategoryKeyCurrency).First(&existing).Error; err == nil {
		return nil // Already seeded
	}

	// Create currency category
	category := models.ChoiceCategory{
		FirmID:   firmID,
		Country:  country,
		Key:      models.ChoiceCategoryKeyCurrency,
		Name:     "Currency",
		Order:    4,
		IsActive: true,
		IsSystem: true,
	}

	if err := db.Create(&category).Error; err != nil {
		return err
	}

	// Create currency options (USD and COP)
	currencies := []models.ChoiceOption{
		{CategoryID: category.ID, Code: "USD", Label: "USD ($)", SortOrder: 1, IsActive: true, IsSystem: true},
		{CategoryID: category.ID, Code: "COP", Label: "COP ($)", SortOrder: 2, IsActive: true, IsSystem: true},
	}

	for _, currency := range currencies {
		if err := db.Create(&currency).Error; err != nil {
			return err
		}
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

// seedServiceTypes seeds service type options for legal services
func seedServiceTypes(db *gorm.DB, firmID string, country string) error {
	// Check if already exists
	var existing models.ChoiceCategory
	if err := db.Where("firm_id = ? AND key = ?", firmID, models.ChoiceCategoryKeyServiceType).First(&existing).Error; err == nil {
		return nil // Already seeded
	}

	// Create service type category
	category := models.ChoiceCategory{
		FirmID:   firmID,
		Country:  country,
		Key:      models.ChoiceCategoryKeyServiceType,
		Name:     "Service Type",
		Order:    2,
		IsActive: true,
		IsSystem: true,
	}

	if err := db.Create(&category).Error; err != nil {
		return err
	}

	// Service types vary by country
	var serviceTypes []models.ChoiceOption

	switch country {
	case "Colombia":
		serviceTypes = []models.ChoiceOption{
			{CategoryID: category.ID, Code: "DOCUMENT_CREATION", Label: "Creación de Documentos", SortOrder: 1, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "TAX_OPINION", Label: "Concepto Tributario", SortOrder: 2, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "VISA_PROCESSING", Label: "Trámite de Visa", SortOrder: 3, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "HOURLY_ADVISORY", Label: "Asesoría por Horas", SortOrder: 4, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "COMPANY_FORMATION", Label: "Creación de Empresa", SortOrder: 5, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "CONTRACT_REVIEW", Label: "Revisión de Contratos", SortOrder: 6, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "NOTARIAL_PROCESS", Label: "Trámite Notarial", SortOrder: 7, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "LEGAL_CONCEPT", Label: "Concepto Jurídico", SortOrder: 8, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "TRADEMARK_REGISTRATION", Label: "Registro de Marca", SortOrder: 9, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "REAL_ESTATE", Label: "Estudio de Títulos / Inmobiliario", SortOrder: 10, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "OTHER", Label: "Otro", SortOrder: 99, IsActive: true, IsSystem: true},
		}
	default:
		// Default English options
		serviceTypes = []models.ChoiceOption{
			{CategoryID: category.ID, Code: "DOCUMENT_CREATION", Label: "Document Creation", SortOrder: 1, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "TAX_OPINION", Label: "Tax Opinion", SortOrder: 2, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "VISA_PROCESSING", Label: "Visa Processing", SortOrder: 3, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "HOURLY_ADVISORY", Label: "Hourly Advisory", SortOrder: 4, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "COMPANY_FORMATION", Label: "Company Formation", SortOrder: 5, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "CONTRACT_REVIEW", Label: "Contract Review", SortOrder: 6, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "NOTARIAL_PROCESS", Label: "Notarial Process", SortOrder: 7, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "LEGAL_CONCEPT", Label: "Legal Opinion", SortOrder: 8, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "TRADEMARK_REGISTRATION", Label: "Trademark Registration", SortOrder: 9, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "REAL_ESTATE", Label: "Real Estate / Title Review", SortOrder: 10, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "OTHER", Label: "Other", SortOrder: 99, IsActive: true, IsSystem: true},
		}
	}

	for _, serviceType := range serviceTypes {
		if err := db.Create(&serviceType).Error; err != nil {
			return err
		}
	}

	return nil
}

// seedExpenseCategories seeds expense category options for service expenses
func seedExpenseCategories(db *gorm.DB, firmID string, country string) error {
	// Check if already exists
	var existing models.ChoiceCategory
	if err := db.Where("firm_id = ? AND key = ?", firmID, models.ChoiceCategoryKeyExpenseCategory).First(&existing).Error; err == nil {
		return nil // Already seeded
	}

	// Create expense category
	category := models.ChoiceCategory{
		FirmID:   firmID,
		Country:  country,
		Key:      models.ChoiceCategoryKeyExpenseCategory,
		Name:     "Expense Category",
		Order:    3,
		IsActive: true,
		IsSystem: true,
	}

	if err := db.Create(&category).Error; err != nil {
		return err
	}

	// Expense categories vary by country
	var expenseCategories []models.ChoiceOption

	switch country {
	case "Colombia":
		expenseCategories = []models.ChoiceOption{
			{CategoryID: category.ID, Code: "NOTARY", Label: "Gastos Notariales", SortOrder: 1, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "TRANSLATION", Label: "Traducción Oficial", SortOrder: 2, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "APOSTILLE", Label: "Apostilla", SortOrder: 3, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "FILING", Label: "Radicación de Documentos", SortOrder: 4, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "GOVERNMENT_FEE", Label: "Tasas Gubernamentales", SortOrder: 5, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "CHAMBER_COMMERCE", Label: "Cámara de Comercio", SortOrder: 6, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "COURIER", Label: "Mensajería / Envíos", SortOrder: 7, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "TRAVEL", Label: "Gastos de Viaje", SortOrder: 8, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "COPIES", Label: "Copias y Autenticaciones", SortOrder: 9, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "OTHER", Label: "Otro", SortOrder: 99, IsActive: true, IsSystem: true},
		}
	default:
		// Default English options
		expenseCategories = []models.ChoiceOption{
			{CategoryID: category.ID, Code: "NOTARY", Label: "Notary Fees", SortOrder: 1, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "TRANSLATION", Label: "Translation", SortOrder: 2, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "APOSTILLE", Label: "Apostille", SortOrder: 3, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "FILING", Label: "Filing Fees", SortOrder: 4, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "GOVERNMENT_FEE", Label: "Government Fees", SortOrder: 5, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "REGISTRY", Label: "Registry Fees", SortOrder: 6, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "COURIER", Label: "Courier / Shipping", SortOrder: 7, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "TRAVEL", Label: "Travel Expenses", SortOrder: 8, IsActive: true, IsSystem: true},
			{CategoryID: category.ID, Code: "OTHER", Label: "Other", SortOrder: 99, IsActive: true, IsSystem: true},
		}
	}

	for _, expenseCategory := range expenseCategories {
		if err := db.Create(&expenseCategory).Error; err != nil {
			return err
		}
	}

	return nil
}
