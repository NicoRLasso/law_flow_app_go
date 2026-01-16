package services

import (
	"law_flow_app_go/db"
	"law_flow_app_go/models"
)

// GetAppointmentTypes returns all appointment types for a firm
func GetAppointmentTypes(firmID string) ([]models.AppointmentType, error) {
	var types []models.AppointmentType
	err := db.DB.Where("firm_id = ?", firmID).
		Order(`"order" asc, name asc`).
		Find(&types).Error
	return types, err
}

// GetActiveAppointmentTypes returns only active appointment types for a firm
func GetActiveAppointmentTypes(firmID string) ([]models.AppointmentType, error) {
	var types []models.AppointmentType
	err := db.DB.Where("firm_id = ? AND is_active = ?", firmID, true).
		Order(`"order" asc, name asc`).
		Find(&types).Error
	return types, err
}

// GetAppointmentTypeByID fetches a single appointment type
func GetAppointmentTypeByID(id string) (*models.AppointmentType, error) {
	var aptType models.AppointmentType
	err := db.DB.First(&aptType, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &aptType, nil
}

// CreateAppointmentType creates a new appointment type
func CreateAppointmentType(aptType *models.AppointmentType) error {
	return db.DB.Create(aptType).Error
}

// UpdateAppointmentType updates an appointment type
func UpdateAppointmentType(id string, updates map[string]interface{}) error {
	return db.DB.Model(&models.AppointmentType{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteAppointmentType soft deletes an appointment type
func DeleteAppointmentType(id string) error {
	return db.DB.Delete(&models.AppointmentType{}, "id = ?", id).Error
}

// EnsureDefaultAppointmentTypes creates default types if none exist for a firm
func EnsureDefaultAppointmentTypes(firmID string) error {
	var count int64
	db.DB.Model(&models.AppointmentType{}).Where("firm_id = ?", firmID).Count(&count)
	if count == 0 {
		return models.CreateDefaultAppointmentTypes(db.DB, firmID)
	}
	return nil
}
