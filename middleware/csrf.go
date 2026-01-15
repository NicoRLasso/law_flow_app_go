package middleware

import (
	"github.com/labstack/echo/v4"
)

// GetCSRFToken retrieves the CSRF token from the Echo context
// This token should be included in forms and AJAX requests
func GetCSRFToken(c echo.Context) string {
	token := c.Get("csrf")
	if token == nil {
		return ""
	}
	if tokenStr, ok := token.(string); ok {
		return tokenStr
	}
	return ""
}
