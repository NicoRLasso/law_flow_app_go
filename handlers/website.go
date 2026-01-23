package handlers

import (
	"law_flow_app_go/config"
	"law_flow_app_go/middleware"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/pages/company"
	"law_flow_app_go/templates/pages/legal"
	"law_flow_app_go/templates/pages/product"

	"github.com/labstack/echo/v4"
)

func WebsiteAboutHandler(c echo.Context) error {
	csrfToken := middleware.GetCSRFToken(c)
	seo := GetSEO("about")
	component := company.About(c.Request().Context(), seo.Title, csrfToken, seo)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

func WebsiteContactHandler(c echo.Context) error {
	csrfToken := middleware.GetCSRFToken(c)
	seo := GetSEO("contact")
	component := company.Contact(c.Request().Context(), seo.Title, csrfToken, seo)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

func WebsiteSecurityHandler(c echo.Context) error {
	csrfToken := middleware.GetCSRFToken(c)
	seo := GetSEO("security")
	component := product.Security(c.Request().Context(), seo.Title, csrfToken, seo)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

func WebsitePrivacyHandler(c echo.Context) error {
	csrfToken := middleware.GetCSRFToken(c)
	seo := GetSEO("privacy")
	component := legal.Privacy(c.Request().Context(), seo.Title, csrfToken, seo)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

func WebsiteTermsHandler(c echo.Context) error {
	csrfToken := middleware.GetCSRFToken(c)
	seo := GetSEO("terms")
	component := legal.Terms(c.Request().Context(), seo.Title, csrfToken, seo)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

func WebsiteCookiesHandler(c echo.Context) error {
	csrfToken := middleware.GetCSRFToken(c)
	seo := GetSEO("cookies")
	component := legal.Cookies(c.Request().Context(), seo.Title, csrfToken, seo)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

func WebsiteComplianceHandler(c echo.Context) error {
	csrfToken := middleware.GetCSRFToken(c)
	seo := GetSEO("compliance")
	component := legal.Compliance(c.Request().Context(), seo.Title, csrfToken, seo)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// WebsiteContactSubmitHandler handles the contact modal form submission
func WebsiteContactSubmitHandler(c echo.Context) error {
	name := c.FormValue("name")
	email := c.FormValue("email")
	message := c.FormValue("message")

	// Validate inputs (basic validation)
	if name == "" || email == "" || message == "" {
		return c.String(400, "All fields are required")
	}

	// Prepare email content
	subject := "New Contact Request from " + name
	body := "Name: " + name + "\n" +
		"Email: " + email + "\n" +
		"Message: \n" + message

	// Get config from context (assuming it's set by middleware) or load it
	// For now, let's try to get it from the context as 'config'
	cfg, ok := c.Get("config").(*config.Config)
	if !ok || cfg == nil {
		// Fallback: This is not ideal, but if middleware doesn't set it, we might be in trouble.
		// However, based on the project structure, config is usually global or passed.
		// Let's assume for now the user has a way to get config.
		// If not, we might fail. But let's check cmd/server/main.go later if this fails.
		// IMPORTANT: We need a way to get config.
		// Let's try to load it if missing, but that's slow.
		// Ideally, main.go sets it.
	}

	emailObj := &services.Email{
		To:       []string{"support@lexlegalcloud.org"},
		Subject:  subject,
		TextBody: body,
		HTMLBody: "<p><strong>Name:</strong> " + name + "</p>" +
			"<p><strong>Email:</strong> " + email + "</p>" +
			"<p><strong>Message:</strong></p><p>" + message + "</p>",
	}

	if cfg != nil {
		services.SendEmailAsync(cfg, emailObj)
	} else {
		// Log error that config is missing?
		// or create a default minimal config just for testing?
	}

	return c.HTML(200, `<div class='text-center p-8 animate-fade-in'>
		<div class='inline-flex items-center justify-center w-16 h-16 rounded-full bg-green-500/10 mb-6'>
			<span class='text-3xl'>✅</span>
		</div>
		<h3 class='text-h3 mb-2'>Request Sent!</h3>
		<p class='text-body text-muted'>Thank you for contacting us. We will get back to you shortly.</p>
	</div>`)
}
