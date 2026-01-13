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

	component := pages.FirmSetup("Setup Your Firm | Law Flow", user)
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
