package handlers

import (
	"law_flow_app_go/config"
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/pages"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// FirmSetupHandler renders the firm setup page
func FirmSetupHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	// If user already has a firm, redirect to dashboard
	if user.HasFirm() {
		return c.Redirect(http.StatusSeeOther, "/dashboard")
	}

	csrfToken := middleware.GetCSRFToken(c)
	component := pages.FirmSetup(c.Request().Context(), "Setup Your Firm | Law Flow", csrfToken, user)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// FirmSetupPostHandler handles the firm setup form submission
func FirmSetupPostHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Not authenticated")
	}

	// Defense in depth: double-check user doesn't have a firm
	if user.HasFirm() {
		// Redirect to dashboard if user already has a firm
		if c.Request().Header.Get("HX-Request") == "true" {
			c.Response().Header().Set("HX-Redirect", "/dashboard")
			return c.NoContent(http.StatusSeeOther)
		}
		return c.Redirect(http.StatusSeeOther, "/dashboard")
	}

	// Parse form data
	name := strings.TrimSpace(c.FormValue("name"))
	country := strings.TrimSpace(c.FormValue("country"))
	timezone := strings.TrimSpace(c.FormValue("timezone"))
	address := strings.TrimSpace(c.FormValue("address"))
	city := strings.TrimSpace(c.FormValue("city"))
	phone := strings.TrimSpace(c.FormValue("phone"))
	description := strings.TrimSpace(c.FormValue("description"))
	billingEmail := strings.TrimSpace(c.FormValue("billing_email"))
	infoEmail := strings.TrimSpace(c.FormValue("info_email"))
	noreplyEmail := strings.TrimSpace(c.FormValue("noreply_email"))

	// Validate required fields
	if name == "" || country == "" || billingEmail == "" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="text-red-500 text-sm mt-2">Firm name, country, and billing email are required</div>`)
		}
		return c.Redirect(http.StatusSeeOther, "/firm/setup")
	}

	// Set default timezone if not provided
	if timezone == "" {
		timezone = "UTC"
	}

	// Create the firm
	firm := &models.Firm{
		Name:         name,
		Country:      country,
		Timezone:     timezone,
		Address:      address,
		City:         city,
		Phone:        phone,
		Description:  description,
		BillingEmail: billingEmail,
		InfoEmail:    infoEmail,
		NoreplyEmail: noreplyEmail,
	}

	if err := db.DB.Create(firm).Error; err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div class="text-red-500 text-sm mt-2">Failed to create firm. Please try again.</div>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create firm")
	}

	// Assign firm to user and set as admin
	user.FirmID = &firm.ID
	user.Role = "admin" // First user of a firm becomes admin
	if err := db.DB.Save(user).Error; err != nil {
		// Rollback: delete the firm if we can't assign it
		db.DB.Delete(firm)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to assign firm to user")
	}

	// Seed default choice categories and options for the firm
	if err := services.SeedDefaultChoices(db.DB, firm.ID, firm.Country); err != nil {
		// Log error but don't fail the firm creation
		c.Logger().Errorf("Failed to seed default choices for firm %s: %v", firm.ID, err)
	}

	// Seed case classifications for the firm
	if err := services.SeedCaseClassifications(db.DB, firm.ID, firm.Country); err != nil {
		// Log error but don't fail the firm creation
		c.Logger().Errorf("Failed to seed case classifications for firm %s: %v", firm.ID, err)
	}

	// Send firm setup confirmation email asynchronously (non-blocking)
	cfg := config.Load()
	if user.Email != "" {
		userName := user.Name
		if userName == "" {
			userName = user.Email
		}
		email := services.BuildFirmSetupEmail(user.Email, userName, firm.Name)
		services.SendEmailAsync(cfg, email)
	}

	// Redirect to dashboard
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Redirect", "/dashboard")
		return c.NoContent(http.StatusOK)
	}

	return c.Redirect(http.StatusSeeOther, "/dashboard")
}

// FirmSettingsPageHandler renders the firm settings page (admin only)
func FirmSettingsPageHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)
	csrfToken := middleware.GetCSRFToken(c)

	// Render the firm settings page
	component := pages.FirmSettings(c.Request().Context(), "Firm Settings | Law Flow", csrfToken, user, firm)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// UpdateFirmHandler updates the firm information (admin only)
func UpdateFirmHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)
	updateType := c.FormValue("update_type")

	// Helper function for HTMX error response
	htmxError := func(msg string) error {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="text-red-500 text-sm mt-2">`+msg+`</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, msg)
	}

	if updateType == "general" {
		name := strings.TrimSpace(c.FormValue("name"))
		country := strings.TrimSpace(c.FormValue("country"))
		timezone := strings.TrimSpace(c.FormValue("timezone"))

		if name == "" || country == "" {
			return htmxError("Firm name and country are required")
		}

		if timezone == "" {
			timezone = "UTC"
		}

		firm.Name = name
		firm.Country = country
		firm.Timezone = timezone
		firm.Address = strings.TrimSpace(c.FormValue("address"))
		firm.City = strings.TrimSpace(c.FormValue("city"))
		firm.Phone = strings.TrimSpace(c.FormValue("phone"))
		firm.Description = strings.TrimSpace(c.FormValue("description"))

	} else if updateType == "email" {
		billingEmail := strings.TrimSpace(c.FormValue("billing_email"))

		if billingEmail == "" {
			return htmxError("Billing email is required")
		}

		firm.BillingEmail = billingEmail
		firm.InfoEmail = strings.TrimSpace(c.FormValue("info_email"))
		firm.NoreplyEmail = strings.TrimSpace(c.FormValue("noreply_email"))

	} else {
		// Fallback for legacy requests or unknown types
		// Try to parse everything but only if critical fields are present
		name := strings.TrimSpace(c.FormValue("name"))
		billingEmail := strings.TrimSpace(c.FormValue("billing_email"))

		if name != "" && billingEmail != "" {
			firm.Name = name
			firm.BillingEmail = billingEmail
			// Update other fields if they look present?
			// Safer to just require update_type for robust partial updates.
			// But to be safe against the error currently seen (missing one or the other),
			// we just return error if we can't determine intent.
			return htmxError("Invalid update request type")
		}
		return htmxError("Invalid update request")
	}

	// Save changes
	if err := db.DB.Save(firm).Error; err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div class="text-red-500 text-sm mt-2">Failed to update firm settings. Please try again.</div>`)
		}
		c.Logger().Errorf("Error saving firm: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update firm settings")
	}

	// Log security event
	services.LogSecurityEvent("FIRM_UPDATED", currentUser.ID, "Admin updated firm settings ("+updateType+"): "+firm.ID)

	// Check if this is an HTMX request
	if c.Request().Header.Get("HX-Request") == "true" {
		return c.HTML(http.StatusOK, `<div class="text-green-500 text-sm mt-2">Firm settings updated successfully!</div>`)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Firm settings updated successfully",
	})
}
