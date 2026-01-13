package handlers

import (
	"law_flow_app_go/config"
	"law_flow_app_go/services"
	"net/http"

	"github.com/labstack/echo/v4"
)

// TestEmailHandler sends a test email (development only)
func TestEmailHandler(c echo.Context) error {
	cfg := config.Load()

	// Only allow in development environment
	if cfg.Environment != "development" {
		return echo.NewHTTPError(http.StatusForbidden, "This endpoint is only available in development mode")
	}

	// Get recipient from query param, default to configured SMTP username
	recipient := c.QueryParam("to")
	if recipient == "" {
		recipient = cfg.SMTPUsername
	}

	// Build test email
	email := &services.Email{
		To:      []string{recipient},
		Subject: "LawFlow App - Test Email",
		HTMLBody: `
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<style>
		body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; line-height: 1.6; color: #333; }
		.container { max-width: 600px; margin: 0 auto; padding: 20px; }
		.header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; text-align: center; border-radius: 8px 8px 0 0; }
		.content { background: #ffffff; padding: 30px; border: 1px solid #e0e0e0; border-top: none; }
		.footer { background: #f5f5f5; padding: 20px; text-align: center; font-size: 12px; color: #666; border-radius: 0 0 8px 8px; }
		.success { background: #10b981; color: white; padding: 15px; border-radius: 5px; margin: 20px 0; }
		h1 { margin: 0; font-size: 28px; }
	</style>
</head>
<body>
	<div class="container">
		<div class="header">
			<h1>✅ Email Configuration Test</h1>
		</div>
		<div class="content">
			<div class="success">
				<strong>Success!</strong> Your email configuration is working correctly.
			</div>
			<p>This is a test email from LawFlow App to verify your SMTP configuration.</p>
			<p><strong>Configuration Details:</strong></p>
			<ul>
				<li>SMTP Host: ` + cfg.SMTPHost + `</li>
				<li>SMTP Port: ` + cfg.SMTPPort + `</li>
				<li>From Address: ` + cfg.EmailFrom + `</li>
				<li>From Name: ` + cfg.EmailFromName + `</li>
			</ul>
			<p>If you received this email, your email service is configured correctly and ready to send emails to users.</p>
		</div>
		<div class="footer">
			<p>&copy; 2026 LawFlow App. All rights reserved.</p>
		</div>
	</div>
</body>
</html>
`,
		TextBody: `Email Configuration Test

Success! Your email configuration is working correctly.

This is a test email from LawFlow App to verify your SMTP configuration.

Configuration Details:
- SMTP Host: ` + cfg.SMTPHost + `
- SMTP Port: ` + cfg.SMTPPort + `
- From Address: ` + cfg.EmailFrom + `
- From Name: ` + cfg.EmailFromName + `

If you received this email, your email service is configured correctly and ready to send emails to users.

---
© 2026 LawFlow App. All rights reserved.
`,
	}

	// Send email synchronously for testing (so we can return errors)
	if err := services.SendEmail(cfg, email); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error":   "Failed to send test email",
			"details": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message":   "Test email sent successfully",
		"recipient": recipient,
		"smtp_host": cfg.SMTPHost,
	})
}
