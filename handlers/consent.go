package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/components"
	"net/http"

	"github.com/labstack/echo/v4"
)

// AcceptConsentHandler records the user's acceptance of data processing consent
func AcceptConsentHandler(c echo.Context) error {
	user := c.Get("user").(*models.User)
	ctx := c.Request().Context()

	consentType := c.FormValue("consent_type")
	if consentType == "" {
		consentType = "DATA_PROCESSING"
	}

	// Log the consent
	err := services.LogConsent(
		ctx,
		db.DB,
		user.ID,
		user.Email,
		user.FirmID,
		models.ConsentType(consentType),
		true, // granted
		c.RealIP(),
		c.Request().UserAgent(),
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to record consent")
	}

	// For HTMX requests, just return empty (modal will be deleted)
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Trigger", "consentAccepted")
		return c.NoContent(http.StatusOK)
	}

	// For regular requests, redirect to dashboard
	return c.Redirect(http.StatusSeeOther, "/dashboard")
}

// GetConsentModalHandler returns the consent modal if user needs to accept
func GetConsentModalHandler(c echo.Context) error {
	user, ok := c.Get("user").(*models.User)
	if !ok || user == nil {
		// If no user is logged in, no consent is needed
		return c.NoContent(http.StatusOK)
	}

	ctx := c.Request().Context()
	csrfToken := middleware.GetCSRFToken(c)

	// Check if user has valid consent
	hasConsent, err := services.HasValidConsent(db.DB, user.ID, models.ConsentTypeDataProcessing)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check consent status")
	}

	// If user already has consent, return empty
	if hasConsent {
		return c.NoContent(http.StatusOK)
	}

	// Return consent modal
	return render(c, components.ConsentModal(ctx, csrfToken))
}

// RevokeConsentHandler allows users to revoke their consent
func RevokeConsentHandler(c echo.Context) error {
	user := c.Get("user").(*models.User)
	ctx := c.Request().Context()

	consentType := c.FormValue("consent_type")
	if consentType == "" {
		consentType = "DATA_PROCESSING"
	}

	// Log the revocation (creates a new record with granted=false)
	err := services.LogConsent(
		ctx,
		db.DB,
		user.ID,
		user.Email,
		user.FirmID,
		models.ConsentType(consentType),
		false, // revoked
		c.RealIP(),
		c.Request().UserAgent(),
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to record consent revocation")
	}

	// Logout the user after revoking consent
	return c.Redirect(http.StatusSeeOther, "/logout")
}
