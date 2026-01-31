package services

import (
	"context"
	"fmt"
	"law_flow_app_go/models"
	"law_flow_app_go/services/i18n"
	"log"
	"time"

	"gorm.io/gorm"
)

// CurrentPrivacyPolicyVersion is the current version of the privacy policy.
// Update this when the policy changes.
const CurrentPrivacyPolicyVersion = "1.0.0"

// GetLocalizedPolicyText returns the privacy policy text in the user's language.
// This function retrieves i18n keys for the policy sections.
func GetLocalizedPolicyText(ctx context.Context) string {
	return i18n.T(ctx, "compliance.consent.policy_text")
}

// LogConsent records a user's consent to data processing in an immutable log.
func LogConsent(ctx context.Context, db *gorm.DB, userID, userEmail string, firmID *string, consentType models.ConsentType, granted bool, ipAddress, userAgent string) error {
	consent := models.ConsentLog{
		UserID:        userID,
		UserEmail:     userEmail,
		FirmID:        firmID,
		ConsentType:   consentType,
		Granted:       granted,
		PolicyVersion: CurrentPrivacyPolicyVersion,
		PolicyText:    GetLocalizedPolicyText(ctx),
		IPAddress:     ipAddress,
		UserAgent:     userAgent,
	}

	if err := db.Create(&consent).Error; err != nil {
		return fmt.Errorf("failed to log consent: %w", err)
	}

	return nil
}

// GetUserConsents retrieves all consent records for a user.
func GetUserConsents(db *gorm.DB, userID string) ([]models.ConsentLog, error) {
	var consents []models.ConsentLog
	if err := db.Where("user_id = ?", userID).Order("created_at DESC").Find(&consents).Error; err != nil {
		return nil, fmt.Errorf("failed to get user consents: %w", err)
	}
	return consents, nil
}

// GetLatestConsent retrieves the most recent consent of a specific type for a user.
func GetLatestConsent(db *gorm.DB, userID string, consentType models.ConsentType) (*models.ConsentLog, error) {
	var consent models.ConsentLog
	err := db.Where("user_id = ? AND consent_type = ?", userID, consentType).
		Order("created_at DESC").
		First(&consent).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No consent found
		}
		return nil, fmt.Errorf("failed to get latest consent: %w", err)
	}
	return &consent, nil
}

// HasValidConsent checks if a user has a valid (granted, not revoked) consent for a specific type.
func HasValidConsent(db *gorm.DB, userID string, consentType models.ConsentType) (bool, error) {
	consent, err := GetLatestConsent(db, userID, consentType)
	if err != nil {
		return false, err
	}
	if consent == nil {
		return false, nil
	}
	return consent.Granted, nil
}

// CreateSubjectRightsRequest creates a new ARCO rights request.
func CreateSubjectRightsRequest(db *gorm.DB, userID, userEmail string, firmID *string, requestType models.SubjectRequestType, justification, ipAddress, userAgent string) (*models.SubjectRightsRequest, error) {
	request := models.SubjectRightsRequest{
		UserID:        userID,
		UserEmail:     userEmail,
		FirmID:        firmID,
		RequestType:   requestType,
		Status:        models.SubjectRequestStatusPending,
		Justification: justification,
		IPAddress:     ipAddress,
		UserAgent:     userAgent,
	}

	if err := db.Create(&request).Error; err != nil {
		return nil, fmt.Errorf("failed to create subject rights request: %w", err)
	}

	return &request, nil
}

// GetSubjectRightsRequests retrieves all ARCO requests for a firm.
func GetSubjectRightsRequests(db *gorm.DB, firmID string, status *models.SubjectRequestStatus, page, limit int) ([]models.SubjectRightsRequest, int64, error) {
	var requests []models.SubjectRightsRequest
	var total int64

	query := db.Model(&models.SubjectRightsRequest{}).Where("firm_id = ?", firmID)
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count requests: %w", err)
	}

	if err := query.Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&requests).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get requests: %w", err)
	}

	return requests, total, nil
}

// ResolveSubjectRightsRequest resolves an ARCO request (approve/deny).
func ResolveSubjectRightsRequest(db *gorm.DB, requestID string, resolverID string, status models.SubjectRequestStatus, response string) error {
	now := time.Now()
	result := db.Model(&models.SubjectRightsRequest{}).
		Where("id = ?", requestID).
		Updates(map[string]interface{}{
			"status":         status,
			"response":       response,
			"resolved_at":    now,
			"resolved_by_id": resolverID,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to resolve request: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("request not found")
	}
	return nil
}

// MassiveDownloadThreshold is the number of downloads in a time window that triggers a breach alert.
const MassiveDownloadThreshold = 20
const MassiveDownloadWindow = 1 * time.Minute

// TrackDocumentDownload tracks a document download and triggers alerts if threshold is exceeded.
func (m *SecurityEventMonitor) TrackDocumentDownload(ipAddress, userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Use a combined key for tracking
	key := fmt.Sprintf("dl:%s:%s", ipAddress, userID)
	now := time.Now()

	if _, exists := m.failedLogins[key]; !exists {
		m.failedLogins[key] = []time.Time{}
	}

	m.failedLogins[key] = append(m.failedLogins[key], now)

	// Filter to window
	windowStart := now.Add(-MassiveDownloadWindow)
	validDownloads := []time.Time{}
	for _, t := range m.failedLogins[key] {
		if t.After(windowStart) {
			validDownloads = append(validDownloads, t)
		}
	}
	m.failedLogins[key] = validDownloads

	if len(validDownloads) >= MassiveDownloadThreshold {
		m.triggerBreachAlert(ipAddress, userID, "Massive document download detected")
	}
}

// triggerBreachAlert triggers a potential data breach alert
func (m *SecurityEventMonitor) triggerBreachAlert(ip, userID, reason string) {
	// Rate limit alerts
	alertKey := fmt.Sprintf("breach:%s", ip)
	lastAlert, alerted := m.alertedIPs[alertKey]
	if alerted && time.Since(lastAlert) < 1*time.Hour {
		return
	}
	m.alertedIPs[alertKey] = time.Now()

	alert := SecurityAlert{
		Timestamp: time.Now(),
		IP:        ip,
		Reason:    fmt.Sprintf("%s (User: %s)", reason, userID),
		Level:     "CRITICAL",
	}

	m.alerts = append([]SecurityAlert{alert}, m.alerts...)
	if len(m.alerts) > 100 {
		m.alerts = m.alerts[:100]
	}

	log.Printf("[SECURITY BREACH ALERT] %s from IP: %s, User: %s", reason, ip, userID)
	// TODO: Send email notification to Compliance Officer
}
