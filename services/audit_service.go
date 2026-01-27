package services

import (
	"encoding/json"
	"law_flow_app_go/models"
	"log"
	"time"

	"gorm.io/gorm"
)

// AuditContext contains contextual information for audit logging
type AuditContext struct {
	UserID    string
	UserName  string
	UserRole  string
	FirmID    string
	FirmName  string
	IPAddress string
	UserAgent string
}

// LogAuditEvent creates a new audit log entry asynchronously
func LogAuditEvent(
	db *gorm.DB,
	ctx AuditContext,
	action models.AuditAction,
	resourceType string,
	resourceID string,
	resourceName string,
	description string,
	oldValues interface{},
	newValues interface{},
) {
	// Run in goroutine to avoid blocking the request
	go func() {
		var oldJSON, newJSON string

		if oldValues != nil {
			if bytes, err := json.Marshal(oldValues); err == nil {
				oldJSON = string(bytes)
			}
		}

		if newValues != nil {
			if bytes, err := json.Marshal(newValues); err == nil {
				newJSON = string(bytes)
			}
		}

		auditLog := models.AuditLog{
			UserID:       ptrIfNotEmpty(ctx.UserID),
			UserName:     ctx.UserName,
			UserRole:     ctx.UserRole,
			FirmID:       ptrIfNotEmpty(ctx.FirmID),
			FirmName:     ctx.FirmName,
			ResourceType: resourceType,
			ResourceID:   resourceID,
			ResourceName: resourceName,
			Action:       action,
			Description:  description,
			OldValues:    oldJSON,
			NewValues:    newJSON,
			IPAddress:    ctx.IPAddress,
			UserAgent:    ctx.UserAgent,
		}

		if err := db.Create(&auditLog).Error; err != nil {
			log.Printf("[AUDIT] Failed to create audit log: %v", err)
		}
	}()
}

// ptrIfNotEmpty returns a pointer to the string if not empty, nil otherwise
func ptrIfNotEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// GetResourceAuditHistory retrieves the audit history for a specific resource
func GetResourceAuditHistory(db *gorm.DB, resourceType, resourceID string) ([]models.AuditLog, error) {
	var logs []models.AuditLog
	err := db.Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Order("created_at DESC").
		Find(&logs).Error
	return logs, err
}

// GetFirmAuditLogs retrieves paginated audit logs for a firm
func GetFirmAuditLogs(
	db *gorm.DB,
	firmID string,
	filters AuditLogFilters,
	page, pageSize int,
) ([]models.AuditLog, int64, error) {
	query := db.Model(&models.AuditLog{}).Where("firm_id = ?", firmID)

	// Apply filters
	if filters.UserID != "" {
		query = query.Where("user_id = ?", filters.UserID)
	}
	if filters.ResourceType != "" {
		query = query.Where("resource_type = ?", filters.ResourceType)
	}
	if filters.Action != "" {
		query = query.Where("action = ?", filters.Action)
	}
	if !filters.DateFrom.IsZero() {
		query = query.Where("created_at >= ?", filters.DateFrom)
	}
	if !filters.DateTo.IsZero() {
		query = query.Where("created_at <= ?", filters.DateTo)
	}
	if filters.SearchQuery != "" {
		searchPattern := "%" + filters.SearchQuery + "%"
		query = query.Where(
			"resource_name LIKE ? OR description LIKE ? OR user_name LIKE ?",
			searchPattern, searchPattern, searchPattern,
		)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	var logs []models.AuditLog
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&logs).Error

	return logs, total, err
}

// AuditLogFilters contains filter options for audit log queries
type AuditLogFilters struct {
	UserID       string
	ResourceType string
	Action       string
	DateFrom     time.Time
	DateTo       time.Time
	SearchQuery  string
}

// LogSecurityEvent logs security-related events to the database and standard log
func LogSecurityEvent(db *gorm.DB, eventType, userID, details string) {
	// Log to stdout for immediate visibility (e.g. into aggregation limits)
	log.Printf("[SECURITY] %s | User: %s | Details: %s", eventType, userID, details)

	// Persist to database asynchronously
	go func() {
		auditLog := models.AuditLog{
			UserID:       ptrIfNotEmpty(userID),
			Action:       models.AuditAction("SECURITY"), // Cast string to AuditAction
			ResourceType: "SECURITY_EVENT",
			ResourceID:   eventType, // Use event type as resource ID or similar
			Description:  details,
			NewValues:    eventType, // Store event type in NewValues or similar
		}

		// If headers/IP are important for security logs, they should be passed.
		// For now, keeping signature simple as requested, but we can enhance later.

		if err := db.Create(&auditLog).Error; err != nil {
			log.Printf("[AUDIT] Failed to create security audit log: %v", err)
		}
	}()
}
