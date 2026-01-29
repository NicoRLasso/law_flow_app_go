package services

import (
	"errors"
	"law_flow_app_go/models"
	"time"

	"gorm.io/gorm"
)

// Milestone-related errors
var (
	ErrMilestoneNotFound = errors.New("milestone not found")
)

// MilestoneProgress holds milestone completion statistics
type MilestoneProgress struct {
	Total     int `json:"total"`
	Completed int `json:"completed"`
	Pending   int `json:"pending"`
	Skipped   int `json:"skipped"`
	Percent   int `json:"percent"` // Completion percentage (0-100)
}

// GetMilestonesByService retrieves all milestones for a service ordered by sort_order
func GetMilestonesByService(db *gorm.DB, serviceID string) ([]models.ServiceMilestone, error) {
	var milestones []models.ServiceMilestone
	err := db.Where("service_id = ?", serviceID).
		Preload("OutputDocument").
		Preload("Completer").
		Order("sort_order ASC").
		Find(&milestones).Error
	return milestones, err
}

// GetMilestoneByID retrieves a milestone by ID
func GetMilestoneByID(db *gorm.DB, milestoneID string) (*models.ServiceMilestone, error) {
	var milestone models.ServiceMilestone
	err := db.Preload("OutputDocument").
		Preload("Completer").
		First(&milestone, "id = ?", milestoneID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMilestoneNotFound
		}
		return nil, err
	}
	return &milestone, nil
}

// GetMilestoneProgress returns completion statistics for a service's milestones
func GetMilestoneProgress(db *gorm.DB, serviceID string) (*MilestoneProgress, error) {
	progress := &MilestoneProgress{}

	// Get all milestone counts by status
	type statusCount struct {
		Status string
		Count  int
	}
	var counts []statusCount

	err := db.Model(&models.ServiceMilestone{}).
		Where("service_id = ?", serviceID).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&counts).Error

	if err != nil {
		return nil, err
	}

	for _, c := range counts {
		progress.Total += c.Count
		switch c.Status {
		case models.MilestoneStatusCompleted:
			progress.Completed = c.Count
		case models.MilestoneStatusSkipped:
			progress.Skipped = c.Count
		case models.MilestoneStatusPending, models.MilestoneStatusInProgress:
			progress.Pending += c.Count
		}
	}

	// Calculate percentage (completed + skipped count as done)
	if progress.Total > 0 {
		done := progress.Completed + progress.Skipped
		progress.Percent = (done * 100) / progress.Total
	}

	return progress, nil
}

// CompleteMilestone marks a milestone as completed
func CompleteMilestone(db *gorm.DB, milestoneID, userID string) error {
	now := time.Now()
	return db.Model(&models.ServiceMilestone{}).
		Where("id = ?", milestoneID).
		Updates(map[string]interface{}{
			"status":       models.MilestoneStatusCompleted,
			"completed_at": now,
			"completed_by": userID,
		}).Error
}

// SkipMilestone marks a milestone as skipped
func SkipMilestone(db *gorm.DB, milestoneID, userID string) error {
	now := time.Now()
	return db.Model(&models.ServiceMilestone{}).
		Where("id = ?", milestoneID).
		Updates(map[string]interface{}{
			"status":       models.MilestoneStatusSkipped,
			"completed_at": now,
			"completed_by": userID,
		}).Error
}

// StartMilestone marks a milestone as in progress
func StartMilestone(db *gorm.DB, milestoneID string) error {
	return db.Model(&models.ServiceMilestone{}).
		Where("id = ?", milestoneID).
		Update("status", models.MilestoneStatusInProgress).Error
}

// ResetMilestone resets a milestone to pending status
func ResetMilestone(db *gorm.DB, milestoneID string) error {
	return db.Model(&models.ServiceMilestone{}).
		Where("id = ?", milestoneID).
		Updates(map[string]interface{}{
			"status":       models.MilestoneStatusPending,
			"completed_at": nil,
			"completed_by": nil,
		}).Error
}

// ReorderMilestones updates the sort order of milestones
func ReorderMilestones(db *gorm.DB, serviceID string, milestoneIDs []string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for i, id := range milestoneIDs {
			if err := tx.Model(&models.ServiceMilestone{}).
				Where("id = ? AND service_id = ?", id, serviceID).
				Update("sort_order", i+1).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// LinkDocumentToMilestone links a document as the output of a milestone
func LinkDocumentToMilestone(db *gorm.DB, milestoneID, documentID string) error {
	return db.Model(&models.ServiceMilestone{}).
		Where("id = ?", milestoneID).
		Update("output_document_id", documentID).Error
}

// CreateDefaultMilestones creates default milestones based on service type
func CreateDefaultMilestones(db *gorm.DB, service *models.LegalService) error {
	// Get service type code if available
	serviceTypeCode := ""
	if service.ServiceType != nil {
		serviceTypeCode = service.ServiceType.Code
	} else if service.ServiceTypeID != nil {
		// Load the service type to get the code
		var option models.ChoiceOption
		if err := db.First(&option, "id = ?", service.ServiceTypeID).Error; err == nil {
			serviceTypeCode = option.Code
		}
	}

	// Define default milestones based on service type
	var milestones []models.ServiceMilestone

	switch serviceTypeCode {
	case "COMPANY_FORMATION":
		milestones = []models.ServiceMilestone{
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Recolección de documentos", SortOrder: 1, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Redacción de estatutos", SortOrder: 2, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Registro en Cámara de Comercio", SortOrder: 3, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Obtención de RUT", SortOrder: 4, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Apertura de cuenta bancaria", SortOrder: 5, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Entrega final", SortOrder: 6, Status: models.MilestoneStatusPending},
		}
	case "VISA_PROCESSING":
		milestones = []models.ServiceMilestone{
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Recolección de documentos", SortOrder: 1, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Revisión de requisitos", SortOrder: 2, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Preparación de solicitud", SortOrder: 3, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Radicación ante autoridad", SortOrder: 4, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Seguimiento y respuesta", SortOrder: 5, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Entrega de visa", SortOrder: 6, Status: models.MilestoneStatusPending},
		}
	case "TRADEMARK_REGISTRATION":
		milestones = []models.ServiceMilestone{
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Búsqueda de antecedentes", SortOrder: 1, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Preparación de solicitud", SortOrder: 2, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Radicación ante SIC", SortOrder: 3, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Publicación en Gaceta", SortOrder: 4, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Período de oposiciones", SortOrder: 5, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Resolución de registro", SortOrder: 6, Status: models.MilestoneStatusPending},
		}
	case "NOTARIAL_PROCESS":
		milestones = []models.ServiceMilestone{
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Recolección de documentos", SortOrder: 1, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Redacción de minuta", SortOrder: 2, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Revisión por cliente", SortOrder: 3, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Firma en notaría", SortOrder: 4, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Entrega de escritura", SortOrder: 5, Status: models.MilestoneStatusPending},
		}
	case "CONTRACT_REVIEW":
		milestones = []models.ServiceMilestone{
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Recepción de contrato", SortOrder: 1, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Análisis jurídico", SortOrder: 2, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Elaboración de observaciones", SortOrder: 3, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Entrega de concepto", SortOrder: 4, Status: models.MilestoneStatusPending},
		}
	case "DOCUMENT_CREATION", "LEGAL_CONCEPT", "TAX_OPINION":
		milestones = []models.ServiceMilestone{
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Recolección de información", SortOrder: 1, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Elaboración de borrador", SortOrder: 2, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Revisión interna", SortOrder: 3, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Revisión por cliente", SortOrder: 4, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Entrega final", SortOrder: 5, Status: models.MilestoneStatusPending},
		}
	case "REAL_ESTATE":
		milestones = []models.ServiceMilestone{
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Recolección de documentos", SortOrder: 1, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Estudio de títulos", SortOrder: 2, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Verificación de gravámenes", SortOrder: 3, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Elaboración de concepto", SortOrder: 4, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Entrega de informe", SortOrder: 5, Status: models.MilestoneStatusPending},
		}
	default:
		// Generic milestones for other types
		milestones = []models.ServiceMilestone{
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Recepción de requerimiento", SortOrder: 1, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Análisis inicial", SortOrder: 2, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Desarrollo", SortOrder: 3, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Revisión", SortOrder: 4, Status: models.MilestoneStatusPending},
			{FirmID: service.FirmID, ServiceID: service.ID, Title: "Entrega", SortOrder: 5, Status: models.MilestoneStatusPending},
		}
	}

	// Create all milestones
	for i := range milestones {
		if err := db.Create(&milestones[i]).Error; err != nil {
			return err
		}
	}

	return nil
}

// GetNextPendingMilestone returns the next pending milestone for a service
func GetNextPendingMilestone(db *gorm.DB, serviceID string) (*models.ServiceMilestone, error) {
	var milestone models.ServiceMilestone
	err := db.Where("service_id = ? AND status = ?", serviceID, models.MilestoneStatusPending).
		Order("sort_order ASC").
		First(&milestone).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No pending milestones
		}
		return nil, err
	}
	return &milestone, nil
}

// AreAllMilestonesComplete checks if all milestones are completed or skipped
func AreAllMilestonesComplete(db *gorm.DB, serviceID string) (bool, error) {
	var pendingCount int64
	err := db.Model(&models.ServiceMilestone{}).
		Where("service_id = ? AND status IN ?", serviceID,
			[]string{models.MilestoneStatusPending, models.MilestoneStatusInProgress}).
		Count(&pendingCount).Error

	if err != nil {
		return false, err
	}
	return pendingCount == 0, nil
}
