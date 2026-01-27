package middleware

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestComputeFileHash(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.css")
	content := []byte("body { color: red; }")
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	// Test with existing file
	hash := computeFileHash(tmpFile)
	if hash == "" {
		t.Error("expected a hash, got empty string")
	}
	if len(hash) != 8 {
		t.Errorf("expected hash length 8, got %d", len(hash))
	}

	// Test with non-existent file
	hash = computeFileHash("non_existent_file.css")
	if hash != "" {
		t.Errorf("expected empty hash for non-existent file, got %s", hash)
	}
}

func TestGetVersionsDefault(t *testing.T) {
	ctx := context.Background()

	// Reset versions to ensure we test default values (since these are global vars)
	// We can't easily reset assetVersionsOnce, but we can check if they are "1" initially
	// because InitAssetVersions hasn't been called in this test yet (or it might have if other tests ran)

	// Just check if they return something reasonable (either a hash or "1")
	if v := GetCSSVersion(ctx); v == "" {
		t.Error("GetCSSVersion returned empty string")
	}
	if v := GetFaviconVersion(ctx); v == "" {
		t.Error("GetFaviconVersion returned empty string")
	}
	if v := GetAppJSVersion(ctx); v == "" {
		t.Error("GetAppJSVersion returned empty string")
	}
	if v := GetMarqueeJSVersion(ctx); v == "" {
		t.Error("GetMarqueeJSVersion returned empty string")
	}
	if v := GetEditorJSVersion(ctx, "nonexistent.js"); v != "1" {
		t.Errorf("expected default version '1' for nonexistent editor file, got %s", v)
	}
}

func TestInitAssetVersions(t *testing.T) {
	// Since InitAssetVersions uses assetVersionsOnce, we can only truly test it once per process.
	// But we can simulate the file structure and see if it picks up something.

	// Create required directories
	os.MkdirAll("static/css", 0755)
	os.MkdirAll("static/images", 0755)
	os.MkdirAll("static/js/editor", 0755)

	// Create mock files
	os.WriteFile("static/css/style.css", []byte("css"), 0644)
	os.WriteFile("static/images/favicon.png", []byte("img"), 0644)
	os.WriteFile("static/js/app.js", []byte("js"), 0644)
	os.WriteFile("static/js/marquee.js", []byte("js"), 0644)
	os.WriteFile("static/js/template-editor.js", []byte("js"), 0644)
	os.WriteFile("static/js/editor/config.js", []byte("js"), 0644)

	defer func() {
		// Clean up is a bit dangerous if we are in a real project dir
		// Better to just leave it or use a specific test execution environment if possible
		// But usually tests run in the package dir.
	}()

	InitAssetVersions()

	ctx := context.Background()
	if GetCSSVersion(ctx) == "1" {
		t.Error("expected computed CSS version, got default '1'")
	}
	if GetFaviconVersion(ctx) == "1" {
		t.Error("expected computed Favicon version, got default '1'")
	}
	if GetEditorJSVersion(ctx, "config.js") == "1" {
		t.Error("expected computed Editor JS version, got default '1'")
	}
}
