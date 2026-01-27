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

func TestTruncate(t *testing.T) {
	s := "Hello World"
	assert.Equal(t, "Hello", truncate(s, 5))
	assert.Equal(t, "Hello World", truncate(s, 20))
}
