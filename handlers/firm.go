package handlers

import (
	"fmt"
	"law_flow_app_go/config"
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/components"
	"law_flow_app_go/templates/pages"
	"net/http"
	"os"
	"strings"
	"time"

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
	component := pages.FirmSetup(c.Request().Context(), "Setup Your Firm | LexLegal Cloud", csrfToken, user)
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

	// Force timezone based on location
	if defaultTz := services.GetDefaultTimezone(country); defaultTz != "" {
		timezone = defaultTz
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

	// Create trial subscription for the new firm
	if err := services.CreateTrialSubscription(db.DB, firm.ID); err != nil {
		c.Logger().Errorf("Failed to create trial subscription for firm %s: %v", firm.ID, err)
	}

	// Initialize usage tracking
	if _, err := services.RecalculateFirmUsage(db.DB, firm.ID); err != nil {
		c.Logger().Errorf("Failed to initialize usage tracking for firm %s: %v", firm.ID, err)
	}

	// Send firm setup confirmation email asynchronously (non-blocking)
	cfg := config.Load()
	if user.Email != "" {
		userName := user.Name
		if userName == "" {
			userName = user.Email
		}
		userLang := user.Language
		if userLang == "" {
			userLang = "es"
		}
		email := services.BuildFirmSetupEmail(user.Email, userName, firm.Name, userLang)
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

	// Get subscription info for billing tab
	subscriptionInfo, _ := services.GetFirmSubscriptionInfo(db.DB, firm.ID)

	// Get available add-ons for the purchase modal
	availableAddOns, _ := services.GetAvailableAddOns(db.DB)

	// Render the firm settings page
	component := pages.FirmSettings(c.Request().Context(), "Firm Settings | LexLegal Cloud", csrfToken, user, firm, subscriptionInfo, availableAddOns)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// FirmBillingTabHandler renders only the billing tab component for HTMX updates
func FirmBillingTabHandler(c echo.Context) error {
	firm := middleware.GetCurrentFirm(c)
	if firm == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Firm not found")
	}

	subscriptionInfo, _ := services.GetFirmSubscriptionInfo(db.DB, firm.ID)

	component := components.BillingTab(c.Request().Context(), subscriptionInfo)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// UpdateFirmHandler updates the firm information (admin only)
func UpdateFirmHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)
	updateType := c.FormValue("update_type")

	// Capture old values for audit
	oldValues := map[string]interface{}{
		"name":          firm.Name,
		"country":       firm.Country,
		"timezone":      firm.Timezone,
		"address":       firm.Address,
		"city":          firm.City,
		"phone":         firm.Phone,
		"description":   firm.Description,
		"billing_email": firm.BillingEmail,
		"info_email":    firm.InfoEmail,
		"noreply_email": firm.NoreplyEmail,
	}

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

		// Check if name has changed
		if firm.Name != name {
			firm.Name = name
			// Regenerate slug
			firm.Slug = models.GenerateSlug(db.DB, name)
		}

		// Force timezone based on location
		if defaultTz := services.GetDefaultTimezone(country); defaultTz != "" {
			timezone = defaultTz
		}

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
	services.LogSecurityEvent(db.DB, "FIRM_UPDATED", currentUser.ID, "Admin updated firm settings ("+updateType+"): "+firm.ID)

	// Log audit event
	services.LogAuditEvent(db.DB, services.AuditContext{
		UserID:    currentUser.ID,
		UserName:  currentUser.Name,
		UserRole:  currentUser.Role,
		FirmID:    firm.ID,
		FirmName:  firm.Name,
		IPAddress: c.RealIP(),
		UserAgent: c.Request().UserAgent(),
	}, models.AuditActionUpdate, "firm", firm.ID, firm.Name, "Updated firm settings ("+updateType+")", oldValues, firm)

	// Check if this is an HTMX request
	if c.Request().Header.Get("HX-Request") == "true" {
		successMsg := `<div class="text-green-500 text-sm mt-2">Firm settings updated successfully!</div>`

		// If slug changed, append OOB swap for the details tab
		if updateType == "general" && firm.Slug != "" {
			// We can assume slug might have changed if we are here and saved successfully.
			// Ideally we should have tracked if it changed.
			// But sending the OOB update even if it didn't change is harmless (idempotent).
			// Just to be safe and simple, we send it.
			successMsg += fmt.Sprintf(`<span id="firm-slug-display" hx-swap-oob="true" class="font-mono text-xs text-foreground">%s</span>`, firm.Slug)
		}

		return c.HTML(http.StatusOK, successMsg)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Firm settings updated successfully",
	})
}

// UploadFirmLogoHandler handles firm logo file upload (admin only)
func UploadFirmLogoHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)

	// Get the uploaded file
	file, err := c.FormFile("logo")
	if err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="text-red-500 text-sm mt-2">Please select a file to upload</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "No file uploaded")
	}

	// Validate file size (max 2MB)
	if file.Size > 2*1024*1024 {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="text-red-500 text-sm mt-2">File size must be less than 2MB</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "File size must be less than 2MB")
	}

	// Validate file type
	allowedTypes := map[string]bool{
		"image/png":     true,
		"image/jpeg":    true,
		"image/jpg":     true,
		"image/svg+xml": true,
	}

	contentType := file.Header.Get("Content-Type")
	if !allowedTypes[contentType] {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="text-red-500 text-sm mt-2">Only PNG, JPG, JPEG, and SVG files are allowed</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid file type. Only PNG, JPG, JPEG, and SVG are allowed")
	}

	// Capture old values for audit
	oldLogoURL := firm.LogoURL

	// Delete old logo if exists (handle both R2 and local paths)
	if oldLogoURL != "" {
		ctx := c.Request().Context()
		// Check if it's an R2 URL or local path
		if strings.HasPrefix(oldLogoURL, "/static/uploads/logos/") {
			// Old local path - delete from filesystem
			oldPath := "." + oldLogoURL
			if err := os.Remove(oldPath); err != nil && !os.IsNotExist(err) {
				c.Logger().Warnf("Failed to delete old local logo: %v", err)
			}
		} else {
			// R2 path - extract key and delete
			oldKey := extractStorageKeyFromURL(oldLogoURL)
			if oldKey != "" {
				if err := services.Storage.Delete(ctx, oldKey); err != nil {
					c.Logger().Warnf("Failed to delete old logo from storage: %v", err)
				}
			}
		}
	}

	// Generate storage key for the logo
	storageKey := services.GenerateFirmLogoKey(firm.ID, file.Filename)

	// Upload to storage (R2 or local depending on configuration)
	ctx := c.Request().Context()
	result, err := services.Storage.Upload(ctx, file, storageKey)
	if err != nil {
		c.Logger().Errorf("Failed to upload logo to storage: %v", err)
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div class="text-red-500 text-sm mt-2">Failed to upload logo. Please try again.</div>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to upload logo")
	}

	// Update firm's logo URL
	logoURL := result.URL
	firm.LogoURL = logoURL

	if err := db.DB.Save(firm).Error; err != nil {
		// Clean up the uploaded file if DB update fails
		if delErr := services.Storage.Delete(ctx, storageKey); delErr != nil {
			c.Logger().Warnf("Failed to cleanup uploaded logo after DB error: %v", delErr)
		}
		c.Logger().Errorf("Failed to update firm logo URL: %v", err)
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div class="text-red-500 text-sm mt-2">Failed to save logo. Please try again.</div>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update firm")
	}

	// Log security event
	services.LogSecurityEvent(db.DB, "FIRM_LOGO_UPLOADED", currentUser.ID, "Admin uploaded firm logo: "+firm.ID)

	// Log audit event
	services.LogAuditEvent(db.DB, services.AuditContext{
		UserID:    currentUser.ID,
		UserName:  currentUser.Name,
		UserRole:  currentUser.Role,
		FirmID:    firm.ID,
		FirmName:  firm.Name,
		IPAddress: c.RealIP(),
		UserAgent: c.Request().UserAgent(),
	}, models.AuditActionUpdate, "firm", firm.ID, firm.Name, "Uploaded firm logo",
		map[string]interface{}{"logo_url": oldLogoURL},
		map[string]interface{}{"logo_url": logoURL})

	// Return HTMX response with logo preview
	if c.Request().Header.Get("HX-Request") == "true" {
		html := fmt.Sprintf(`
			<div id="logo-preview-container" class="space-y-4">
				<div class="flex items-center gap-4">
					<div class="w-20 h-20 rounded-xl bg-white/5 border border-white/10 flex items-center justify-center overflow-hidden">
						<img src="%s?t=%d" alt="Firm Logo" class="max-w-full max-h-full object-contain"/>
					</div>
					<div class="flex flex-col gap-2">
						<span class="text-sm text-green-500">Logo uploaded successfully!</span>
						<button
							type="button"
							hx-delete="/api/firm/logo"
							hx-target="#logo-preview-container"
							hx-swap="outerHTML"
							class="text-sm text-red-400 hover:text-red-300 flex items-center gap-1"
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path></svg>
							Remove Logo
						</button>
					</div>
				</div>
			</div>
		`, logoURL, time.Now().Unix())
		return c.HTML(http.StatusOK, html)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message":  "Logo uploaded successfully",
		"logo_url": logoURL,
	})
}

// DeleteFirmLogoHandler deletes the firm's logo (admin only)
func DeleteFirmLogoHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)

	if firm.LogoURL == "" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="text-red-500 text-sm mt-2">No logo to delete</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "No logo to delete")
	}

	// Capture old value for audit
	oldLogoURL := firm.LogoURL

	// Delete the file (handle both R2 and local paths)
	ctx := c.Request().Context()
	if strings.HasPrefix(oldLogoURL, "/static/uploads/logos/") {
		// Old local path - delete from filesystem
		filePath := "." + oldLogoURL
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			c.Logger().Warnf("Failed to delete local logo file: %v", err)
		}
	} else {
		// R2 path - extract key and delete from storage
		storageKey := extractStorageKeyFromURL(oldLogoURL)
		if storageKey != "" {
			if err := services.Storage.Delete(ctx, storageKey); err != nil {
				c.Logger().Warnf("Failed to delete logo from storage: %v", err)
			}
		}
	}

	// Clear logo URL in database
	firm.LogoURL = ""
	if err := db.DB.Save(firm).Error; err != nil {
		c.Logger().Errorf("Failed to clear firm logo URL: %v", err)
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div class="text-red-500 text-sm mt-2">Failed to delete logo. Please try again.</div>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update firm")
	}

	// Log security event
	services.LogSecurityEvent(db.DB, "FIRM_LOGO_DELETED", currentUser.ID, "Admin deleted firm logo: "+firm.ID)

	// Log audit event
	services.LogAuditEvent(db.DB, services.AuditContext{
		UserID:    currentUser.ID,
		UserName:  currentUser.Name,
		UserRole:  currentUser.Role,
		FirmID:    firm.ID,
		FirmName:  firm.Name,
		IPAddress: c.RealIP(),
		UserAgent: c.Request().UserAgent(),
	}, models.AuditActionUpdate, "firm", firm.ID, firm.Name, "Deleted firm logo",
		map[string]interface{}{"logo_url": oldLogoURL},
		map[string]interface{}{"logo_url": ""})

	// Return HTMX response with empty logo placeholder
	if c.Request().Header.Get("HX-Request") == "true" {
		html := `
			<div id="logo-preview-container" class="space-y-4">
				<div class="flex items-center gap-4">
					<div class="w-20 h-20 rounded-xl bg-white/5 border border-white/10 border-dashed flex items-center justify-center">
						<svg class="w-8 h-8 text-muted-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"></path>
						</svg>
					</div>
					<div class="text-sm text-muted-foreground">
						No logo uploaded
					</div>
				</div>
			</div>
		`
		return c.HTML(http.StatusOK, html)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Logo deleted successfully",
	})
}

// extractStorageKeyFromURL extracts the storage key from an R2 public URL
// For example, "https://cdn.example.com/logos/firm123.png" -> "logos/firm123.png"
func extractStorageKeyFromURL(url string) string {
	cfg := config.Load()
	if cfg.R2PublicURL == "" {
		return ""
	}
	publicURL := strings.TrimSuffix(cfg.R2PublicURL, "/")
	if strings.HasPrefix(url, publicURL+"/") {
		return strings.TrimPrefix(url, publicURL+"/")
	}
	return ""
}
