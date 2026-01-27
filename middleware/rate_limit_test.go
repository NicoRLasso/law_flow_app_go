package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{
		Requests: 10,
		Window:   time.Minute,
	})

	assert.NotNil(t, rl)
	assert.Equal(t, 10, rl.config.Requests)
	assert.Equal(t, time.Minute, rl.config.Window)
	assert.NotNil(t, rl.config.KeyFunc)
	assert.Equal(t, "Too many requests. Please try again later.", rl.config.Message)
}

func TestRateLimiterMiddleware(t *testing.T) {
	e := echo.New()

	t.Run("WithinLimit", func(t *testing.T) {
		rl := NewRateLimiter(RateLimitConfig{
			Requests: 2,
			Window:   time.Second,
		})

		handler := rl.Middleware()(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		// First request
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		assert.NoError(t, handler(c))
		assert.Equal(t, http.StatusOK, rec.Code)

		// Second request
		req = httptest.NewRequest(http.MethodGet, "/", nil)
		rec = httptest.NewRecorder()
		c = e.NewContext(req, rec)
		assert.NoError(t, handler(c))
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("ExceededLimit", func(t *testing.T) {
		rl := NewRateLimiter(RateLimitConfig{
			Requests: 1,
			Window:   time.Second,
		})

		handler := rl.Middleware()(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		// First request (OK)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		assert.NoError(t, handler(c))

		// Second request (Rate Limited)
		req = httptest.NewRequest(http.MethodGet, "/", nil)
		rec = httptest.NewRecorder()
		c = e.NewContext(req, rec)
		err := handler(c)

		assert.Error(t, err)
		he, ok := err.(*echo.HTTPError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusTooManyRequests, he.Code)
	})

	t.Run("HXRequestExceeded", func(t *testing.T) {
		rl := NewRateLimiter(RateLimitConfig{
			Requests: 1,
			Window:   time.Second,
		})

		handler := rl.Middleware()(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		// First request (OK)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		assert.NoError(t, handler(c))

		// Second request (Rate Limited)
		req = httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("HX-Request", "true")
		rec = httptest.NewRecorder()
		c = e.NewContext(req, rec)

		err := handler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
		assert.Contains(t, rec.Body.String(), "Too many requests")
	})
}
