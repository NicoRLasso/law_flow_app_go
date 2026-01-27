package middleware

import (
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestAuditContext(t *testing.T) {
	e := echo.New()

	t.Run("FullContext", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("User-Agent", "test-agent")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Set user and firm in context
		user := &models.User{ID: "user-123", Name: "Test User", Role: "admin"}
		firm := &models.Firm{ID: "firm-456", Name: "Test Firm"}
		c.Set(ContextKeyUser, user)
		c.Set(ContextKeyFirm, firm)

		handler := AuditContext()(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		err := handler(c)
		assert.NoError(t, err)

		auditCtx := GetAuditContext(c)
		assert.Equal(t, "user-123", auditCtx.UserID)
		assert.Equal(t, "Test User", auditCtx.UserName)
		assert.Equal(t, "admin", auditCtx.UserRole)
		assert.Equal(t, "firm-456", auditCtx.FirmID)
		assert.Equal(t, "Test Firm", auditCtx.FirmName)
		assert.Equal(t, "test-agent", auditCtx.UserAgent)
	})

	t.Run("NoAuth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := AuditContext()(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		err := handler(c)
		assert.NoError(t, err)

		auditCtx := GetAuditContext(c)
		assert.Empty(t, auditCtx.UserID)
		assert.Empty(t, auditCtx.FirmID)
	})
}

func TestGetAuditContext(t *testing.T) {
	e := echo.New()

	t.Run("Exists", func(t *testing.T) {
		c := e.NewContext(nil, nil)
		expected := services.AuditContext{UserID: "123"}
		c.Set(ContextKeyAuditContext, expected)

		result := GetAuditContext(c)
		assert.Equal(t, expected, result)
	})

	t.Run("NotExists", func(t *testing.T) {
		c := e.NewContext(nil, nil)
		result := GetAuditContext(c)
		assert.Equal(t, services.AuditContext{}, result)
	})
}
