package handlers

import (
	"fmt"
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/pages"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// ProfileSettingsPageHandler renders the profile settings page
func ProfileSettingsPageHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)

	// Preload document type for display
	var user models.User
	if err := db.DB.Preload("DocumentType").First(&user, "id = ?", currentUser.ID).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load user data")
	}

	// Render the profile settings page
	csrfToken := middleware.GetCSRFToken(c)
	component := pages.ProfileSettings(c.Request().Context(), "Profile Settings | LexLegal Cloud", csrfToken, &user, firm)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// UpdateProfileHandler updates the current user's profile information
func UpdateProfileHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)

	var user models.User
	if err := db.DB.First(&user, "id = ?", currentUser.ID).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "User not found")
	}

	// Read form values
	name := strings.TrimSpace(c.FormValue("name"))
	email := strings.TrimSpace(c.FormValue("email"))
	language := strings.TrimSpace(c.FormValue("language"))
	phoneNumber := strings.TrimSpace(c.FormValue("phone_number"))
	address := strings.TrimSpace(c.FormValue("address"))
	documentTypeID := strings.TrimSpace(c.FormValue("document_type_id"))
	documentNumber := strings.TrimSpace(c.FormValue("document_number"))

	// Validate required fields
	if name == "" || email == "" || language == "" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="text-red-500 text-sm mt-2">Name, email, and language are required</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "Name, email, and language are required")
	}

	// Update fields
	user.Name = name
	user.Email = email
	user.Language = language

	// Handle optional fields
	if phoneNumber != "" {
		user.PhoneNumber = &phoneNumber
	} else {
		user.PhoneNumber = nil
	}

	if address != "" {
		user.Address = &address
	} else {
		user.Address = nil
	}

	if documentTypeID != "" {
		user.DocumentTypeID = &documentTypeID
	} else {
		user.DocumentTypeID = nil
	}

	if documentNumber != "" {
		user.DocumentNumber = &documentNumber
	} else {
		user.DocumentNumber = nil
	}

	// Save changes
	if err := db.DB.Save(&user).Error; err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div class="text-red-500 text-sm mt-2">Failed to update profile. Please try again.</div>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update profile")
	}

	// Update language cookie
	middleware.SetLanguageCookie(c, user.Language)

	// Log security event
	services.LogSecurityEvent("PROFILE_UPDATED", user.ID, "User updated their profile")

	// Check if this is an HTMX request
	if c.Request().Header.Get("HX-Request") == "true" {
		// Trigger a page reload to apply language changes if language changed
		if language != middleware.GetLocale(c) {
			c.Response().Header().Set("HX-Redirect", c.Request().Referer())
		}
		return c.HTML(http.StatusOK, `<div class="text-green-500 text-sm mt-2">Profile updated successfully!</div>`)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Profile updated successfully",
	})
}

// ChangePasswordHandler changes the current user's password
func ChangePasswordHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)

	var user models.User
	if err := db.DB.First(&user, "id = ?", currentUser.ID).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "User not found")
	}

	// Read form values
	currentPassword := c.FormValue("current_password")
	newPassword := c.FormValue("new_password")
	confirmPassword := c.FormValue("confirm_password")

	// Validate required fields
	if currentPassword == "" || newPassword == "" || confirmPassword == "" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="text-red-500 text-sm mt-2">All password fields are required</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "All password fields are required")
	}

	// Verify current password
	if !services.CheckPassword(currentPassword, user.Password) {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusUnauthorized, `<div class="text-red-500 text-sm mt-2">Current password is incorrect</div>`)
		}
		return echo.NewHTTPError(http.StatusUnauthorized, "Current password is incorrect")
	}

	// Validate new password matches confirmation
	if newPassword != confirmPassword {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="text-red-500 text-sm mt-2">New passwords do not match</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "New passwords do not match")
	}

	// Validate password strength
	if err := services.ValidatePassword(newPassword); err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, fmt.Sprintf(`<div class="text-red-500 text-sm mt-2">%s</div>`, err.Error()))
		}
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Hash new password
	hashedPassword, err := services.HashPassword(newPassword)
	if err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div class="text-red-500 text-sm mt-2">Failed to update password. Please try again.</div>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update password")
	}

	// Update password
	user.Password = hashedPassword
	if err := db.DB.Save(&user).Error; err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div class="text-red-500 text-sm mt-2">Failed to update password. Please try again.</div>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update password")
	}

	// Log security event
	services.LogSecurityEvent("PASSWORD_CHANGED", user.ID, "User changed their password")

	// Check if this is an HTMX request
	if c.Request().Header.Get("HX-Request") == "true" {
		// Clear the form and show success message
		c.Response().Header().Set("HX-Trigger", "password-changed")
		return c.HTML(http.StatusOK, `<div class="text-green-500 text-sm mt-2">Password changed successfully!</div>`)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Password changed successfully",
	})
}
