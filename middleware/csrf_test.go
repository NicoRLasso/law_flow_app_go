package middleware

import (
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestGetCSRFToken(t *testing.T) {
	e := echo.New()

	t.Run("TokenExists", func(t *testing.T) {
		c := e.NewContext(nil, nil)
		expectedToken := "test-csrf-token"
		c.Set("csrf", expectedToken)

		token := GetCSRFToken(c)
		assert.Equal(t, expectedToken, token)
	})

	t.Run("TokenMissing", func(t *testing.T) {
		c := e.NewContext(nil, nil)

		token := GetCSRFToken(c)
		assert.Equal(t, "", token)
	})

	t.Run("TokenInvalidType", func(t *testing.T) {
		c := e.NewContext(nil, nil)
		c.Set("csrf", 123) // Not a string

		token := GetCSRFToken(c)
		assert.Equal(t, "", token)
	})
}
