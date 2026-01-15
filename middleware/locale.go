package middleware

import (
	"context"
	"law_flow_app_go/config"
	"law_flow_app_go/services/i18n"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// Locale middleware handles language detection and persistence.
// Priority:
// 1. Query param "lang" (sets cookie)
// 2. Cookie "lang"
// 3. Accept-Language header
// 4. Default ("en")
func Locale(cfg *config.Config) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Check query param
			lang := c.QueryParam("lang")
			if lang != "" {
				// Validate supported languages (basic check)
				if lang != "en" && lang != "es" {
					lang = "en"
				}

				// Set cookie
				cookie := new(http.Cookie)
				cookie.Name = "lang"
				cookie.Value = lang
				cookie.Expires = time.Now().Add(24 * 365 * time.Hour) // 1 year
				cookie.Path = "/"
				cookie.HttpOnly = true
				cookie.SameSite = http.SameSiteLaxMode
				if cfg.Environment == "production" {
					cookie.Secure = true
				}
				c.SetCookie(cookie)
			} else {
				// Check cookie
				cookie, err := c.Cookie("lang")
				if err == nil {
					lang = cookie.Value
				}
			}

			// Check header if still empty
			if lang == "" {
				accept := c.Request().Header.Get("Accept-Language")
				if strings.Contains(accept, "es") {
					lang = "es"
				} else {
					lang = "en"
				}
			}

			// Provide context with locale
			// We set it in both echo context and request context
			c.Set("locale", lang)

			// Update request context for Templ (standard context)
			ctx := context.WithValue(c.Request().Context(), i18n.LocaleContextKey, lang)
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}

// SetLanguageCookie sets the language cookie
func SetLanguageCookie(c echo.Context, lang string) {
	// Get config from context if available
	cfg, ok := c.Get("config").(*config.Config)

	cookie := new(http.Cookie)
	cookie.Name = "lang"
	cookie.Value = lang
	cookie.Expires = time.Now().Add(24 * 365 * time.Hour) // 1 year
	cookie.Path = "/"
	cookie.HttpOnly = true
	cookie.SameSite = http.SameSiteLaxMode

	if ok && cfg.Environment == "production" {
		cookie.Secure = true
	}

	c.SetCookie(cookie)
}

// GetLocale returns the current locale from context
func GetLocale(c echo.Context) string {
	val := c.Get("locale")
	if lang, ok := val.(string); ok {
		return lang
	}
	return "en"
}
