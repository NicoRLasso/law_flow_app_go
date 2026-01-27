package services

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultPDFOptions(t *testing.T) {
	opts := DefaultPDFOptions()
	assert.Equal(t, "portrait", opts.PageOrientation)
	assert.Equal(t, "letter", opts.PageSize)
	assert.Equal(t, 72, opts.MarginTop)
	assert.Equal(t, 72, opts.MarginBottom)
	assert.Equal(t, 72, opts.MarginLeft)
	assert.Equal(t, 72, opts.MarginRight)
}

func TestWrapHTMLForPDF(t *testing.T) {
	content := "<h1>Test Title</h1><p>Test Content</p>"
	html := WrapHTMLForPDF(content)

	assert.Contains(t, html, "<!DOCTYPE html>")
	assert.Contains(t, html, "font-family: \"Times New Roman\"")
	assert.Contains(t, html, content)
}

func TestGeneratePDFSmoke(t *testing.T) {
	// Skip if no chrome/chromium is likely to be available in CI-like environment
	// or if CHROME_PATH is set but invalid.
	chromePath := os.Getenv("CHROME_PATH")
	if chromePath == "" {
		// Try some common paths or just skip if we want to be safe in environments without Chrome
		// For now, if no path is provided, we skip the heavy test to avoid failures in restricted environments.
		t.Skip("Skipping PDF generation test: CHROME_PATH not set")
	}

	html := "<h1>Hello World</h1>"
	opts := DefaultPDFOptions()

	pdf, err := GeneratePDF(html, opts)
	if err != nil {
		// If it's a "file not found" error for the chrome path, we skip instead of fail
		if os.IsNotExist(err) {
			t.Skipf("Skipping: Chrome not found at %s", chromePath)
		}
		t.Errorf("GeneratePDF failed: %v", err)
		return
	}

	assert.NotNil(t, pdf)
	assert.True(t, len(pdf) > 0)
	// PDF header
	assert.Contains(t, string(pdf[:5]), "%PDF-")
}
