package middleware

import (
	"law_flow_app_go/config"
	"law_flow_app_go/services/i18n"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestLocale(t *testing.T) {
	e := echo.New()
	cfg := &config.Config{Environment: "development"}

	t.Run("PriorityQueryParam", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?lang=es", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := Locale(cfg)(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		err := handler(c)
		assert.NoError(t, err)
		assert.Equal(t, "es", c.Get("locale"))
		// Check cookie
		cookies := rec.Result().Cookies()
		found := false
		for _, cookie := range cookies {
			if cookie.Name == "lang" {
				assert.Equal(t, "es", cookie.Value)
				found = true
			}
		}
		assert.True(t, found)
	})

	t.Run("PriorityCookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "lang", Value: "es"})
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := Locale(cfg)(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		err := handler(c)
		assert.NoError(t, err)
		assert.Equal(t, "es", c.Get("locale"))
	})

	t.Run("PriorityHeader", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Accept-Language", "es-ES,es;q=0.9")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := Locale(cfg)(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		err := handler(c)
		assert.NoError(t, err)
		assert.Equal(t, "es", c.Get("locale"))
	})

	t.Run("DefaultLanguage", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := Locale(cfg)(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		err := handler(c)
		assert.NoError(t, err)
		assert.Equal(t, "en", c.Get("locale"))
	})

	t.Run("RequestContext", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?lang=es", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := Locale(cfg)(func(c echo.Context) error {
			lang := c.Request().Context().Value(i18n.LocaleContextKey)
			assert.Equal(t, "es", lang)
			return c.NoContent(http.StatusOK)
		})

		err := handler(c)
		assert.NoError(t, err)
	})
}

func TestSetLanguageCookie(t *testing.T) {
	e := echo.New()
	t.Run("Development", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("config", &config.Config{Environment: "development"})

		SetLanguageCookie(c, "es")

		cookies := rec.Result().Cookies()
		var langCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == "lang" {
				langCookie = cookie
			}
		}
		assert.NotNil(t, langCookie)
		assert.Equal(t, "es", langCookie.Value)
		assert.False(t, langCookie.Secure)
	})

	t.Run("Production", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("config", &config.Config{Environment: "production"})

		SetLanguageCookie(c, "en")

		cookies := rec.Result().Cookies()
		var langCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == "lang" {
				langCookie = cookie
			}
		}
		assert.NotNil(t, langCookie)
		assert.True(t, langCookie.Secure)
	})
}

func TestGetLocale(t *testing.T) {
	e := echo.New()
	t.Run("WithLocale", func(t *testing.T) {
		c := e.NewContext(nil, nil)
		c.Set("locale", "es")
		assert.Equal(t, "es", GetLocale(c))
	})

	t.Run("WithoutLocale", func(t *testing.T) {
		c := e.NewContext(nil, nil)
		assert.Equal(t, "en", GetLocale(c))
	})
}
