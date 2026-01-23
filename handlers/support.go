package handlers

import (
	"law_flow_app_go/config"
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/services/i18n"
	"law_flow_app_go/templates/pages"
	"net/http"

	"github.com/labstack/echo/v4"
)

// SupportPageHandler renders the support page
func SupportPageHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c) // Might be nil for some users if logic changes, but for now we assume firm or at least auth
	csrfToken := middleware.GetCSRFToken(c)

	// Initial contact form data
	formData := pages.SupportContactFormData{
		Name:  user.Name,
		Email: user.Email,
	}

	var successMsgPtr *string
	if c.QueryParam("success") == "true" {
		msg := i18n.T(c.Request().Context(), "support.success.submitted")
		successMsgPtr = &msg
	}

	var firmUsers []models.User
	if user.Role == "admin" && firm != nil {
		// Fetch firm users for filter (excluding password for safety, though not strictly needed here)
		db.DB.Select("id, name").Where("firm_id = ?", firm.ID).Find(&firmUsers)
	}

	component := pages.Support(c.Request().Context(), "Support | LexLegal Cloud", csrfToken, user, firm, formData, successMsgPtr, firmUsers)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// SubmitSupportRequestHandler handles the support form submission
func SubmitSupportRequestHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	// firm := middleware.GetCurrentFirm(c)
	csrfToken := middleware.GetCSRFToken(c)
	cfg := c.Get("config").(*config.Config)

	// Parse form
	subject := c.FormValue("subject")
	message := c.FormValue("message")

	// Validate
	errors := make(map[string]string)
	if subject == "" {
		errors["subject"] = i18n.T(c.Request().Context(), "support.error.subject_required")
	}
	if message == "" {
		errors["message"] = i18n.T(c.Request().Context(), "support.error.message_required")
	}

	if len(errors) > 0 {
		// Re-render with errors
		firm := middleware.GetCurrentFirm(c)
		formData := pages.SupportContactFormData{
			Name:    user.Name,
			Email:   user.Email,
			Subject: subject,
			Message: message,
			Errors:  errors,
		}
		component := pages.Support(c.Request().Context(), "Support | LexLegal Cloud", csrfToken, user, firm, formData, nil, nil)
		return component.Render(c.Request().Context(), c.Response().Writer)
	}

	// 1. Save to Database
	ticket := models.SupportTicket{
		UserID:  user.ID,
		Subject: subject,
		Message: message,
		Status:  "open",
	}

	if err := db.DB.Create(&ticket).Error; err != nil {
		c.Logger().Error("Failed to create support ticket:", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to submit request")
	}

	// 2. Notify Superadmins via Email
	go func() {
		// Find all superadmins
		var superadmins []models.User
		if err := db.DB.Where("role = ?", "superadmin").Find(&superadmins).Error; err != nil {
			c.Logger().Error("Failed to fetch superadmins for notification:", err)
			return
		}

		// Send email to each superadmin
		for _, admin := range superadmins {
			email := services.BuildSupportTicketNotificationEmail(
				admin.Email,
				admin.Name,
				user.Name,
				user.Email,
				ticket.ID,
				subject,
				message,
				admin.Language,
			)
			if err := services.SendEmail(cfg, email); err != nil {
				c.Logger().Error("Failed to send support notification email:", err)
			}
		}
	}()

	// 3. Redirect to support page with success param (PRG pattern to prevent double submission)
	return c.Redirect(http.StatusSeeOther, "/support?success=true&tab=contact")
}
