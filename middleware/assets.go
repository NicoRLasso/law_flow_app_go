package middleware

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"
	"log"
	"os"
	"sync"
)

var (
	cssVersion        string
	faviconVersion    string
	appJSVersion      string
	marqueeJSVersion  string
	editorJSVersions  map[string]string
	assetVersionsOnce sync.Once
)

// InitAssetVersions computes file hashes for cache busting at startup
func InitAssetVersions() {
	assetVersionsOnce.Do(func() {
		// CSS
		cssVersion = computeFileHash("static/css/style.css")
		if cssVersion == "" {
			cssVersion = "1"
		}
		log.Printf("[INFO] CSS version initialized: %s", cssVersion)

		// Favicon
		faviconVersion = computeFileHash("static/images/favicon.png")
		if faviconVersion == "" {
			faviconVersion = "1"
		}
		log.Printf("[INFO] Favicon version initialized: %s", faviconVersion)

		// App JS
		appJSVersion = computeFileHash("static/js/app.js")
		if appJSVersion == "" {
			appJSVersion = "1"
		}
		log.Printf("[INFO] App JS version initialized: %s", appJSVersion)

		// Marquee JS
		marqueeJSVersion = computeFileHash("static/js/marquee.js")
		if marqueeJSVersion == "" {
			marqueeJSVersion = "1"
		}
		log.Printf("[INFO] Marquee JS version initialized: %s", marqueeJSVersion)

		// Editor JS files
		editorJSVersions = make(map[string]string)
		editorFiles := []string{
			"config.js",
			"selection.js",
			"menu.js",
			"formatting.js",
			"paging.js",
			"auto-paging.js",
			"zoom.js",
		}
		for _, file := range editorFiles {
			version := computeFileHash("static/js/editor/" + file)
			if version == "" {
				version = "1"
			}
			editorJSVersions[file] = version
		}
		// Template editor main file
		templateEditorVersion := computeFileHash("static/js/template-editor.js")
		if templateEditorVersion == "" {
			templateEditorVersion = "1"
		}
		editorJSVersions["template-editor.js"] = templateEditorVersion
		log.Printf("[INFO] Editor JS versions initialized: %d files", len(editorJSVersions))
	})
}

// computeFileHash returns the first 8 characters of the MD5 hash of a file
func computeFileHash(path string) string {
	file, err := os.Open(path)
	if err != nil {
		log.Printf("[WARNING] Failed to open file for hashing %s: %v", path, err)
		return ""
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		log.Printf("[WARNING] Failed to hash file %s: %v", path, err)
		return ""
	}

	// Return first 8 chars of the hash for brevity
	return hex.EncodeToString(hash.Sum(nil))[:8]
}

// GetCSSVersion returns the CSS file version hash for cache busting
// Note: ctx parameter is for API consistency with other middleware helpers,
// but the version is computed once at startup and is global
func GetCSSVersion(ctx context.Context) string {
	if cssVersion == "" {
		return "1"
	}
	return cssVersion
}

// GetFaviconVersion returns the favicon file version hash for cache busting
func GetFaviconVersion(ctx context.Context) string {
	if faviconVersion == "" {
		return "1"
	}
	return faviconVersion
}

// GetAppJSVersion returns the app.js file version hash for cache busting
func GetAppJSVersion(ctx context.Context) string {
	if appJSVersion == "" {
		return "1"
	}
	return appJSVersion
}

// GetMarqueeJSVersion returns the marquee.js file version hash for cache busting
func GetMarqueeJSVersion(ctx context.Context) string {
	if marqueeJSVersion == "" {
		return "1"
	}
	return marqueeJSVersion
}

// GetEditorJSVersion returns the version hash for a specific editor JS file
func GetEditorJSVersion(ctx context.Context, filename string) string {
	if editorJSVersions == nil {
		return "1"
	}
	if version, ok := editorJSVersions[filename]; ok {
		return version
	}
	return "1"
}
