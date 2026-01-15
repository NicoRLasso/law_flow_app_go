package handlers

import (
	"law_flow_app_go/config"
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"net/http"

	"github.com/labstack/echo/v4"
)

// AddCaseCollaboratorHandler adds a lawyer as a collaborator to a case
func AddCaseCollaboratorHandler(c echo.Context) error {
	caseID := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)

	// Only admins can add collaborators
	if currentUser.Role != "admin" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusForbidden, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Only admins can manage collaborators</div>`)
		}
		return echo.NewHTTPError(http.StatusForbidden, "Only admins can manage collaborators")
	}

	// Get user ID from form
	userID := c.FormValue("user_id")
	if userID == "" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">User ID is required</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "User ID is required")
	}

	// Fetch case with firm scoping and relationships for email
	var caseRecord models.Case
	query := middleware.GetFirmScopedQuery(c, db.DB)
	if err := query.Preload("Collaborators").Preload("Client").Preload("AssignedTo").First(&caseRecord, "id = ?", caseID).Error; err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusNotFound, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Case not found</div>`)
		}
		return echo.NewHTTPError(http.StatusNotFound, "Case not found")
	}

	// Verify the user is a valid lawyer/admin in the same firm
	var user models.User
	userQuery := middleware.GetFirmScopedQuery(c, db.DB)
	if err := userQuery.
		Where("role IN (?, ?)", "lawyer", "admin").
		Where("is_active = ?", true).
		First(&user, "id = ?", userID).Error; err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Invalid user selected</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid user selected")
	}

	// Check if user is already the assigned lawyer
	if caseRecord.AssignedToID != nil && *caseRecord.AssignedToID == userID {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-yellow-500/20 text-yellow-400 rounded-lg">This user is already the primary assigned lawyer</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "This user is already the primary assigned lawyer")
	}

	// Check if user is already a collaborator
	for _, collab := range caseRecord.Collaborators {
		if collab.ID == userID {
			if c.Request().Header.Get("HX-Request") == "true" {
				return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-yellow-500/20 text-yellow-400 rounded-lg">This user is already a collaborator</div>`)
			}
			return echo.NewHTTPError(http.StatusBadRequest, "This user is already a collaborator")
		}
	}

	// Add collaborator using GORM's association
	if err := db.DB.Model(&caseRecord).Association("Collaborators").Append(&user); err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Failed to add collaborator</div>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to add collaborator")
	}

	// Send email notification to collaborator (async)
	cfg := c.Get("config").(*config.Config)
	clientName := caseRecord.Client.Name
	assignedLawyer := "Unassigned"
	if caseRecord.AssignedTo != nil {
		assignedLawyer = caseRecord.AssignedTo.Name
	}
	email := services.BuildCollaboratorAddedEmail(user.Email, user.Name, caseRecord.CaseNumber, clientName, assignedLawyer)
	services.SendEmailAsync(cfg, email)

	// Return success and trigger page reload
	if c.Request().Header.Get("HX-Request") == "true" {
		return c.HTML(http.StatusOK, `
			<div class="p-4 bg-green-500/20 text-green-400 rounded-lg mb-4">
				Collaborator added successfully!
			</div>
			<script>
				setTimeout(function() {
					window.location.reload();
				}, 1000);
			</script>
		`)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Collaborator added successfully",
	})
}

// RemoveCaseCollaboratorHandler removes a collaborator from a case
func RemoveCaseCollaboratorHandler(c echo.Context) error {
	caseID := c.Param("id")
	userID := c.Param("userId")
	currentUser := middleware.GetCurrentUser(c)

	// Only admins can remove collaborators
	if currentUser.Role != "admin" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusForbidden, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Only admins can manage collaborators</div>`)
		}
		return echo.NewHTTPError(http.StatusForbidden, "Only admins can manage collaborators")
	}

	// Fetch case with firm scoping
	var caseRecord models.Case
	query := middleware.GetFirmScopedQuery(c, db.DB)
	if err := query.First(&caseRecord, "id = ?", caseID).Error; err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusNotFound, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Case not found</div>`)
		}
		return echo.NewHTTPError(http.StatusNotFound, "Case not found")
	}

	// Fetch user to remove
	var user models.User
	userQuery := middleware.GetFirmScopedQuery(c, db.DB)
	if err := userQuery.First(&user, "id = ?", userID).Error; err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusNotFound, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">User not found</div>`)
		}
		return echo.NewHTTPError(http.StatusNotFound, "User not found")
	}

	// Remove collaborator using GORM's association
	if err := db.DB.Model(&caseRecord).Association("Collaborators").Delete(&user); err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Failed to remove collaborator</div>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to remove collaborator")
	}

	// Return success and trigger page reload
	if c.Request().Header.Get("HX-Request") == "true" {
		return c.HTML(http.StatusOK, `
			<div class="p-4 bg-green-500/20 text-green-400 rounded-lg mb-4">
				Collaborator removed successfully!
			</div>
			<script>
				setTimeout(function() {
					window.location.reload();
				}, 1000);
			</script>
		`)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Collaborator removed successfully",
	})
}

// GetAvailableCollaboratorsHandler returns lawyers that can be added as collaborators
func GetAvailableCollaboratorsHandler(c echo.Context) error {
	caseID := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)

	// Only admins can fetch available collaborators
	if currentUser.Role != "admin" {
		return echo.NewHTTPError(http.StatusForbidden, "Only admins can manage collaborators")
	}

	// Fetch case with firm scoping
	var caseRecord models.Case
	query := middleware.GetFirmScopedQuery(c, db.DB)
	if err := query.Preload("Collaborators").First(&caseRecord, "id = ?", caseID).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Case not found")
	}

	// Get IDs to exclude (assigned lawyer + existing collaborators)
	excludeIDs := []string{}
	if caseRecord.AssignedToID != nil {
		excludeIDs = append(excludeIDs, *caseRecord.AssignedToID)
	}
	for _, collab := range caseRecord.Collaborators {
		excludeIDs = append(excludeIDs, collab.ID)
	}

	// Fetch available lawyers
	var users []models.User
	userQuery := middleware.GetFirmScopedQuery(c, db.DB)
	userQuery = userQuery.
		Where("role IN (?, ?)", "lawyer", "admin").
		Where("is_active = ?", true)

	if len(excludeIDs) > 0 {
		userQuery = userQuery.Where("id NOT IN (?)", excludeIDs)
	}

	if err := userQuery.Order("name ASC").Find(&users).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch users")
	}

	return c.JSON(http.StatusOK, users)
}
