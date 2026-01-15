package handlers

import (
	"fmt"
	"law_flow_app_go/config"
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/pages"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// ForgotPasswordHandler renders the forgot password page
func ForgotPasswordHandler(c echo.Context) error {
	csrfToken := middleware.GetCSRFToken(c)
	component := pages.ForgotPassword(c.Request().Context(), "Forgot Password | Law Flow", csrfToken)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// ForgotPasswordPostHandler handles the forgot password form submission
func ForgotPasswordPostHandler(c echo.Context) error {
	email := strings.TrimSpace(c.FormValue("email"))

	// Validate email input
	if email == "" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="text-red-500 text-sm">Email address is required</div>`)
		}
		return c.Redirect(http.StatusSeeOther, "/forgot-password")
	}

	// Generate reset token (returns nil if email doesn't exist - security best practice)
	resetToken, err := services.GenerateResetToken(db.DB, email)
	if err != nil {
		// Log error but don't reveal to user
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div class="text-red-500 text-sm">An error occurred. Please try again.</div>`)
		}
		return c.Redirect(http.StatusSeeOther, "/forgot-password")
	}

	// Send password reset email asynchronously if token was created
	if resetToken != nil {
		cfg := config.Load()

		// Build reset link
		baseURL := c.Scheme() + "://" + c.Request().Host
		resetLink := fmt.Sprintf("%s/reset-password?token=%s", baseURL, resetToken.Token)

		// Format expiration time
		expiresAt := resetToken.ExpiresAt.Format("January 2, 2006 at 3:04 PM MST")

		// Get user name from token
		userName := email
		if resetToken.User != nil && resetToken.User.Name != "" {
			userName = resetToken.User.Name
		}

		// Build and send email
		emailMsg := services.BuildPasswordResetEmail(email, userName, resetLink, expiresAt)
		services.SendEmailAsync(cfg, emailMsg)
	}

	// Always show success message (security best practice - don't reveal if email exists)
	successMsg := `<div class="rounded-md bg-green-50 p-4">
		<div class="flex">
			<div class="flex-shrink-0">
				<svg class="h-5 w-5 text-green-400" viewBox="0 0 20 20" fill="currentColor">
					<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"/>
				</svg>
			</div>
			<div class="ml-3">
				<p class="text-sm font-medium text-green-800">
					If an account exists with that email, you will receive a password reset link shortly.
				</p>
			</div>
		</div>
	</div>`

	if c.Request().Header.Get("HX-Request") == "true" {
		return c.HTML(http.StatusOK, successMsg)
	}

	return c.Redirect(http.StatusSeeOther, "/login")
}

// ResetPasswordHandler renders the reset password page
func ResetPasswordHandler(c echo.Context) error {
	csrfToken := middleware.GetCSRFToken(c)
	token := c.QueryParam("token")

	if token == "" {
		component := pages.ResetPassword(c.Request().Context(), "Reset Password | Law Flow", csrfToken, "", false)
		return component.Render(c.Request().Context(), c.Response().Writer)
	}

	// Validate token
	_, err := services.ValidateResetToken(db.DB, token)
	validToken := err == nil

	component := pages.ResetPassword(c.Request().Context(), "Reset Password | Law Flow", csrfToken, token, validToken)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// ResetPasswordPostHandler handles the reset password form submission
func ResetPasswordPostHandler(c echo.Context) error {
	token := c.FormValue("token")
	password := c.FormValue("password")
	passwordConfirm := c.FormValue("password_confirm")

	// Validate inputs
	if token == "" || password == "" || passwordConfirm == "" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="text-red-500 text-sm">All fields are required</div>`)
		}
		return c.Redirect(http.StatusSeeOther, "/reset-password?token="+token)
	}

	// Check passwords match
	if password != passwordConfirm {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="text-red-500 text-sm">Passwords do not match</div>`)
		}
		return c.Redirect(http.StatusSeeOther, "/reset-password?token="+token)
	}

	// Reset password
	if err := services.ResetPassword(db.DB, token, password); err != nil {
		errorMsg := fmt.Sprintf(`<div class="text-red-500 text-sm">%s</div>`, err.Error())
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, errorMsg)
		}
		return c.Redirect(http.StatusSeeOther, "/reset-password?token="+token)
	}

	// Success - redirect to login with success message
	successMsg := `<div class="rounded-md bg-green-50 p-4 mb-4">
		<div class="flex">
			<div class="flex-shrink-0">
				<svg class="h-5 w-5 text-green-400" viewBox="0 0 20 20" fill="currentColor">
					<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"/>
				</svg>
			</div>
			<div class="ml-3">
				<p class="text-sm font-medium text-green-800">
					Your password has been reset successfully. Please log in with your new password.
				</p>
			</div>
		</div>
	</div>`

	if c.Request().Header.Get("HX-Request") == "true" {
		// Return success message and trigger redirect after delay
		c.Response().Header().Set("HX-Redirect", "/login")
		return c.HTML(http.StatusOK, successMsg)
	}

	return c.Redirect(http.StatusSeeOther, "/login")
}
