package services

import (
	"errors"
	"fmt"
	"law_flow_app_go/models"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Service-related errors
var (
	ErrServiceNotFound     = errors.New("service not found")
	ErrServiceLimitReached = errors.New("service limit reached for this plan")
)

// ServiceFilters holds filter options for querying services
type ServiceFilters struct {
	Status        string
	ServiceTypeID string
	ClientID      string
	AssignedToID  string
	Priority      string
	Keyword       string
	DateFrom      *time.Time
	DateTo        *time.Time
}

// GenerateServiceNumber generates a unique service number for a firm
// Format: {FIRM_SLUG}-SVC-{YEAR}-{SEQUENCE}
// Example: LAW-SVC-2026-00042
func GenerateServiceNumber(db *gorm.DB, firmID string) (string, error) {
	// Fetch firm to get slug
	var firm models.Firm
	if err := db.First(&firm, "id = ?", firmID).Error; err != nil {
		return "", fmt.Errorf("failed to fetch firm: %w", err)
	}

	currentYear := time.Now().Year()

	// Find the highest sequence number for this firm and year
	var maxService models.LegalService
	prefix := fmt.Sprintf("%s-SVC-%d-", firm.Slug, currentYear)
	err := db.Where("firm_id = ? AND service_number LIKE ?", firmID, prefix+"%").
		Order("service_number DESC").
		First(&maxService).Error

	sequence := 1
	if err == nil {
		// Parse sequence from existing service number
		var parsedSeq int
		// Scan format: {SLUG}-SVC-{YEAR}-{SEQ}
		// Since SLUG can contain dashes, we should be careful, but fmt.Sscanf might work if we know the structure.
		// A safer way is to find the last dash.
		parts := strings.Split(maxService.ServiceNumber, "-")
		if len(parts) >= 4 {
			seqStr := parts[len(parts)-1]
			fmt.Sscanf(seqStr, "%d", &parsedSeq)
			sequence = parsedSeq + 1
		}
	} else if err != gorm.ErrRecordNotFound {
		return "", fmt.Errorf("failed to query max service number: %w", err)
	}

	// Format service number with zero-padded sequence
	serviceNumber := fmt.Sprintf("%s-SVC-%d-%05d", firm.Slug, currentYear, sequence)
	return serviceNumber, nil
}

// EnsureUniqueServiceNumber generates a unique service number with retry logic
func EnsureUniqueServiceNumber(db *gorm.DB, firmID string) (string, error) {
	const maxRetries = 10

	for i := 0; i < maxRetries; i++ {
		serviceNumber, err := GenerateServiceNumber(db, firmID)
		if err != nil {
			return "", err
		}

		// Check if service number already exists
		var count int64
		if err := db.Model(&models.LegalService{}).Where("service_number = ?", serviceNumber).Count(&count).Error; err != nil {
			return "", fmt.Errorf("failed to check service number uniqueness: %w", err)
		}

		if count == 0 {
			return serviceNumber, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique service number after %d retries", maxRetries)
}

// CanAddService checks if a firm can add more services based on subscription limits
// Services share the same limit as cases
func CanAddService(db *gorm.DB, firmID string) (*LimitCheckResult, error) {
	return CanAddCase(db, firmID)
}

// GetServiceByID retrieves a service by ID with all relationships preloaded
func GetServiceByID(db *gorm.DB, firmID, serviceID string) (*models.LegalService, error) {
	var service models.LegalService
	err := db.Where("firm_id = ? AND id = ?", firmID, serviceID).
		Preload("Firm").
		Preload("Client").
		Preload("Client.DocumentType").
		Preload("ServiceType").
		Preload("AssignedTo").
		Preload("Milestones", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC")
		}).
		Preload("Documents").
		Preload("Expenses").
		Preload("Expenses.Category").
		Preload("Collaborators").
		First(&service).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrServiceNotFound
		}
		return nil, err
	}

	return &service, nil
}

// GetServicesByFirm retrieves services for a firm with filters and pagination
func GetServicesByFirm(db *gorm.DB, firmID string, filters ServiceFilters, page, limit int) ([]models.LegalService, int64, error) {
	var services []models.LegalService
	var total int64

	query := db.Where("firm_id = ?", firmID)

	// Apply filters
	if filters.Status != "" && models.IsValidServiceStatus(filters.Status) {
		query = query.Where("status = ?", filters.Status)
	}
	if filters.ServiceTypeID != "" {
		query = query.Where("service_type_id = ?", filters.ServiceTypeID)
	}
	if filters.ClientID != "" {
		query = query.Where("client_id = ?", filters.ClientID)
	}
	if filters.AssignedToID != "" {
		query = query.Where("assigned_to_id = ?", filters.AssignedToID)
	}
	if filters.Priority != "" && models.IsValidServicePriority(filters.Priority) {
		query = query.Where("priority = ?", filters.Priority)
	}
	if filters.Keyword != "" {
		kw := "%" + filters.Keyword + "%"
		query = query.Where(
			db.Where("service_number LIKE ?", kw).
				Or("title LIKE ?", kw).
				Or("description LIKE ?", kw).
				Or("objective LIKE ?", kw),
		)
	}
	if filters.DateFrom != nil {
		query = query.Where("created_at >= ?", filters.DateFrom)
	}
	if filters.DateTo != nil {
		query = query.Where("created_at <= ?", filters.DateTo)
	}

	// Count total
	if err := query.Model(&models.LegalService{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	offset := (page - 1) * limit
	err := query.
		Preload("Client").
		Preload("ServiceType").
		Preload("AssignedTo").
		Preload("Milestones").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&services).Error

	return services, total, err
}

// GetServiceTotalExpenses calculates total expenses for a service (excluding rejected)
func GetServiceTotalExpenses(db *gorm.DB, serviceID string) (float64, error) {
	var total float64
	err := db.Model(&models.ServiceExpense{}).
		Where("service_id = ? AND status != ?", serviceID, models.ExpenseStatusRejected).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error
	return total, err
}

// GetServiceApprovedExpenses calculates approved expenses for a service
func GetServiceApprovedExpenses(db *gorm.DB, serviceID string) (float64, error) {
	var total float64
	err := db.Model(&models.ServiceExpense{}).
		Where("service_id = ? AND status IN ?", serviceID, []string{models.ExpenseStatusApproved, models.ExpenseStatusPaid}).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error
	return total, err
}

// UpdateServiceStatus updates the status of a service with tracking
func UpdateServiceStatus(db *gorm.DB, serviceID, newStatus, userID string) error {
	if !models.IsValidServiceStatus(newStatus) {
		return fmt.Errorf("invalid service status: %s", newStatus)
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":            newStatus,
		"status_changed_at": now,
		"status_changed_by": userID,
	}

	// Set started_at when moving to IN_PROGRESS
	if newStatus == models.ServiceStatusInProgress {
		var service models.LegalService
		if err := db.Select("started_at").First(&service, "id = ?", serviceID).Error; err == nil {
			if service.StartedAt == nil {
				updates["started_at"] = now
			}
		}
	}

	// Set completed_at when moving to COMPLETED
	if newStatus == models.ServiceStatusCompleted {
		updates["completed_at"] = now
	}

	return db.Model(&models.LegalService{}).
		Where("id = ?", serviceID).
		Updates(updates).Error
}

// DeleteService deletes a service and all its related entities
func DeleteService(db *gorm.DB, firmID, serviceID string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// Verify service exists and belongs to the firm
		var service models.LegalService
		if err := tx.Where("firm_id = ? AND id = ?", firmID, serviceID).First(&service).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrServiceNotFound
			}
			return err
		}

		// Delete related entities first (if not handled by cascade)
		// Usually GORM handles soft deletes or Cascade if configured in DB.
		// For this project, we explicitly delete or rely on GORM's delete which honors foreign key constraints if set.
		// However, it's safer to delete them or ensure they are deleted.
		// In models/legal_service.go, these should have ON DELETE CASCADE or we do it here.

		// Delete milestones
		if err := tx.Where("service_id = ?", serviceID).Delete(&models.ServiceMilestone{}).Error; err != nil {
			return err
		}

		// Delete documents references (but files remain in storage?)
		// Usually we keep files or delete them from storage too.
		// For now, let's stick to DB deletion.
		if err := tx.Where("service_id = ?", serviceID).Delete(&models.ServiceDocument{}).Error; err != nil {
			return err
		}

		// Delete expenses
		if err := tx.Where("service_id = ?", serviceID).Delete(&models.ServiceExpense{}).Error; err != nil {
			return err
		}

		// Delete collaborators
		// Many-to-many relationship
		if err := tx.Model(&service).Association("Collaborators").Clear(); err != nil {
			return err
		}

		// Finally delete the service
		if err := tx.Delete(&service).Error; err != nil {
			return err
		}

		return nil
	})
}

// GetServicesByClient retrieves all services for a specific client
func GetServicesByClient(db *gorm.DB, firmID, clientID string) ([]models.LegalService, error) {
	var services []models.LegalService
	err := db.Where("firm_id = ? AND client_id = ?", firmID, clientID).
		Preload("ServiceType").
		Preload("AssignedTo").
		Order("created_at DESC").
		Find(&services).Error
	return services, err
}

// GetServicesByAssignee retrieves all services assigned to a specific user
func GetServicesByAssignee(db *gorm.DB, firmID, userID string) ([]models.LegalService, error) {
	var services []models.LegalService
	err := db.Where("firm_id = ? AND assigned_to_id = ?", firmID, userID).
		Preload("Client").
		Preload("ServiceType").
		Order("created_at DESC").
		Find(&services).Error
	return services, err
}

// GetActiveServicesCount returns the count of active services for a firm
func GetActiveServicesCount(db *gorm.DB, firmID string) (int64, error) {
	var count int64
	err := db.Model(&models.LegalService{}).
		Where("firm_id = ? AND status IN ?", firmID,
			[]string{models.ServiceStatusIntake, models.ServiceStatusInProgress}).
		Count(&count).Error
	return count, err
}

// ServiceSummary holds summary statistics for services
type ServiceSummary struct {
	TotalServices     int64   `json:"total_services"`
	ActiveServices    int64   `json:"active_services"`
	CompletedServices int64   `json:"completed_services"`
	TotalHours        float64 `json:"total_hours"`
	TotalExpenses     float64 `json:"total_expenses"`
}

// GetServiceSummary returns summary statistics for services in a firm
func GetServiceSummary(db *gorm.DB, firmID string) (*ServiceSummary, error) {
	summary := &ServiceSummary{}

	// Total services
	if err := db.Model(&models.LegalService{}).
		Where("firm_id = ?", firmID).
		Count(&summary.TotalServices).Error; err != nil {
		return nil, err
	}

	// Active services
	if err := db.Model(&models.LegalService{}).
		Where("firm_id = ? AND status IN ?", firmID,
			[]string{models.ServiceStatusIntake, models.ServiceStatusInProgress}).
		Count(&summary.ActiveServices).Error; err != nil {
		return nil, err
	}

	// Completed services
	if err := db.Model(&models.LegalService{}).
		Where("firm_id = ? AND status = ?", firmID, models.ServiceStatusCompleted).
		Count(&summary.CompletedServices).Error; err != nil {
		return nil, err
	}

	// Total hours
	if err := db.Model(&models.LegalService{}).
		Where("firm_id = ?", firmID).
		Select("COALESCE(SUM(actual_hours), 0)").
		Scan(&summary.TotalHours).Error; err != nil {
		return nil, err
	}

	// Total expenses (approved + paid)
	if err := db.Model(&models.ServiceExpense{}).
		Joins("JOIN legal_services ON legal_services.id = service_expenses.service_id").
		Where("legal_services.firm_id = ? AND service_expenses.status IN ?",
			firmID, []string{models.ExpenseStatusApproved, models.ExpenseStatusPaid}).
		Select("COALESCE(SUM(service_expenses.amount), 0)").
		Scan(&summary.TotalExpenses).Error; err != nil {
		return nil, err
	}

	return summary, nil
}
