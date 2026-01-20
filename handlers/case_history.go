package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/pages"
	"law_flow_app_go/templates/partials"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

// HistoricalCasesPageHandler renders the historical cases page
func HistoricalCasesPageHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)
	csrfToken := middleware.GetCSRFToken(c)

	component := pages.HistoricalCases(c.Request().Context(), "Historical Cases", csrfToken, currentUser, firm)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// GetHistoricalCaseFormHandler renders the historical case creation modal
func GetHistoricalCaseFormHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)

	// Fetch available clients (users with role 'client' in the same firm)
	var clients []models.User
	clientQuery := middleware.GetFirmScopedQuery(c, db.DB)
	if err := clientQuery.
		Where("role = ?", "client").
		Where("is_active = ?", true).
		Order("name ASC").
		Find(&clients).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch clients")
	}

	// Fetch available lawyers (users with role 'lawyer', 'admin', or 'staff' in the same firm)
	var lawyers []models.User
	lawyerQuery := middleware.GetFirmScopedQuery(c, db.DB)
	if err := lawyerQuery.
		Where("role IN (?, ?, ?)", "lawyer", "admin", "staff").
		Where("is_active = ?", true).
		Order("name ASC").
		Find(&lawyers).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch lawyers")
	}

	// Fetch classification domains
	var domains []models.CaseDomain
	domainQuery := middleware.GetFirmScopedQuery(c, db.DB)
	if err := domainQuery.
		Where("is_active = ?", true).
		Order("name ASC").
		Find(&domains).Error; err != nil {
		// Non-fatal, classification is optional
		domains = []models.CaseDomain{}
	}

	// Fetch document types for new client creation
	documentTypes, _ := services.GetChoiceOptions(db.DB, firm.ID, "document_type")

	// Render the modal
	component := partials.CaseHistoryModal(c.Request().Context(), clients, lawyers, domains, documentTypes, currentUser)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// CreateHistoricalCaseHandler creates a new historical case
func CreateHistoricalCaseHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)
	firmID := firm.ID

	// Get form values
	clientMode := c.FormValue("client_mode") // "existing" or "new"
	originalFilingDateStr := c.FormValue("original_filing_date")
	historicalCaseNumber := strings.TrimSpace(c.FormValue("historical_case_number"))
	title := strings.TrimSpace(c.FormValue("title"))
	description := strings.TrimSpace(c.FormValue("description"))
	assignedToID := c.FormValue("assigned_to_id")
	domainID := c.FormValue("domain_id")
	branchID := c.FormValue("branch_id")
	subtypeIDs := c.Request().Form["subtype_ids[]"]
	migrationNotes := strings.TrimSpace(c.FormValue("migration_notes"))

	// Client fields (for new client)
	clientID := c.FormValue("client_id")
	newClientName := strings.TrimSpace(c.FormValue("new_client_name"))
	newClientEmail := strings.TrimSpace(c.FormValue("new_client_email"))
	newClientPhone := strings.TrimSpace(c.FormValue("new_client_phone"))
	newClientDocTypeID := c.FormValue("new_client_doc_type_id")
	newClientDocNumber := strings.TrimSpace(c.FormValue("new_client_doc_number"))

	// Validate required fields
	if originalFilingDateStr == "" {
		return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Original filing date is required</div>`)
	}

	if description == "" {
		return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Description is required</div>`)
	}

	if assignedToID == "" {
		return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Assigned lawyer is required</div>`)
	}

	// Validate client based on mode
	if clientMode == "new" {
		if newClientName == "" || newClientEmail == "" {
			return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Client name and email are required</div>`)
		}
	} else {
		if clientID == "" {
			return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Client is required</div>`)
		}
	}

	// Parse original filing date
	originalFilingDate, err := services.ParseDate(originalFilingDateStr)
	if err != nil {
		return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Invalid date format (expected YYYY-MM-DD)</div>`)
	}

	// Validate lawyer exists and belongs to firm
	var lawyer models.User
	lawyerQuery := middleware.GetFirmScopedQuery(c, db.DB)
	if err := lawyerQuery.
		Where("role IN (?, ?, ?)", "lawyer", "admin", "staff").
		Where("is_active = ?", true).
		First(&lawyer, "id = ?", assignedToID).Error; err != nil {
		return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Invalid lawyer selected</div>`)
	}

	// Start transaction
	tx := db.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Handle client creation or validation
	var finalClientID string
	if clientMode == "new" {
		// Check if client with same email already exists
		var existingClient models.User
		if err := tx.Where("firm_id = ? AND email = ? AND role = ?", firmID, newClientEmail, "client").First(&existingClient).Error; err == nil {
			tx.Rollback()
			return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">A client with this email already exists</div>`)
		}

		// Generate random password for new client
		randomBytes := make([]byte, 32)
		if _, err := rand.Read(randomBytes); err != nil {
			tx.Rollback()
			return c.HTML(http.StatusInternalServerError, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Failed to generate password</div>`)
		}
		randomPassword := base64.URLEncoding.EncodeToString(randomBytes)
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(randomPassword), bcrypt.DefaultCost)
		if err != nil {
			tx.Rollback()
			return c.HTML(http.StatusInternalServerError, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Failed to hash password</div>`)
		}

		// Create new client
		newClient := models.User{
			Name:     newClientName,
			Email:    newClientEmail,
			Password: string(hashedPassword),
			FirmID:   &firmID,
			Role:     "client",
			IsActive: true,
		}

		if newClientPhone != "" {
			newClient.PhoneNumber = &newClientPhone
		}
		if newClientDocTypeID != "" {
			newClient.DocumentTypeID = &newClientDocTypeID
		}
		if newClientDocNumber != "" {
			newClient.DocumentNumber = &newClientDocNumber
		}

		if err := tx.Create(&newClient).Error; err != nil {
			tx.Rollback()
			return c.HTML(http.StatusInternalServerError, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Failed to create client</div>`)
		}

		finalClientID = newClient.ID
	} else {
		// Validate existing client
		var client models.User
		clientQuery := middleware.GetFirmScopedQuery(c, tx)
		if err := clientQuery.
			Where("role = ?", "client").
			Where("is_active = ?", true).
			First(&client, "id = ?", clientID).Error; err != nil {
			tx.Rollback()
			return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Invalid client selected</div>`)
		}
		finalClientID = clientID
	}

	// Generate case number
	caseNumber, err := services.EnsureUniqueCaseNumber(tx, firmID)
	if err != nil {
		tx.Rollback()
		return c.HTML(http.StatusInternalServerError, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Failed to generate case number</div>`)
	}

	// Create the historical case (ALWAYS with CLOSED status)
	now := time.Now()
	newCase := models.Case{
		FirmID:       firmID,
		ClientID:     finalClientID,
		CaseNumber:   caseNumber,
		CaseType:     "HISTORICAL",
		Description:  description,
		Status:       models.CaseStatusClosed, // Historical cases are always closed
		OpenedAt:     originalFilingDate,      // Use the original filing date as opened date
		ClosedAt:     &now,                    // Set closed date to now
		AssignedToID: &assignedToID,

		// Historical case specific fields
		IsHistorical:       true,
		OriginalFilingDate: &originalFilingDate,
		MigratedAt:         &now,
		MigratedBy:         &currentUser.ID,
	}

	// Optional fields
	if title != "" {
		newCase.Title = &title
	}

	if historicalCaseNumber != "" {
		newCase.HistoricalCaseNumber = &historicalCaseNumber
	}

	if migrationNotes != "" {
		newCase.MigrationNotes = &migrationNotes
	}

	if domainID != "" {
		newCase.DomainID = &domainID
		newCase.ClassifiedAt = &now
		newCase.ClassifiedBy = &currentUser.ID
	}

	if branchID != "" {
		newCase.BranchID = &branchID
	}

	// Create the case
	if err := tx.Create(&newCase).Error; err != nil {
		tx.Rollback()
		return c.HTML(http.StatusInternalServerError, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Failed to create case</div>`)
	}

	// Handle subtypes if provided
	if len(subtypeIDs) > 0 {
		var subtypes []models.CaseSubtype
		if err := tx.Where("id IN ?", subtypeIDs).Find(&subtypes).Error; err == nil && len(subtypes) > 0 {
			if err := tx.Model(&newCase).Association("Subtypes").Replace(subtypes); err != nil {
				tx.Rollback()
				return c.HTML(http.StatusInternalServerError, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Failed to assign subtypes</div>`)
			}
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return c.HTML(http.StatusInternalServerError, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Failed to save case</div>`)
	}

	// Handle document uploads (outside transaction - non-critical)
	form, err := c.MultipartForm()
	if err == nil && form != nil && form.File != nil {
		files := form.File["documents[]"]
		for _, fileHeader := range files {
			if err := saveHistoricalCaseDocument(c, fileHeader, newCase.ID, firmID, currentUser.ID); err != nil {
				c.Logger().Errorf("Failed to save document %s: %v", fileHeader.Filename, err)
				// Continue with other documents, don't fail the whole request
			}
		}
	}

	// Return success with redirect
	c.Response().Header().Set("HX-Trigger", "reload-cases")
	return c.HTML(http.StatusOK, `
		<div class="p-4 bg-green-500/20 text-green-400 rounded-lg mb-4">
			Historical case created successfully!
		</div>
		<script>
			setTimeout(function() {
				document.getElementById('case-history-modal').remove();
				document.body.dispatchEvent(new CustomEvent('reload-cases'));
			}, 1000);
		</script>
	`)
}

// saveHistoricalCaseDocument saves an uploaded document to the case
func saveHistoricalCaseDocument(c echo.Context, fileHeader *multipart.FileHeader, caseID, firmID, uploadedByID string) error {
	// Open the uploaded file
	src, err := fileHeader.Open()
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// Generate unique filename
	ext := filepath.Ext(fileHeader.Filename)
	uniqueFilename := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), strings.ReplaceAll(fileHeader.Filename, ext, ""), ext)

	// Create destination path
	destDir := filepath.Join("uploads", "firms", firmID, "cases", caseID)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	destPath := filepath.Join(destDir, uniqueFilename)

	// Create destination file
	dst, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	// Copy file contents
	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Create document record
	doc := models.CaseDocument{
		FirmID:           firmID,
		CaseID:           &caseID,
		FileName:         uniqueFilename,
		FileOriginalName: fileHeader.Filename,
		FilePath:         destPath,
		FileSize:         fileHeader.Size,
		DocumentType:     "other", // Default type for historical documents
		UploadedByID:     &uploadedByID,
	}

	if err := db.DB.Create(&doc).Error; err != nil {
		// Clean up file if database insert fails
		os.Remove(destPath)
		return fmt.Errorf("failed to save document record: %w", err)
	}

	return nil
}

// GetHistoricalCaseBranchesHandler returns branches for a domain (JSON for Alpine.js)
func GetHistoricalCaseBranchesHandler(c echo.Context) error {
	domainID := c.QueryParam("domain_id")
	if domainID == "" {
		return c.JSON(http.StatusOK, map[string]interface{}{"branches": []interface{}{}})
	}

	var branches []models.CaseBranch
	branchQuery := middleware.GetFirmScopedQuery(c, db.DB)
	if err := branchQuery.
		Where("domain_id = ?", domainID).
		Where("is_active = ?", true).
		Order("name ASC").
		Find(&branches).Error; err != nil {
		return c.JSON(http.StatusOK, map[string]interface{}{"branches": []interface{}{}})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"branches": branches})
}

// GetHistoricalCaseSubtypesHandler returns subtypes for a branch (JSON for Alpine.js)
func GetHistoricalCaseSubtypesHandler(c echo.Context) error {
	branchID := c.QueryParam("branch_id")
	if branchID == "" {
		return c.JSON(http.StatusOK, map[string]interface{}{"subtypes": []interface{}{}})
	}

	var subtypes []models.CaseSubtype
	subtypeQuery := middleware.GetFirmScopedQuery(c, db.DB)
	if err := subtypeQuery.
		Where("branch_id = ?", branchID).
		Where("is_active = ?", true).
		Order("name ASC").
		Find(&subtypes).Error; err != nil {
		return c.JSON(http.StatusOK, map[string]interface{}{"subtypes": []interface{}{}})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"subtypes": subtypes})
}
