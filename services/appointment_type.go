package services

import (
	"law_flow_app_go/models"

	"gorm.io/gorm"
)

// GetAppointmentTypes returns all appointment types for a firm
func GetAppointmentTypes(db *gorm.DB, firmID string) ([]models.AppointmentType, error) {
	var types []models.AppointmentType
	err := db.Where("firm_id = ?", firmID).
		Order(`"order" asc, name asc`).
		Find(&types).Error
	return types, err
}

// GetActiveAppointmentTypes returns only active appointment types for a firm
func GetActiveAppointmentTypes(db *gorm.DB, firmID string) ([]models.AppointmentType, error) {
	var types []models.AppointmentType
	err := db.Where("firm_id = ? AND is_active = ?", firmID, true).
		Order(`"order" asc, name asc`).
		Find(&types).Error
	return types, err
}

// GetAppointmentTypeByID fetches a single appointment type
func GetAppointmentTypeByID(db *gorm.DB, id string) (*models.AppointmentType, error) {
	var aptType models.AppointmentType
	err := db.First(&aptType, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &aptType, nil
}

// CreateAppointmentType creates a new appointment type
func CreateAppointmentType(db *gorm.DB, aptType *models.AppointmentType) error {
	return db.Create(aptType).Error
}

// UpdateAppointmentType updates an appointment type
func UpdateAppointmentType(db *gorm.DB, id string, updates map[string]interface{}) error {
	return db.Model(&models.AppointmentType{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteAppointmentType soft deletes an appointment type
func DeleteAppointmentType(db *gorm.DB, id string) error {
	return db.Delete(&models.AppointmentType{}, "id = ?", id).Error
}

// EnsureDefaultAppointmentTypes creates default types if none exist for a firm
func EnsureDefaultAppointmentTypes(db *gorm.DB, firmID string) error {
	var count int64
	db.Model(&models.AppointmentType{}).Where("firm_id = ?", firmID).Count(&count)
	if count == 0 {
		return models.CreateDefaultAppointmentTypes(db, firmID)
	}
	return nil
}
