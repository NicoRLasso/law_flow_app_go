package services

import "strings"

// GetDefaultTimezone returns the default timezone for a given country.
// Returns an empty string if no default is enforced.
func GetDefaultTimezone(country string) string {
	if strings.EqualFold(country, "Colombia") {
		return "America/Bogota"
	}
	return ""
}
