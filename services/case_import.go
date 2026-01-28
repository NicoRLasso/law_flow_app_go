package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"law_flow_app_go/config"
	"law_flow_app_go/models"
	"law_flow_app_go/services/i18n"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// ImportResult contains the summary of the import process
type ImportResult struct {
	TotalProcessed        int
	SuccessCount          int
	FailedCount           int
	SkippedOverLimitCount int
	Errors                []string
}

// GenerateExcelTemplate generates the Excel template for case import
func GenerateExcelTemplate(ctx context.Context, dbConn *gorm.DB, firmID string) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	defer f.Close()

	// Get Firm to check Country/Config
	var firm models.Firm
	if err := dbConn.First(&firm, "id = ?", firmID).Error; err != nil {
		return nil, fmt.Errorf("failed to get firm: %w", err)
	}

	// Rename Sheet1 to Instructions
	sheetInstructions := i18n.T(ctx, "cases.import.sheets.instructions")
	f.SetSheetName("Sheet1", sheetInstructions)

	// --- Instructions Sheet ---
	f.SetCellValue(sheetInstructions, "A1", i18n.T(ctx, "cases.import.instructions.title"))
	f.SetCellValue(sheetInstructions, "A3", i18n.T(ctx, "cases.import.instructions.considerations"))
	f.SetCellValue(sheetInstructions, "A4", "- "+i18n.T(ctx, "cases.import.instructions.cons_1"))
	f.SetCellValue(sheetInstructions, "A5", "- "+i18n.T(ctx, "cases.import.instructions.cons_2"))
	f.SetCellValue(sheetInstructions, "A6", "- "+i18n.T(ctx, "cases.import.instructions.cons_3"))
	f.SetCellValue(sheetInstructions, "A7", "- "+i18n.T(ctx, "cases.import.instructions.cons_4"))
	f.SetCellValue(sheetInstructions, "A8", "- "+i18n.T(ctx, "cases.import.instructions.cons_5"))

	// Dynamic Document Type Instruction
	docTypeCodes := []string{}
	var docTypes []models.ChoiceOption
	// Fetch actual document types for this firm
	if err := dbConn.Joins("Category").
		Where("Category.firm_id = ? AND Category.key = ? AND choice_options.is_active = ?", firmID, "document_type", true).
		Find(&docTypes).Error; err == nil {
		for _, dt := range docTypes {
			docTypeCodes = append(docTypeCodes, dt.Code)
		}
	}

	docTypeInstruction := i18n.T(ctx, "cases.import.instructions.cons_6")
	if len(docTypeCodes) > 0 {
		docTypeInstruction += fmt.Sprintf(" (%s)", strings.Join(docTypeCodes, ", "))
	}
	f.SetCellValue(sheetInstructions, "A9", "- "+docTypeInstruction)

	f.SetCellValue(sheetInstructions, "A10", "- "+i18n.T(ctx, "cases.import.instructions.cons_7"))

	// List Valid Classifications
	f.SetCellValue(sheetInstructions, "A12", i18n.T(ctx, "cases.import.instructions.valid_classifications"))
	titleStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 12}})
	f.SetCellStyle(sheetInstructions, "A12", "A12", titleStyle)

	// Fetch Classifications (Scoped to Firm)
	var domains []models.CaseDomain
	if err := dbConn.Where("firm_id = ? AND is_active = ?", firmID, true).
		Preload("Branches", "is_active = ?", true).
		Preload("Branches.Subtypes", "is_active = ?", true).
		Find(&domains).Error; err == nil {

		row := 13
		f.SetCellValue(sheetInstructions, fmt.Sprintf("A%d", row), "Domain > Branch > Subtype")
		f.SetCellStyle(sheetInstructions, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), titleStyle)
		row++

		for _, d := range domains {
			if len(d.Branches) == 0 {
				f.SetCellValue(sheetInstructions, fmt.Sprintf("A%d", row), d.Name)
				row++
				continue
			}
			for _, b := range d.Branches {
				if len(b.Subtypes) == 0 {
					f.SetCellValue(sheetInstructions, fmt.Sprintf("A%d", row), fmt.Sprintf("%s > %s", d.Name, b.Name))
					row++
					continue
				}
				for _, s := range b.Subtypes {
					f.SetCellValue(sheetInstructions, fmt.Sprintf("A%d", row), fmt.Sprintf("%s > %s > %s", d.Name, b.Name, s.Name))
					row++
				}
			}
		}
	}

	// Style Instructions Title
	mainTitleStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 14}})
	f.SetCellStyle(sheetInstructions, "A1", "A1", mainTitleStyle)
	f.SetColWidth(sheetInstructions, "A", "A", 80)

	// --- Clients Sheet ---
	sheetClients := i18n.T(ctx, "cases.import.sheets.clients")
	f.NewSheet(sheetClients)
	clientHeaders := []string{
		i18n.T(ctx, "cases.import.headers.email") + "*",     // A
		i18n.T(ctx, "cases.import.headers.name"),            // B
		i18n.T(ctx, "cases.import.headers.phone"),           // C
		i18n.T(ctx, "cases.import.headers.document_type"),   // D
		i18n.T(ctx, "cases.import.headers.document_number"), // E
	}
	for i, header := range clientHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetClients, cell, header)
	}
	f.SetColWidth(sheetClients, "A", "E", 20)

	// Simplified Example Client
	exampleDocType := "CC"
	if len(docTypeCodes) > 0 {
		exampleDocType = docTypeCodes[0]
	}

	f.SetCellValue(sheetClients, "A2", "client@example.com")
	f.SetCellValue(sheetClients, "B2", "John Doe")
	f.SetCellValue(sheetClients, "C2", "+123456789")
	f.SetCellValue(sheetClients, "D2", exampleDocType)
	f.SetCellValue(sheetClients, "E2", "1234567890")

	// --- Cases Sheet ---
	sheetCases := i18n.T(ctx, "cases.import.sheets.cases")
	f.NewSheet(sheetCases)
	caseHeaders := []string{
		i18n.T(ctx, "cases.import.headers.email") + "*",       // A
		i18n.T(ctx, "cases.import.headers.legacy_number"),     // B
		i18n.T(ctx, "cases.import.headers.filing_number"),     // C [NEW]
		i18n.T(ctx, "cases.import.headers.case_title") + "*",  // D
		i18n.T(ctx, "cases.import.headers.description") + "*", // E
		i18n.T(ctx, "cases.import.headers.domain"),            // F (Optional)
		i18n.T(ctx, "cases.import.headers.branch"),            // G (Optional)
		i18n.T(ctx, "cases.import.headers.subtype"),           // H (Optional)
		i18n.T(ctx, "cases.import.headers.status") + "*",      // I
		i18n.T(ctx, "cases.import.headers.opened_date"),       // J
		i18n.T(ctx, "cases.import.headers.closed_date"),       // K
	}
	for i, header := range caseHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetCases, cell, header)
	}
	f.SetColWidth(sheetCases, "A", "K", 20)

	// Fetch a real Domain/Branch/Subtype for the example
	var exampleDomainName = "Civil"
	var exampleBranchName = "Family"
	var exampleSubtypeName = ""

	if len(domains) > 0 {
		d := domains[0]
		exampleDomainName = d.Name
		if len(d.Branches) > 0 {
			b := d.Branches[0]
			exampleBranchName = b.Name
			if len(b.Subtypes) > 0 {
				exampleSubtypeName = b.Subtypes[0].Name
			}
		}
	}

	// Example Case
	f.SetCellValue(sheetCases, "A2", "client@example.com")
	f.SetCellValue(sheetCases, "B2", "LEGACY-001")
	f.SetCellValue(sheetCases, "C2", "123-456-789")
	f.SetCellValue(sheetCases, "D2", "Example Case Title")
	f.SetCellValue(sheetCases, "E2", "Description of the case...")
	f.SetCellValue(sheetCases, "F2", exampleDomainName)
	f.SetCellValue(sheetCases, "G2", exampleBranchName)
	f.SetCellValue(sheetCases, "H2", exampleSubtypeName)
	f.SetCellValue(sheetCases, "I2", "OPEN")
	f.SetCellValue(sheetCases, "J2", time.Now().Format("2006-01-02"))

	// Header Style
	headerStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	f.SetCellStyle(sheetClients, "A1", "E1", headerStyle)
	f.SetCellStyle(sheetCases, "A1", "K1", headerStyle)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to write excel buffer: %w", err)
	}

	return buf, nil
}

// AnalyzeExcelFile reads the file and returns basic stats like total rows to process
func AnalyzeExcelFile(file io.Reader) (int, error) {
	// We need to read into a byte slicer because OpenReader needs a readerAt or we use OpenFile?
	// Excelize OpenReader takes io.Reader, so we are good.
	f, err := excelize.OpenReader(file)
	if err != nil {
		return 0, fmt.Errorf("failed to open excel file: %w", err)
	}
	defer f.Close()

	if f.SheetCount < 3 {
		return 0, fmt.Errorf("invalid excel format: missing sheets")
	}

	sheets := f.GetSheetList()
	caseSheetName := sheets[2]

	rows, err := f.GetRows(caseSheetName)
	if err != nil {
		return 0, fmt.Errorf("failed to read cases sheet: %w", err)
	}

	totalRows := 0
	for i, row := range rows {
		if i == 0 {
			continue
		} // Header
		// Basic check for empty row
		if len(row) > 0 {
			// At least email required
			if strings.TrimSpace(row[0]) != "" {
				totalRows++
			}
		}
	}

	return totalRows, nil
}

// BulkCreateFromExcel parses the Excel file and creates records
func BulkCreateFromExcel(ctx context.Context, dbConn *gorm.DB, firmID string, userID string, file io.Reader, limit int) (*ImportResult, error) {
	f, err := excelize.OpenReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to open excel file: %w", err)
	}
	defer f.Close()

	result := &ImportResult{
		Errors: []string{},
	}

	// Check Sheet count
	if f.SheetCount < 3 {
		return nil, fmt.Errorf("invalid excel format: missing sheets")
	}

	sheets := f.GetSheetList()
	clientSheetName := sheets[1]
	caseSheetName := sheets[2]

	// --- Phase 1: Clients ---
	clientRows, err := f.GetRows(clientSheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read clients sheet: %w", err)
	}

	clientEmailToID := make(map[string]string)

	// Pre-fetch Map of Document Types (Code -> ID)
	docTypeMap := make(map[string]string)
	var docTypes []models.ChoiceOption
	// Join with ChoiceCategory to find 'document_type' code or similar.
	// Assuming category Key is 'document_type' AND firm_id matches
	if err := dbConn.Joins("Category").Where("Category.firm_id = ? AND Category.key = ? AND choice_options.is_active = ?", firmID, "document_type", true).Find(&docTypes).Error; err == nil {
		for _, dt := range docTypes {
			docTypeMap[strings.ToUpper(dt.Code)] = dt.ID
			docTypeMap[strings.ToUpper(dt.Label)] = dt.ID // Allow matching by label too if unique
		}
	}

	tx := dbConn.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Track new users to send emails after commit
	var newUsersCreated []models.User

	// Skip Header
	for i, row := range clientRows {
		if i == 0 {
			continue
		} // Header
		if len(row) < 1 {
			continue
		}

		email := strings.TrimSpace(row[0])
		if email == "" {
			continue
		}

		name := ""
		if len(row) > 1 {
			name = row[1]
		}

		phone := ""
		if len(row) > 2 {
			phone = row[2]
		}

		docTypeCode := ""
		if len(row) > 3 {
			docTypeCode = strings.ToUpper(strings.TrimSpace(row[3]))
		}

		docNumber := ""
		if len(row) > 4 {
			docNumber = strings.TrimSpace(row[4])
		}

		// Resolve Doc Type ID
		var docTypeID *string
		if docTypeCode != "" {
			if id, ok := docTypeMap[docTypeCode]; ok {
				docTypeID = &id
			}
		}

		// Check if user exists
		var existingUser models.User
		err := tx.Where("email = ? AND firm_id = ?", email, firmID).First(&existingUser).Error
		if err == nil {
			clientEmailToID[strings.ToLower(email)] = existingUser.ID
			// Update existing user with doc info if missing?
			// For import, maybe update if provided?
			updates := make(map[string]interface{})
			if existingUser.DocumentTypeID == nil && docTypeID != nil {
				updates["document_type_id"] = docTypeID
			}
			if existingUser.DocumentNumber == nil && docNumber != "" {
				updates["document_number"] = docNumber
			}
			if len(updates) > 0 {
				if err := tx.Model(&existingUser).Updates(updates).Error; err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Row %d (Client): Warning: Failed to update existing user info: %v", i+1, err))
				}
			}

		} else if err == gorm.ErrRecordNotFound {
			// Create new client user
			var docNumPtr *string
			if docNumber != "" {
				docNumPtr = &docNumber
			}

			newUser := models.User{
				FirmID:         &firmID,
				Email:          email,
				Name:           name,
				PhoneNumber:    &phone,
				Role:           "client",
				IsActive:       true,
				Password:       uuid.New().String(),
				DocumentTypeID: docTypeID,
				DocumentNumber: docNumPtr,
			}
			if err := tx.Create(&newUser).Error; err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Row %d (Client): Failed to create user %s: %v", i+1, email, err))
				continue
			}
			clientEmailToID[strings.ToLower(email)] = newUser.ID
			newUsersCreated = append(newUsersCreated, newUser)
		} else {
			result.Errors = append(result.Errors, fmt.Sprintf("Row %d (Client): Database error for %s: %v", i+1, email, err))
			tx.Rollback()
			return result, err
		}
	}

	// --- Phase 2: Cases ---
	caseRows, err := f.GetRows(caseSheetName)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to read cases sheet: %w", err)
	}

	for i, row := range caseRows {
		if i == 0 {
			continue
		} // Header
		if len(row) < 9 { // Increased minimum columns due to Subtype & Filing Number insertion
			continue
		}

		// Check Limits
		if limit != -1 && result.SuccessCount >= limit {
			result.SkippedOverLimitCount++
			continue
		}

		result.TotalProcessed++

		// Columns:
		// 0: Email*, 1: LegacyNumber, 2: FilingNumber, 3: Title*, 4: Description*, 5: Domain, 6: Branch, 7: Subtype, 8: Status*, 9: OpenedDate, 10: ClosedDate

		email := strings.TrimSpace(row[0])
		if email == "" {
			continue
		}

		clientID, ok := clientEmailToID[strings.ToLower(email)]
		if !ok {
			result.FailedCount++
			result.Errors = append(result.Errors, fmt.Sprintf("Row %d (Case): Client email %s not found in Clients sheet or System", i+1, email))
			continue
		}

		legacyNumber := strings.TrimSpace(row[1])
		filingNumber := strings.TrimSpace(row[2])
		title := row[3]
		description := row[4]
		domainName := strings.TrimSpace(row[5])
		branchName := strings.TrimSpace(row[6])
		subtypeName := strings.TrimSpace(row[7])
		statusRaw := strings.ToUpper(strings.TrimSpace(row[8]))

		// Map Status
		status := models.CaseStatusOpen
		if statusRaw == "CLOSED" {
			status = models.CaseStatusClosed
		}
		if statusRaw == "ON_HOLD" {
			status = models.CaseStatusOnHold
		}
		// Verify valid status string just in case
		if !models.IsValidCaseStatus(status) {
			status = models.CaseStatusOpen
		}

		// Dates
		openedAt := time.Now()
		if len(row) > 9 && row[9] != "" {
			if t, err := time.Parse("2006-01-02", row[9]); err == nil {
				openedAt = t
			}
		}

		var closedAt *time.Time
		if status == models.CaseStatusClosed && len(row) > 10 && row[10] != "" {
			if t, err := time.Parse("2006-01-02", row[10]); err == nil {
				closedAt = &t
			}
		}

		// Resolve Domain/Branch/Subtype (Optional)
		var domainID *string
		var branchID *string
		var subtypes []models.CaseSubtype

		if domainName != "" {
			var domain models.CaseDomain
			if err := tx.Where("firm_id = ? AND name = ?", firmID, domainName).First(&domain).Error; err != nil {
				result.FailedCount++
				result.Errors = append(result.Errors, fmt.Sprintf("Row %d (Case): Domain '%s' not found", i+1, domainName))
				continue
			}
			domainID = &domain.ID

			if branchName != "" {
				var branch models.CaseBranch
				if err := tx.Where("domain_id = ? AND name = ?", domain.ID, branchName).First(&branch).Error; err != nil {
					result.FailedCount++
					result.Errors = append(result.Errors, fmt.Sprintf("Row %d (Case): Branch '%s' not found in Domain '%s'", i+1, branchName, domainName))
					continue
				}
				branchID = &branch.ID

				if subtypeName != "" {
					var subtype models.CaseSubtype
					if err := tx.Where("branch_id = ? AND name = ?", branch.ID, subtypeName).First(&subtype).Error; err != nil {
						// Optional: could warn and proceed, or fail. User asked for optional *classification*, implying if data is present it should be valid?
						// "classification could be optional" usually means fields can be empty. If they are provided but wrong, it's an error.
						result.FailedCount++
						result.Errors = append(result.Errors, fmt.Sprintf("Row %d (Case): Subtype '%s' not found in Branch '%s'", i+1, subtypeName, branchName))
						continue
					}
					subtypes = append(subtypes, subtype)
				}
			}
		}

		// Generate System Case Number
		sysCaseNumber, err := EnsureUniqueCaseNumber(tx, firmID)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to generate case number: %w", err)
		}

		// Create Case
		var filingNumPtr *string
		if filingNumber != "" {
			filingNumPtr = &filingNumber
		}

		newCase := models.Case{
			ID:                   uuid.New().String(),
			FirmID:               firmID,
			ClientID:             clientID,
			CaseNumber:           sysCaseNumber,
			FilingNumber:         filingNumPtr,
			Title:                &title,
			Description:          description,
			CaseType:             "Imported",
			Status:               status,
			OpenedAt:             openedAt,
			ClosedAt:             closedAt,
			DomainID:             domainID,
			BranchID:             branchID,
			IsHistorical:         (status == models.CaseStatusClosed),
			HistoricalCaseNumber: &legacyNumber,
			StatusChangedBy:      &userID,
			StatusChangedAt:      &openedAt, // Approximation
			Subtypes:             subtypes,
		}

		if err := tx.Create(&newCase).Error; err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, fmt.Sprintf("Row %d (Case): Failed to save case: %v", i+1, err))
			continue
		}

		result.SuccessCount++
	}

	if result.FailedCount > 0 && result.SuccessCount == 0 {
		tx.Rollback()
		return result, fmt.Errorf("all rows failed")
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// Send Emails to New Users (Async)
	if len(newUsersCreated) > 0 {
		go func() {
			cfg := config.Load()
			for _, user := range newUsersCreated {
				if user.Email != "" {
					name := user.Name
					if name == "" {
						name = user.Email
					}
					lang := "es" // Default to Spanish or infer from somewhere
					if user.Language != "" {
						lang = user.Language
					}
					email := BuildWelcomeEmail(user.Email, name, lang)
					SendEmailAsync(cfg, email)
				}
			}
		}()
	}

	return result, nil
}
