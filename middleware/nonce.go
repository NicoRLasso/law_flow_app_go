package middleware

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/labstack/echo/v4"
)

type contextKey string

const NonceKey contextKey = "csp_nonce"

// GenerateNonce creates a random nonce string
func GenerateNonce() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// CSPNonce middleware generates a nonce for each request and adds it to the context
func CSPNonce() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			nonce, err := GenerateNonce()
			if err != nil {
				c.Logger().Errorf("Failed to generate nonce: %v", err)
				nonce = "fallback-nonce-value" // Should rarely happen, but prevents crash
			}

			// Add to Echo context (for handlers)
			c.Set(string(NonceKey), nonce)

			// Add to Request context (for Templ)
			ctx := context.WithValue(c.Request().Context(), NonceKey, nonce)
			c.SetRequest(c.Request().WithContext(ctx))

			// Construct CSP with Nonce
			// Note: 'unsafe-eval' is currently preserved for Alpine.js support.
			// We remove 'unsafe-inline' and replace it with 'nonce-{nonce}'.
			csp := fmt.Sprintf("default-src 'self'; script-src 'self' 'nonce-%s' 'unsafe-eval' https://unpkg.com https://static.cloudflareinsights.com https://challenges.cloudflare.com; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; img-src 'self' data:; font-src 'self' https://fonts.gstatic.com; connect-src 'self' https://unpkg.com https://cloudflareinsights.com https://challenges.cloudflare.com; frame-src https://challenges.cloudflare.com", nonce)

			c.Response().Header().Set("Content-Security-Policy", csp)

			return next(c)
		}
	}
}

// GetNonce retrieves the nonce from the context
func GetNonce(ctx context.Context) string {
	if val, ok := ctx.Value(NonceKey).(string); ok {
		return val
	}
	return ""
}
