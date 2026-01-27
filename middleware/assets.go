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
	cssVersion     string
	cssVersionOnce sync.Once
)

// InitAssetVersions computes file hashes for cache busting at startup
func InitAssetVersions() {
	cssVersionOnce.Do(func() {
		cssVersion = computeFileHash("static/css/style.css")
		if cssVersion == "" {
			// Fallback to a default version if hash fails
			cssVersion = "1"
		}
		log.Printf("[INFO] CSS version initialized: %s", cssVersion[:8])
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
