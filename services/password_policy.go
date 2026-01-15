package services

import (
	"fmt"
	"unicode"
)

// Password requirements
const (
	MinPasswordLength = 12
)

// ValidatePassword checks if the password meets the complexity requirements
// - At least 12 characters
// - At least one uppercase letter
// - At least one lowercase letter
// - At least one number
// - At least one special character
func ValidatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return fmt.Errorf("password must be at least %d characters long", MinPasswordLength)
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasNumber  bool
		hasSpecial bool
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !hasNumber {
		return fmt.Errorf("password must contain at least one number")
	}
	if !hasSpecial {
		return fmt.Errorf("password must contain at least one special character")
	}

	return nil
}

// IsWeakPassword is a helper to check if a password is weak without returning specific error
func IsWeakPassword(password string) bool {
	return ValidatePassword(password) != nil
}
