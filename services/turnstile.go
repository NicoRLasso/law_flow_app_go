package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type TurnstileResponse struct {
	Success     bool      `json:"success"`
	ChallengeTS time.Time `json:"challenge_ts"`
	Hostname    string    `json:"hostname"`
	ErrorCodes  []string  `json:"error-codes"`
}

// VerifyTurnstileToken verifies the token with Cloudflare
func VerifyTurnstileToken(token, secretKey, ip string) (bool, error) {
	if token == "" || secretKey == "" {
		return false, fmt.Errorf("missing token or secret key")
	}

	resp, err := http.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify", url.Values{
		"secret":   {secretKey},
		"response": {token},
		"remoteip": {ip},
	})
	if err != nil {
		return false, fmt.Errorf("failed to verify token: %w", err)
	}
	defer resp.Body.Close()

	var result TurnstileResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode turnstile response: %w", err)
	}

	// If success is false, return an error with the error codes
	if !result.Success {
		return false, fmt.Errorf("turnstile verification failed, error codes: %v", result.ErrorCodes)
	}

	return true, nil
}
