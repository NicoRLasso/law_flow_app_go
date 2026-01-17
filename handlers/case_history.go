package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/pages"
	"law_flow_app_go/templates/partials"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
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

	// Render the modal
	component := partials.CaseHistoryModal(c.Request().Context(), clients, lawyers, domains, currentUser)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// CreateHistoricalCaseHandler creates a new historical case
func CreateHistoricalCaseHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)
	firmID := firm.ID

	// Get form values
	originalFilingDateStr := c.FormValue("original_filing_date")
	historicalCaseNumber := strings.TrimSpace(c.FormValue("historical_case_number"))
	title := strings.TrimSpace(c.FormValue("title"))
	description := strings.TrimSpace(c.FormValue("description"))
	clientID := c.FormValue("client_id")
	assignedToID := c.FormValue("assigned_to_id")
	domainID := c.FormValue("domain_id")
	branchID := c.FormValue("branch_id")
	subtypeIDs := c.Request().Form["subtype_ids[]"]
	migrationNotes := strings.TrimSpace(c.FormValue("migration_notes"))

	// Validate required fields
	if originalFilingDateStr == "" {
		return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Original filing date is required</div>`)
	}

	if description == "" {
		return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Description is required</div>`)
	}

	if clientID == "" {
		return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Client is required</div>`)
	}

	if assignedToID == "" {
		return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Assigned lawyer is required</div>`)
	}

	// Parse original filing date
	originalFilingDate, err := time.Parse("2006-01-02", originalFilingDateStr)
	if err != nil {
		return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Invalid date format</div>`)
	}

	// Validate client exists and belongs to firm
	var client models.User
	clientQuery := middleware.GetFirmScopedQuery(c, db.DB)
	if err := clientQuery.
		Where("role = ?", "client").
		Where("is_active = ?", true).
		First(&client, "id = ?", clientID).Error; err != nil {
		return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Invalid client selected</div>`)
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

	// Generate case number
	caseNumber, err := services.EnsureUniqueCaseNumber(db.DB, firmID)
	if err != nil {
		return c.HTML(http.StatusInternalServerError, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Failed to generate case number</div>`)
	}

	// Create the historical case
	now := time.Now()
	newCase := models.Case{
		FirmID:       firmID,
		ClientID:     clientID,
		CaseNumber:   caseNumber,
		CaseType:     "HISTORICAL",
		Description:  description,
		Status:       models.CaseStatusOpen,
		OpenedAt:     originalFilingDate, // Use the original filing date as opened date
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

	// Start transaction
	tx := db.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

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
