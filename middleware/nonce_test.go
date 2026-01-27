package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestGenerateNonce(t *testing.T) {
	nonce1, err := GenerateNonce()
	assert.NoError(t, err)
	assert.NotEmpty(t, nonce1)

	nonce2, err := GenerateNonce()
	assert.NoError(t, err)
	assert.NotEqual(t, nonce1, nonce2)
}

func TestCSPNonce(t *testing.T) {
	e := echo.New()

	t.Run("SetsContextAndHeader", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := CSPNonce()(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		err := handler(c)
		assert.NoError(t, err)

		// Check Echo context
		nonce := c.Get(string(NonceKey)).(string)
		assert.NotEmpty(t, nonce)

		// Check Request context
		ctxNonce := c.Request().Context().Value(NonceKey).(string)
		assert.Equal(t, nonce, ctxNonce)

		// Check CSP Header
		csp := rec.Header().Get("Content-Security-Policy")
		assert.Contains(t, csp, "nonce-"+nonce)
		assert.Contains(t, csp, "script-src")
	})
}

func TestGetNonce(t *testing.T) {
	t.Run("Exists", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), NonceKey, "test-nonce")
		assert.Equal(t, "test-nonce", GetNonce(ctx))
	})

	t.Run("NotExists", func(t *testing.T) {
		assert.Equal(t, "", GetNonce(context.Background()))
	})
}
