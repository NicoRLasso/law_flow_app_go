package middleware

import (
	"law_flow_app_go/db"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"net/http"

	"github.com/labstack/echo/v4"
)

const (
	// SessionCookieName is the name of the session cookie
	SessionCookieName = "law_flow_session"
	// ContextKeyUser is the context key for the authenticated user
	ContextKeyUser = "user"
	// ContextKeyFirm is the context key for the user's firm
	ContextKeyFirm = "firm"
	// ContextKeySession is the context key for the session
	ContextKeySession = "session"
)

// RequireAuth is middleware that requires authentication
func RequireAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get session cookie
			cookie, err := c.Cookie(SessionCookieName)
			if err != nil {
				// No cookie, redirect to login
				if c.Request().Header.Get("HX-Request") == "true" {
					c.Response().Header().Set("HX-Redirect", "/login")
					return c.NoContent(http.StatusUnauthorized)
				}
				return c.Redirect(http.StatusSeeOther, "/login")
			}

			// Validate session
			session, err := services.ValidateSession(db.DB, cookie.Value)
			if err != nil {
				// Invalid or expired session, clear cookie and redirect
				clearSessionCookie(c)
				if c.Request().Header.Get("HX-Request") == "true" {
					c.Response().Header().Set("HX-Redirect", "/login")
					return c.NoContent(http.StatusUnauthorized)
				}
				return c.Redirect(http.StatusSeeOther, "/login")
			}

			// Check if user is active
			if !session.User.IsActive {
				clearSessionCookie(c)
				if c.Request().Header.Get("HX-Request") == "true" {
					c.Response().Header().Set("HX-Redirect", "/login")
					return c.NoContent(http.StatusUnauthorized)
				}
				return c.Redirect(http.StatusSeeOther, "/login")
			}

			// Store user, firm, and session in context
			c.Set(ContextKeyUser, &session.User)
			c.Set(ContextKeyFirm, &session.Firm)
			c.Set(ContextKeySession, session)

			return next(c)
		}
	}
}

// RequireRole is middleware that requires specific roles
func RequireRole(roles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := c.Get(ContextKeyUser).(*models.User)

			// Check if user has one of the required roles
			hasRole := false
			for _, role := range roles {
				if user.Role == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				return echo.NewHTTPError(http.StatusForbidden, "Insufficient permissions")
			}

			return next(c)
		}
	}
}

// GetCurrentUser retrieves the current user from context
func GetCurrentUser(c echo.Context) *models.User {
	user, ok := c.Get(ContextKeyUser).(*models.User)
	if !ok {
		return nil
	}
	return user
}

// GetCurrentFirm retrieves the current firm from context
func GetCurrentFirm(c echo.Context) *models.Firm {
	firm, ok := c.Get(ContextKeyFirm).(*models.Firm)
	if !ok {
		return nil
	}
	return firm
}

// clearSessionCookie clears the session cookie
func clearSessionCookie(c echo.Context) {
	cookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	}
	c.SetCookie(cookie)
}
