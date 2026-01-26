package handlers

import (
	"fmt"
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/services/i18n"
	"law_flow_app_go/services/jobs"
	"law_flow_app_go/templates/partials"
	"log"
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

	// Handle Historical Case Logic
	// 1. If case is Historical and we are trying to change status from CLOSED to something else (Reopening)
	if caseRecord.IsHistorical && status != models.CaseStatusClosed {
		// Only Admin and Lawyer can reopen historical cases
		if currentUser.Role != "admin" && currentUser.Role != "lawyer" {
			if c.Request().Header.Get("HX-Request") == "true" {
				return c.HTML(http.StatusForbidden, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Permission denied: Only Admins and Lawyers can reopen historical cases</div>`)
			}
			return echo.NewHTTPError(http.StatusForbidden, "Permission denied: Only Admins and Lawyers can reopen historical cases")
		}
		// If authorized, verify we aren't blocked by other logic, then unmark historical
		caseRecord.IsHistorical = false
	} else if !caseRecord.IsHistorical && status == models.CaseStatusClosed {
		// 2. If case is NOT Historical but is being CLOSED, mark as Historical
		caseRecord.IsHistorical = true
	} else if caseRecord.IsHistorical && status == models.CaseStatusClosed {
		// 3. Case is Historical and stays Closed - ensure it stays true (redundant but safe)
		caseRecord.IsHistorical = true
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
		// Check for duplicate filing number
		if strings.Contains(err.Error(), "UNIQUE constraint failed: cases.filing_number") ||
			strings.Contains(err.Error(), "cases_filing_number_key") ||
			strings.Contains(err.Error(), "idx_firm_filing_number") {
			errMsg := i18n.T(c.Request().Context(), "case.edit.error.duplicate_radicado")
			if c.Request().Header.Get("HX-Request") == "true" {
				return c.HTML(http.StatusConflict, fmt.Sprintf(`<div class="p-4 bg-red-500/20 text-red-400 rounded-lg flex items-center gap-2"><i data-lucide="alert-circle" class="w-5 h-5"></i> <span>%s</span></div><script>lucide.createIcons();</script>`, errMsg))
			}
			return echo.NewHTTPError(http.StatusConflict, errMsg)
		}

		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Failed to update case: `+err.Error()+`</div>`)
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

	// Trigger async judicial process update if filing number changed or was just set
	oldFilingNumber := ""
	if oldCase.FilingNumber != nil {
		oldFilingNumber = *oldCase.FilingNumber
	}
	newFilingNumber := ""
	if caseRecord.FilingNumber != nil {
		newFilingNumber = *caseRecord.FilingNumber
	}

	// Trigger async judicial process update ONLY if filing number was just added (was empty before)
	// Updates to existing numbers will be picked up by the nightly job to avoid spamming/issues
	if newFilingNumber != "" && oldFilingNumber == "" {
		log.Printf("[HANDLER] New filing number added: %s. Scheduling initial async update for CaseID: %s", newFilingNumber, caseRecord.ID)
		go func(id string) {
			// Small delay to ensure transaction committed if any (though GORM Save is normally blocking until committed)
			time.Sleep(1 * time.Second)
			log.Printf("[HANDLER] Starting async update for CaseID: %s", id)
			fmt.Println(">> [DEBUG] Goroutine started for CaseID:", id) // Force stdout
			if err := jobs.UpdateSingleCase(id); err != nil {
				log.Printf("[HANDLER] Async update failed for CaseID %s: %v", id, err)
				fmt.Printf(">> [DEBUG] UpdateSingleCase failed: %v\n", err)
			} else {
				log.Printf("[HANDLER] Async update completed for CaseID: %s", id)
				fmt.Println(">> [DEBUG] UpdateSingleCase completed successfully")
			}
		}(caseRecord.ID)
	}

	// Determine success message
	successMsg := "Case updated successfully!"
	if newFilingNumber != "" && oldFilingNumber == "" {
		successMsg = "Case updated! Judicial sync started in background."
	}

	// Return success response with page reload
	if c.Request().Header.Get("HX-Request") == "true" {
		return c.HTML(http.StatusOK, fmt.Sprintf(`
			<div class="p-4 bg-green-500/20 text-green-400 rounded-lg mb-4">
				%s
			</div>
			<script>
				setTimeout(function() {
					window.location.reload();
				}, 2000);
			</script>
		`, successMsg))
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Case updated successfully",
		"case":    caseRecord,
	})
}
