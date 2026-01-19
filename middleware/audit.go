package middleware

import (
	"law_flow_app_go/services"

	"github.com/labstack/echo/v4"
)

const ContextKeyAuditContext = "audit_context"

// AuditContext is middleware that extracts user info for audit logging
func AuditContext() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := GetCurrentUser(c)
			firm := GetCurrentFirm(c)

			ctx := services.AuditContext{
				IPAddress: c.RealIP(),
				UserAgent: c.Request().UserAgent(),
			}

			if user != nil {
				ctx.UserID = user.ID
				ctx.UserName = user.Name
				ctx.UserRole = user.Role
			}

			if firm != nil {
				ctx.FirmID = firm.ID
				ctx.FirmName = firm.Name
			}

			c.Set(ContextKeyAuditContext, ctx)
			return next(c)
		}
	}
}

// GetAuditContext retrieves the audit context from the request
func GetAuditContext(c echo.Context) services.AuditContext {
	if ctx, ok := c.Get(ContextKeyAuditContext).(services.AuditContext); ok {
		return ctx
	}
	return services.AuditContext{}
}
