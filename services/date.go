package services

import (
	"fmt"
	"time"
)

// ParseDate parses a date string in typical formats (YYYY-MM-DD)
// It enforces strict checks but centralizes the logic for future format additions
func ParseDate(dateStr string) (time.Time, error) {
	// Primary format: ISO 8601 (standard for HTML5 date inputs)
	layout := "2006-01-02"

	parsedTime, err := time.Parse(layout, dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format: expected YYYY-MM-DD")
	}

	return parsedTime, nil
}
