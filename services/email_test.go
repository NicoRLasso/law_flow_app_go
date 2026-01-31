package services

import (
	"law_flow_app_go/config"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadTemplate(t *testing.T) {
	// Setup temporary templates
	tmpTemplatesDir := "templates/emails"
	err := os.MkdirAll(tmpTemplatesDir, 0755)
	assert.NoError(t, err)
	defer os.RemoveAll("templates")

	// Create a base English template
	baseHTML := "<html><body>Hello {{.UserName}}</body></html>"
	baseText := "Hello {{.UserName}}"
	os.WriteFile(filepath.Join(tmpTemplatesDir, "test_template.html"), []byte(baseHTML), 0644)
	os.WriteFile(filepath.Join(tmpTemplatesDir, "test_template.txt"), []byte(baseText), 0644)

	// Create a localized Spanish template
	esHTML := "<html><body>Hola {{.UserName}}</body></html>"
	esText := "Hola {{.UserName}}"
	os.WriteFile(filepath.Join(tmpTemplatesDir, "test_template_es.html"), []byte(esHTML), 0644)
	os.WriteFile(filepath.Join(tmpTemplatesDir, "test_template_es.txt"), []byte(esText), 0644)

	type data struct {
		UserName string
	}
	tplData := data{UserName: "John"}

	t.Run("Load Base Template", func(t *testing.T) {
		html, text, err := loadTemplate("test_template", "en", tplData)
		assert.NoError(t, err)
		assert.Contains(t, html, "Hello John")
		assert.Contains(t, text, "Hello John")
	})

	t.Run("Load Localized Template", func(t *testing.T) {
		html, text, err := loadTemplate("test_template", "es", tplData)
		assert.NoError(t, err)
		assert.Contains(t, html, "Hola John")
		assert.Contains(t, text, "Hola John")
	})

	t.Run("Fallback to Base when Localized Missing", func(t *testing.T) {
		html, text, err := loadTemplate("test_template", "fr", tplData)
		assert.NoError(t, err)
		assert.Contains(t, html, "Hello John") // Should fallback to base
		assert.Contains(t, text, "Hello John")
	})

	t.Run("Template Not Found", func(t *testing.T) {
		_, _, err := loadTemplate("non_existent", "en", tplData)
		assert.Error(t, err)
	})
}

func TestBuildEmailWithFallback(t *testing.T) {
	// Setup temporary templates
	tmpTemplatesDir := "templates/emails"
	os.MkdirAll(tmpTemplatesDir, 0755)
	defer os.RemoveAll("templates")

	os.WriteFile(filepath.Join(tmpTemplatesDir, "test_build.html"), []byte("HTML {{.Val}}"), 0644)
	os.WriteFile(filepath.Join(tmpTemplatesDir, "test_build.txt"), []byte("Text {{.Val}}"), 0644)

	email := buildEmailWithFallback("test_build", "en", map[string]string{"Val": "OK"}, "test@example.com")
	assert.Equal(t, []string{"test@example.com"}, email.To)
	assert.Equal(t, "HTML OK", email.HTMLBody)
	assert.Equal(t, "Text OK", email.TextBody)
}

func TestSendEmail_TestMode(t *testing.T) {
	cfg := &config.Config{
		EmailTestMode: true,
	}
	email := &Email{
		To:       []string{"test@example.com"},
		Subject:  "Test",
		HTMLBody: "Body",
	}

	err := SendEmail(cfg, email)
	assert.NoError(t, err)
}

func TestSendEmail_NoApiKey(t *testing.T) {
	cfg := &config.Config{
		EmailTestMode: false,
		ResendAPIKey:  "",
	}
	email := &Email{
		To:       []string{"test@example.com"},
		Subject:  "Test",
		HTMLBody: "Body",
	}

	err := SendEmail(cfg, email)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "RESEND_API_KEY not configured")
}

func TestSendEmail_NoBody(t *testing.T) {
	cfg := &config.Config{
		EmailTestMode: false,
		ResendAPIKey:  "key",
	}
	email := &Email{
		To:      []string{"test@example.com"},
		Subject: "Test",
	}

	err := SendEmail(cfg, email)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "email must have either HTMLBody or TextBody")
}

func TestBuildEmailFunctions(t *testing.T) {
	// Setup temporary templates
	tmpTemplatesDir := "templates/emails"
	os.MkdirAll(tmpTemplatesDir, 0755)
	defer os.RemoveAll("templates")

	// Create dummy templates for all builders
	templates := []string{
		"welcome", "firm_setup", "password_reset", "collaborator_added",
		"appointment_confirmation", "appointment_reminder", "appointment_cancelled",
		"lawyer_notification", "new_user_welcome", "support_notification",
	}
	for _, tpl := range templates {
		os.WriteFile(filepath.Join(tmpTemplatesDir, tpl+".html"), []byte("HTML Body {{.}}"), 0644)
		os.WriteFile(filepath.Join(tmpTemplatesDir, tpl+".txt"), []byte("Text Body {{.}}"), 0644)
	}

	t.Run("BuildWelcomeEmail", func(t *testing.T) {
		email := BuildWelcomeEmail("test@test.com", "John", "en")
		assert.Equal(t, []string{"test@test.com"}, email.To)
	})

	t.Run("BuildCollaboratorAddedEmail", func(t *testing.T) {
		email := BuildCollaboratorAddedEmail("collab@test.com", "Collab", "SVC-001", "Client", "Lawyer", "en")
		assert.Equal(t, []string{"collab@test.com"}, email.To)
	})

	t.Run("BuildAppointmentConfirmationEmail", func(t *testing.T) {
		data := AppointmentConfirmationEmailData{FirmName: "Test Firm"}
		email := BuildAppointmentConfirmationEmail("client@test.com", data, "en")
		assert.Equal(t, []string{"client@test.com"}, email.To)
	})

	t.Run("BuildPasswordResetEmail", func(t *testing.T) {
		email := BuildPasswordResetEmail("test@test.com", "John", "http://reset", "1 hour", "en")
		assert.Equal(t, []string{"test@test.com"}, email.To)
	})

	t.Run("BuildNewUserWelcomeEmail", func(t *testing.T) {
		email := BuildNewUserWelcomeEmail("test@test.com", "John", "pass123", "http://login", "en")
		assert.Equal(t, []string{"test@test.com"}, email.To)
		assert.Contains(t, email.HTMLBody, "pass123")
	})

	t.Run("BuildAppointmentCancelledEmail", func(t *testing.T) {
		data := AppointmentCancelledEmailData{FirmName: "Test Firm", Date: "2026-01-01"}
		email := BuildAppointmentCancelledEmail("client@test.com", data, "en")
		assert.Equal(t, []string{"client@test.com"}, email.To)
	})

	t.Run("BuildSupportTicketNotificationEmail", func(t *testing.T) {
		email := BuildSupportTicketNotificationEmail("admin@test.com", "Admin", "User", "user@test.com", "T-001", "Help", "Body", "en")
		assert.Equal(t, []string{"admin@test.com"}, email.To)
	})
}

func TestTruncate(t *testing.T) {
	s := "Hello World"
	assert.Equal(t, "Hello", truncate(s, 5))
	assert.Equal(t, "Hello World", truncate(s, 20))
}

func TestSendEmailAsync(t *testing.T) {
	cfg := &config.Config{EmailTestMode: true}
	email := &Email{To: []string{"test@test.com"}, Subject: "Async", HTMLBody: "Hello"}

	// Just ensure it doesn't panic
	SendEmailAsync(cfg, email)
}
