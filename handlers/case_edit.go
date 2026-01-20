package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/partials"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// GetCaseEditFormHandler renders the edit form for a case
func GetCaseEditFormHandler(c echo.Context) error {
	caseID := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)

	// Build firm-scoped query
	query := middleware.GetFirmScopedQuery(c, db.DB)

	// Apply role-based filter
	if currentUser.Role == "lawyer" {
		// Lawyers only see cases assigned to them
		query = query.Where("assigned_to_id = ?", currentUser.ID)
	}

	// Fetch case with all relationships
	var caseRecord models.Case
	if err := query.
		Preload("Client").
		Preload("AssignedTo").
		Preload("Domain").
		Preload("Branch").
		Preload("Subtypes").
		First(&caseRecord, "id = ?", caseID).Error; err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusNotFound, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Case not found</div>`)
		}
		return echo.NewHTTPError(http.StatusNotFound, "Case not found")
	}

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

	// Fetch available lawyers (users with role 'lawyer' or 'admin' in the same firm)
	var lawyers []models.User
	lawyerQuery := middleware.GetFirmScopedQuery(c, db.DB)
	if err := lawyerQuery.
		Where("role IN (?, ?)", "lawyer", "admin").
		Where("is_active = ?", true).
		Order("name ASC").
		Find(&lawyers).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch lawyers")
	}

	// Fetch classification data
	var domains []models.CaseDomain
	var branches []models.CaseBranch
	var subtypes []models.CaseSubtype

	if currentUser.FirmID != nil {
		domains, _ = services.GetCaseDomains(db.DB, *currentUser.FirmID)
		if caseRecord.DomainID != nil {
			branches, _ = services.GetCaseBranches(db.DB, *currentUser.FirmID, *caseRecord.DomainID)
		}
		if caseRecord.BranchID != nil {
			subtypes, _ = services.GetCaseSubtypes(db.DB, *currentUser.FirmID, *caseRecord.BranchID)
		}
	}

	// Render the edit modal
	component := partials.CaseEditModal(c.Request().Context(), caseRecord, clients, lawyers, currentUser, domains, branches, subtypes, caseRecord.IsHistorical)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// UpdateCaseHandler handles case updates
func UpdateCaseHandler(c echo.Context) error {
	caseID := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)

	// Build firm-scoped query
	query := middleware.GetFirmScopedQuery(c, db.DB)

	// Apply role-based filter
	if currentUser.Role == "lawyer" {
		// Lawyers only edit cases assigned to them
		query = query.Where("assigned_to_id = ?", currentUser.ID)
	}

	// Fetch existing case
	var caseRecord models.Case
	if err := query.First(&caseRecord, "id = ?", caseID).Error; err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusNotFound, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Case not found</div>`)
		}
		return echo.NewHTTPError(http.StatusNotFound, "Case not found")
	}

	// Capture old state for audit logging
	oldCase := caseRecord

	// Get form values
	status := c.FormValue("status")
	description := c.FormValue("description")
	clientID := c.FormValue("client_id")
	assignedToID := c.FormValue("assigned_to_id")
	filingNumber := c.FormValue("filing_number")
	domainID := c.FormValue("domain_id")
	branchID := c.FormValue("branch_id")
	subtypeIDs := c.Request().Form["subtype_ids[]"]

	// Validate required fields
	if status == "" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Status is required</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "Status is required")
	}

	if description == "" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Description is required</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "Description is required")
	}

	// Validate status
	if !models.IsValidCaseStatus(status) {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Invalid status</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid status")
	}

	// Historical cases must always remain CLOSED
	if caseRecord.IsHistorical && status != models.CaseStatusClosed {
		status = models.CaseStatusClosed
	}

	// Validate client if provided
	if clientID != "" {
		var client models.User
		clientQuery := middleware.GetFirmScopedQuery(c, db.DB)
		if err := clientQuery.
			Where("role = ?", "client").
			Where("is_active = ?", true).
			First(&client, "id = ?", clientID).Error; err != nil {
			if c.Request().Header.Get("HX-Request") == "true" {
				return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Invalid client selected</div>`)
			}
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid client selected")
		}
	}

	// Validate lawyer if provided (only if admin is making the change)
	if assignedToID != "" && currentUser.Role == "admin" {
		var lawyer models.User
		lawyerQuery := middleware.GetFirmScopedQuery(c, db.DB)
		if err := lawyerQuery.
			Where("role IN (?, ?)", "lawyer", "admin").
			Where("is_active = ?", true).
			First(&lawyer, "id = ?", assignedToID).Error; err != nil {
			if c.Request().Header.Get("HX-Request") == "true" {
				return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Invalid lawyer selected</div>`)
			}
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid lawyer selected")
		}
	}

	// Track if status changed
	statusChanged := caseRecord.Status != status
	oldStatus := caseRecord.Status

	// Update case fields
	caseRecord.Status = status
	caseRecord.Description = strings.TrimSpace(description)
	if filingNumber != "" {
		trimmedFilingNumber := strings.TrimSpace(filingNumber)
		caseRecord.FilingNumber = &trimmedFilingNumber
	} else {
		caseRecord.FilingNumber = nil
	}

	// Handle classification changes
	// Only update if classification fields are present in the form (domain_id shouldn't be empty if it's being set)
	// If domain_id is provided or cleared
	if c.Request().Form.Has("domain_id") {
		var newDomainID *string
		if domainID != "" {
			newDomainID = &domainID
		}

		var newBranchID *string
		if branchID != "" {
			newBranchID = &branchID
		}

		// Check if classification changed
		classificationChanged := false

		// Domain changed?
		if (caseRecord.DomainID == nil && newDomainID != nil) ||
			(caseRecord.DomainID != nil && newDomainID == nil) ||
			(caseRecord.DomainID != nil && newDomainID != nil && *caseRecord.DomainID != *newDomainID) {
			classificationChanged = true
		}

		// Branch changed?
		if !classificationChanged {
			if (caseRecord.BranchID == nil && newBranchID != nil) ||
				(caseRecord.BranchID != nil && newBranchID == nil) ||
				(caseRecord.BranchID != nil && newBranchID != nil && *caseRecord.BranchID != *newBranchID) {
				classificationChanged = true
			}
		}

		// Subtypes changed?
		if !classificationChanged {
			// Compare subtypes
			// This is a bit complex due to many-to-many, simplified check:
			if len(caseRecord.Subtypes) != len(subtypeIDs) {
				classificationChanged = true
			} else {
				// Create map of existing subtype IDs
				existingMap := make(map[string]bool)
				for _, s := range caseRecord.Subtypes {
					existingMap[s.ID] = true
				}
				// Check if all new IDs exist
				for _, id := range subtypeIDs {
					if !existingMap[id] {
						classificationChanged = true
						break
					}
				}
			}
		}

		if classificationChanged {
			now := time.Now()
			caseRecord.ClassifiedAt = &now
			caseRecord.ClassifiedBy = &currentUser.ID

			caseRecord.DomainID = newDomainID
			caseRecord.BranchID = newBranchID

			// Update subtypes
			if err := db.DB.Model(&caseRecord).Association("Subtypes").Clear(); err != nil {
				// Log error but continue
			}

			if len(subtypeIDs) > 0 {
				var newSubtypes []models.CaseSubtype
				db.DB.Where("id IN ?", subtypeIDs).Find(&newSubtypes)
				caseRecord.Subtypes = newSubtypes
			}
		}
	}

	// Update client if provided
	if clientID != "" {
		caseRecord.ClientID = clientID
	}

	// Update assigned lawyer if provided (only admins can change this)
	if currentUser.Role == "admin" {
		if assignedToID != "" {
			caseRecord.AssignedToID = &assignedToID
		} else if assignedToID == "" && c.FormValue("clear_lawyer") == "true" {
			// Allow clearing the assigned lawyer
			caseRecord.AssignedToID = nil
		}
	}
	// Non-admins cannot change the assigned lawyer, so the value remains unchanged

	// Handle status change logic
	if statusChanged {
		now := time.Now()
		caseRecord.StatusChangedAt = &now
		caseRecord.StatusChangedBy = &currentUser.ID

		// If status changed to CLOSED, set ClosedAt
		if status == models.CaseStatusClosed && oldStatus != models.CaseStatusClosed {
			caseRecord.ClosedAt = &now
		}

		// If status changed from CLOSED to something else, clear ClosedAt
		if status != models.CaseStatusClosed && oldStatus == models.CaseStatusClosed {
			caseRecord.ClosedAt = nil
		}
	}

	// Save updates
	if err := db.DB.Save(&caseRecord).Error; err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Failed to update case</div>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update case")
	}

	// Audit logging
	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(
		db.DB,
		auditCtx,
		models.AuditActionUpdate,
		"Case",
		caseRecord.ID,
		caseRecord.CaseNumber,
		"Case details updated",
		oldCase,
		caseRecord,
	)

	// Return success response with page reload
	if c.Request().Header.Get("HX-Request") == "true" {
		return c.HTML(http.StatusOK, `
			<div class="p-4 bg-green-500/20 text-green-400 rounded-lg mb-4">
				Case updated successfully!
			</div>
			<script>
				setTimeout(function() {
					window.location.reload();
				}, 1000);
			</script>
		`)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Case updated successfully",
		"case":    caseRecord,
	})
}
