package middleware

import (
	"law_flow_app_go/db"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

const (
	// ContextKeySubscription is the context key for the subscription
	ContextKeySubscription = "subscription"
	// ContextKeySubscriptionInfo is the context key for full subscription info
	ContextKeySubscriptionInfo = "subscription_info"
)

// RequireActiveSubscription ensures the firm has an active subscription
func RequireActiveSubscription() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			firm := GetCurrentFirm(c)
			if firm == nil {
				// Let RequireFirm handle missing firm
				return next(c)
			}

			subscription, err := services.GetFirmSubscription(db.DB, firm.ID)
			if err != nil {
				// No subscription found - redirect to settings
				if c.Request().Header.Get("HX-Request") == "true" {
					c.Response().Header().Set("HX-Redirect", "/firm/settings#subscription")
					return c.NoContent(http.StatusForbidden)
				}
				return c.Redirect(http.StatusSeeOther, "/firm/settings#subscription")
			}

			// Check if subscription is active
			if !subscription.IsActive() {
				if c.Request().Header.Get("HX-Request") == "true" {
					c.Response().Header().Set("HX-Redirect", "/firm/settings#subscription")
					return c.NoContent(http.StatusForbidden)
				}
				return c.Redirect(http.StatusSeeOther, "/firm/settings#subscription")
			}

			// Check trial expiration
			if subscription.IsTrialing() && subscription.TrialEndsAt != nil {
				if time.Now().After(*subscription.TrialEndsAt) {
					// Update status to expired
					subscription.Status = models.SubscriptionStatusExpired
					db.DB.Save(subscription)

					if c.Request().Header.Get("HX-Request") == "true" {
						c.Response().Header().Set("HX-Redirect", "/firm/settings#subscription")
						return c.NoContent(http.StatusForbidden)
					}
					return c.Redirect(http.StatusSeeOther, "/firm/settings#subscription")
				}
			}

			// Store subscription in context for use in handlers
			c.Set(ContextKeySubscription, subscription)

			return next(c)
		}
	}
}

// LoadSubscriptionInfo loads subscription info for display (non-blocking)
// This middleware does NOT block requests - it just loads subscription data for templates
func LoadSubscriptionInfo() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			firm := GetCurrentFirm(c)
			if firm == nil {
				return next(c)
			}

			// Load subscription info in background-safe way
			info, err := services.GetFirmSubscriptionInfo(db.DB, firm.ID)
			if err == nil {
				c.Set(ContextKeySubscriptionInfo, info)
			}

			return next(c)
		}
	}
}

// GetSubscription retrieves subscription from context
func GetSubscription(c echo.Context) *models.FirmSubscription {
	sub, ok := c.Get(ContextKeySubscription).(*models.FirmSubscription)
	if !ok {
		return nil
	}
	return sub
}

// GetSubscriptionInfo retrieves subscription info from context
func GetSubscriptionInfo(c echo.Context) *services.SubscriptionInfo {
	info, ok := c.Get(ContextKeySubscriptionInfo).(*services.SubscriptionInfo)
	if !ok {
		return nil
	}
	return info
}

// RequireTemplatesAccess ensures the firm has access to templates feature
func RequireTemplatesAccess() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			firm := GetCurrentFirm(c)
			if firm == nil {
				return next(c)
			}

			canAccess, err := services.CanAccessTemplates(db.DB, firm.ID)
			if err != nil || !canAccess {
				if c.Request().Header.Get("HX-Request") == "true" {
					c.Response().Header().Set("HX-Redirect", "/firm/settings?upgrade=templates#subscription")
					return c.NoContent(http.StatusForbidden)
				}
				return c.Redirect(http.StatusSeeOther, "/firm/settings?upgrade=templates#subscription")
			}

			return next(c)
		}
	}
}
