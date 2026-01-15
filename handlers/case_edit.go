package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
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

	// Render the edit modal
	component := partials.CaseEditModal(caseRecord, clients, lawyers)
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

	// Get form values
	status := c.FormValue("status")
	description := c.FormValue("description")
	clientID := c.FormValue("client_id")
	assignedToID := c.FormValue("assigned_to_id")

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

	// Validate lawyer if provided
	if assignedToID != "" {
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

	// Update client if provided
	if clientID != "" {
		caseRecord.ClientID = clientID
	}

	// Update assigned lawyer if provided
	if assignedToID != "" {
		caseRecord.AssignedToID = &assignedToID
	} else if assignedToID == "" && c.FormValue("clear_lawyer") == "true" {
		// Allow clearing the assigned lawyer
		caseRecord.AssignedToID = nil
	}

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
