package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/services"
	"net/http"

	"github.com/labstack/echo/v4"
)

// SuperadminAnonymizeUser performs a GDPR-compliant anonymization of a user
func SuperadminAnonymizeUser(c echo.Context) error {
	id := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)

	// Validate admin permission (Double check, though middleware covers it)
	if !currentUser.IsSuperadmin() {
		return echo.NewHTTPError(http.StatusForbidden, "Unauthorized")
	}

	// Perform anonymization
	if err := services.AnonymizeUser(db.DB, id, currentUser.ID); err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div class="alert alert-error">Failed to anonymize user</div>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to anonymize user")
	}

	// Response
	if c.Request().Header.Get("HX-Request") == "true" {
		// Return refreshed table
		return SuperadminGetUsersListHTMX(c)
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "User anonymized"})
}
