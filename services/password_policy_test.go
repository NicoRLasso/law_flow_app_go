package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "Valid complex password",
			password: "StrongPassword123!",
			wantErr:  false,
		},
		{
			name:     "Too short",
			password: "Short1!",
			wantErr:  true,
			errMsg:   "password must be at least 12 characters long",
		},
		{
			name:     "Missing uppercase",
			password: "lowercase123!",
			wantErr:  true,
			errMsg:   "password must contain at least one uppercase letter",
		},
		{
			name:     "Missing lowercase",
			password: "UPPERCASE123!",
			wantErr:  true,
			errMsg:   "password must contain at least one lowercase letter",
		},
		{
			name:     "Missing number",
			password: "NoNumberPass!",
			wantErr:  true,
			errMsg:   "password must contain at least one number",
		},
		{
			name:     "Missing special char",
			password: "NoSpecialChar123",
			wantErr:  true,
			errMsg:   "password must contain at least one special character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsWeakPassword(t *testing.T) {
	assert.True(t, IsWeakPassword("weak"))
	assert.False(t, IsWeakPassword("StrongPassword123!"))
}
