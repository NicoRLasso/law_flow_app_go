package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerifyTurnstileToken(t *testing.T) {
	// Backup and restore URL
	oldURL := turnstileVerifyURL
	defer func() { turnstileVerifyURL = oldURL }()

	t.Run("Missing inputs", func(t *testing.T) {
		success, err := VerifyTurnstileToken("", "secret", "127.0.0.1")
		assert.False(t, success)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing token")
	})

	t.Run("Verification success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(TurnstileResponse{
				Success: true,
			})
		}))
		defer server.Close()

		turnstileVerifyURL = server.URL
		success, err := VerifyTurnstileToken("valid-token", "secret", "1.1.1.1")
		assert.NoError(t, err)
		assert.True(t, success)
	})

	t.Run("Verification failure with error codes", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(TurnstileResponse{
				Success:    false,
				ErrorCodes: []string{"invalid-input-response", "timeout-or-duplicate"},
			})
		}))
		defer server.Close()

		turnstileVerifyURL = server.URL
		success, err := VerifyTurnstileToken("invalid-token", "secret", "1.1.1.1")
		assert.False(t, success)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid-input-response")
	})

	t.Run("Malformed JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("{ malformed json }"))
		}))
		defer server.Close()

		turnstileVerifyURL = server.URL
		success, err := VerifyTurnstileToken("token", "secret", "1.1.1.1")
		assert.False(t, success)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode")
	})
}
