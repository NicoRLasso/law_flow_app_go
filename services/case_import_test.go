package services

import (
	"context"
	"fmt"
	"law_flow_app_go/models"
	"law_flow_app_go/services/i18n"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xuri/excelize/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCaseImportTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(
		&models.Firm{},
		&models.User{},
		&models.Case{},
		&models.CaseDomain{},
		&models.CaseBranch{},
		&models.CaseSubtype{},
		&models.ChoiceCategory{},
		&models.ChoiceOption{},
		&models.CaseMilestone{},
		&models.Notification{},
	)

	// Initialize i18n
	i18n.Load()

	return db
}

func TestGenerateExcelTemplate(t *testing.T) {
	db := setupCaseImportTestDB()
	ctx := context.Background()
	firmID := "firm-temp"

	db.Create(&models.Firm{ID: firmID, Name: "Test Firm", Slug: "TEST"})

	buf, err := GenerateExcelTemplate(ctx, db, firmID)
	assert.NoError(t, err)
	assert.NotNil(t, buf)

	// Verify Excel Content
	f, err := excelize.OpenReader(buf)
	assert.NoError(t, err)
	defer f.Close()

	// Should have at least 3 sheets
	assert.GreaterOrEqual(t, f.SheetCount, 3)

	sheets := f.GetSheetList()
	assert.Contains(t, sheets, i18n.T(ctx, "cases.import.sheets.instructions"))
	assert.Contains(t, sheets, i18n.T(ctx, "cases.import.sheets.clients"))
	assert.Contains(t, sheets, i18n.T(ctx, "cases.import.sheets.cases"))
}

func TestBulkCreateFromExcel(t *testing.T) {
	db := setupCaseImportTestDB()
	ctx := context.Background()
	firmID := "firm-bulk"
	userID := "user-admin"

	db.Create(&models.Firm{ID: firmID, Name: "Bulk Firm", Slug: "BULK"})

	// Create required Choice Category for Document Types
	cat := models.ChoiceCategory{FirmID: firmID, Key: "document_type", Name: "Document Type", IsActive: true}
	db.Create(&cat)
	db.Create(&models.ChoiceOption{CategoryID: cat.ID, Code: "CC", Label: "Cédula", IsActive: true})

	// Setup Classifications
	domain := models.CaseDomain{FirmID: firmID, Name: "Civil", Code: "CIVIL", IsActive: true}
	db.Create(&domain)
	branch := models.CaseBranch{FirmID: firmID, DomainID: domain.ID, Name: "Family", Code: "FAMILY", IsActive: true}
	db.Create(&branch)
	subtype := models.CaseSubtype{FirmID: firmID, BranchID: branch.ID, Name: "Succession", Code: "SUCC", IsActive: true}
	db.Create(&subtype)

	// Create a dummy existing client
	existingEmail := "existing@example.com"
	db.Create(&models.User{ID: "client-existing", Email: existingEmail, FirmID: &firmID, Role: "client", IsActive: true})

	// Create Excel File in memory
	f := excelize.NewFile()
	sheet1 := "Instructions"
	sheet2 := "Clients"
	sheet3 := "Cases"
	f.SetSheetName("Sheet1", sheet1)
	f.NewSheet(sheet2)
	f.NewSheet(sheet3)

	// Clients Header
	f.SetCellValue(sheet2, "A1", "Email*")
	f.SetCellValue(sheet2, "B1", "Name")

	// Row 2: New Client
	f.SetCellValue(sheet2, "A2", "new@example.com")
	f.SetCellValue(sheet2, "B2", "New User")
	// Row 3: Existing Client
	f.SetCellValue(sheet2, "A3", existingEmail)
	f.SetCellValue(sheet2, "B3", "Existing User")

	// Cases Header
	f.SetCellValue(sheet3, "A1", "Email*")
	f.SetCellValue(sheet3, "B1", "LegacyNumber")
	f.SetCellValue(sheet3, "C1", "FilingNumber")
	f.SetCellValue(sheet3, "D1", "Title*")
	f.SetCellValue(sheet3, "E1", "Description*")
	f.SetCellValue(sheet3, "F1", "Domain")
	f.SetCellValue(sheet3, "G1", "Branch")
	f.SetCellValue(sheet3, "H1", "Subtype")
	f.SetCellValue(sheet3, "I1", "Status*")

	// Row 2: Case for New Client
	f.SetCellValue(sheet3, "A2", "new@example.com")
	f.SetCellValue(sheet3, "B2", "LEG-001")
	f.SetCellValue(sheet3, "C2", "FIL-001")
	f.SetCellValue(sheet3, "D2", "Title 1")
	f.SetCellValue(sheet3, "E2", "Desc 1")
	f.SetCellValue(sheet3, "F2", "Civil")
	f.SetCellValue(sheet3, "G2", "Family")
	f.SetCellValue(sheet3, "H2", "Succession")
	f.SetCellValue(sheet3, "I2", "OPEN")

	// Row 3: Case for Existing Client
	f.SetCellValue(sheet3, "A3", existingEmail)
	f.SetCellValue(sheet3, "B3", "LEG-002")
	f.SetCellValue(sheet3, "C3", "FIL-002")
	f.SetCellValue(sheet3, "D3", "Title 2")
	f.SetCellValue(sheet3, "E3", "Desc 2")
	f.SetCellValue(sheet3, "I3", "OPEN")

	buf, _ := f.WriteToBuffer()

	// Execute Import
	result, err := BulkCreateFromExcel(ctx, db, firmID, userID, buf, -1)

	// Assert Result
	assert.NoError(t, err)
	assert.Equal(t, 2, result.SuccessCount)
	assert.Equal(t, 0, result.FailedCount)
	assert.Len(t, result.Errors, 0)

	// Verify Database State
	var newUser models.User
	err = db.Where("email = ?", "new@example.com").First(&newUser).Error
	assert.NoError(t, err)
	assert.Equal(t, "New User", newUser.Name)

	var cases []models.Case
	db.Where("firm_id = ?", firmID).Preload("Subtypes").Find(&cases)
	assert.Len(t, cases, 2)

	// Verify subtype relationship was saved for first case
	foundSubtype := false
	for _, c := range cases {
		if *c.Title == "Title 1" && len(c.Subtypes) > 0 {
			assert.Equal(t, subtype.ID, c.Subtypes[0].ID)
			foundSubtype = true
		}
	}
	assert.True(t, foundSubtype, "Subtype should have been associated with case 1")
}

func TestBulkCreateFromExcel_Validation(t *testing.T) {
	db := setupCaseImportTestDB()
	ctx := context.Background()
	firmID := "firm-err"

	db.Create(&models.Firm{ID: firmID, Name: "Error Firm"})

	// Empty file / Missing sheets
	f := excelize.NewFile()
	buf, _ := f.WriteToBuffer()
	result, err := BulkCreateFromExcel(ctx, db, firmID, "user", buf, -1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing sheets")
	assert.Nil(t, result)
}

func TestBulkCreateFromExcel_LimitEnforcement(t *testing.T) {
	db := setupCaseImportTestDB()
	ctx := context.Background()
	firmID := "firm-limit"

	db.Create(&models.Firm{ID: firmID, Name: "Limit Firm"})

	// Create Excel File in memory with 5 cases
	f := excelize.NewFile()
	sheetCases := "Cases"
	sheetClients := "Clients"
	f.NewSheet(sheetClients)
	f.NewSheet(sheetCases)

	// Clients Header & Data
	f.SetCellValue(sheetClients, "A1", "Email*")
	f.SetCellValue(sheetClients, "A2", "test@example.com")

	// Headers
	f.SetCellValue(sheetCases, "A1", "Email*")
	f.SetCellValue(sheetCases, "B1", "LegacyNumber")
	f.SetCellValue(sheetCases, "C1", "FilingNumber")
	f.SetCellValue(sheetCases, "D1", "Title*")
	f.SetCellValue(sheetCases, "E1", "Description*")
	f.SetCellValue(sheetCases, "I1", "Status*")

	// 5 Rows
	for i := 2; i <= 6; i++ {
		cellEmail, _ := excelize.CoordinatesToCellName(1, i)
		cellTitle, _ := excelize.CoordinatesToCellName(4, i)
		cellDesc, _ := excelize.CoordinatesToCellName(5, i)
		cellStatus, _ := excelize.CoordinatesToCellName(9, i) // I is 9th letter

		f.SetCellValue(sheetCases, cellEmail, "test@example.com")
		f.SetCellValue(sheetCases, cellTitle, fmt.Sprintf("Case %d", i))
		f.SetCellValue(sheetCases, cellDesc, "Desc")
		f.SetCellValue(sheetCases, cellStatus, "OPEN")
	}

	// Create Client for foreign key
	db.Create(&models.User{ID: "client-test", Email: "test@example.com", FirmID: &firmID, Role: "client", IsActive: true})

	buf, _ := f.WriteToBuffer()

	// Execute Import with Limit 3
	result, err := BulkCreateFromExcel(ctx, db, firmID, "user", buf, 3)

	// Assert Result
	assert.NoError(t, err)
	assert.Equal(t, 3, result.SuccessCount)
	assert.Equal(t, 2, result.SkippedOverLimitCount)

	// Verify Database State
	var count int64
	db.Model(&models.Case{}).Where("firm_id = ?", firmID).Count(&count)
	assert.Equal(t, int64(3), count)
}
