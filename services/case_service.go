package services

import (
	"fmt"
	"law_flow_app_go/models"
	"time"

	"gorm.io/gorm"
)

// GenerateCaseNumber generates a unique case number for a firm
// Format: {FIRM_SLUG}-{YEAR}-{SEQUENCE}
// Example: LAW-2026-00042
func GenerateCaseNumber(db *gorm.DB, firmID string) (string, error) {
	// Fetch firm to get slug
	var firm models.Firm
	if err := db.First(&firm, "id = ?", firmID).Error; err != nil {
		return "", fmt.Errorf("failed to fetch firm: %w", err)
	}

	// Get current year
	currentYear := time.Now().Year()

	// Find the highest sequence number for this firm and year
	var maxCase models.Case
	err := db.Where("firm_id = ? AND case_number LIKE ?", firmID, fmt.Sprintf("%s-%d-%%", firm.Slug, currentYear)).
		Order("case_number DESC").
		First(&maxCase).Error

	sequence := 1
	if err == nil {
		// Parse sequence from existing case number
		var parsedSeq int
		_, scanErr := fmt.Sscanf(maxCase.CaseNumber, fmt.Sprintf("%s-%d-%%d", firm.Slug, currentYear), &parsedSeq)
		if scanErr == nil {
			sequence = parsedSeq + 1
		}
	} else if err != gorm.ErrRecordNotFound {
		return "", fmt.Errorf("failed to query max case number: %w", err)
	}

	// Format case number with zero-padded sequence
	caseNumber := fmt.Sprintf("%s-%d-%05d", firm.Slug, currentYear, sequence)
	return caseNumber, nil
}

// EnsureUniqueCaseNumber generates a unique case number with retry logic
// Retries up to maxRetries times if a collision occurs
func EnsureUniqueCaseNumber(db *gorm.DB, firmID string) (string, error) {
	const maxRetries = 10

	for i := 0; i < maxRetries; i++ {
		caseNumber, err := GenerateCaseNumber(db, firmID)
		if err != nil {
			return "", err
		}

		// Check if case number already exists
		var count int64
		if err := db.Model(&models.Case{}).Where("case_number = ?", caseNumber).Count(&count).Error; err != nil {
			return "", fmt.Errorf("failed to check case number uniqueness: %w", err)
		}

		if count == 0 {
			return caseNumber, nil
		}

		// Collision detected, retry
	}

	return "", fmt.Errorf("failed to generate unique case number after %d retries", maxRetries)
}
