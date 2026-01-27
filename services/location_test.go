package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDefaultTimezone(t *testing.T) {
	tests := []struct {
		name     string
		country  string
		expected string
	}{
		{
			name:     "Colombia uppercase",
			country:  "COLOMBIA",
			expected: "America/Bogota",
		},
		{
			name:     "Colombia lowercase",
			country:  "colombia",
			expected: "America/Bogota",
		},
		{
			name:     "Colombia mixed case",
			country:  "CoLoMbIa",
			expected: "America/Bogota",
		},
		{
			name:     "Other country",
			country:  "USA",
			expected: "",
		},
		{
			name:     "Empty country",
			country:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetDefaultTimezone(tt.country)
			assert.Equal(t, tt.expected, got)
		})
	}
}
