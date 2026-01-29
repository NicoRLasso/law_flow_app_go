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

// GetDefaultCurrency returns the default currency for a given country.
// Returns "USD" as fallback if no specific default is defined.
func GetDefaultCurrency(country string) string {
	if strings.EqualFold(country, "Colombia") {
		return "COP"
	}
	return "USD"
}
