package middleware

import (
	"law_flow_app_go/config"
	"law_flow_app_go/db"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"net/http"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
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
	// Get config to check environment
	var isProduction bool
	if cfg, ok := c.Get("config").(*config.Config); ok {
		isProduction = cfg.Environment == "production"
	}

	cookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isProduction,
		SameSite: http.SameSiteLaxMode,
	}
	c.SetCookie(cookie)
}

// RequireFirm ensures the user has a firm assigned
func RequireFirm() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := GetCurrentUser(c)

			if user == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "Not authenticated")
			}

			if !user.HasFirm() {
				// Redirect to firm setup
				if c.Request().Header.Get("HX-Request") == "true" {
					c.Response().Header().Set("HX-Redirect", "/firm/setup")
					return c.NoContent(http.StatusSeeOther)
				}
				return c.Redirect(http.StatusSeeOther, "/firm/setup")
			}

			return next(c)
		}
	}
}

// GetFirmScopedQuery returns a GORM query scoped to the current user's firm
func GetFirmScopedQuery(c echo.Context, db *gorm.DB) *gorm.DB {
	currentUser := GetCurrentUser(c)
	if currentUser == nil || currentUser.FirmID == nil {
		// Return query that matches nothing
		return db.Where("1 = 0")
	}

	return db.Where("firm_id = ?", *currentUser.FirmID)
}

// CanAccessUser checks if the current user can access another user's data
func CanAccessUser(c echo.Context, targetUserID string) bool {
	currentUser := GetCurrentUser(c)
	if currentUser == nil {
		return false
	}

	// Admins can access all users in their firm
	if currentUser.Role == "admin" {
		return true
	}

	// Users can always access their own data
	if currentUser.ID == targetUserID {
		return true
	}

	// Lawyers and staff can view (but not edit) users in their firm
	// This is enforced at the handler level
	return false
}

// CanModifyUser checks if the current user can modify another user's data
func CanModifyUser(c echo.Context, targetUserID string) bool {
	currentUser := GetCurrentUser(c)
	if currentUser == nil {
		return false
	}

	// Only admins can modify other users
	if currentUser.Role == "admin" {
		return true
	}

	// Users can modify their own profile
	return currentUser.ID == targetUserID
}
