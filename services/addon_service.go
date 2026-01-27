package services

import (
	"errors"
	"fmt"
	"law_flow_app_go/models"
	"time"

	"gorm.io/gorm"
)

// Add-on related errors
var (
	ErrAddOnNotFound     = errors.New("add-on not found")
	ErrAddOnAlreadyOwned = errors.New("add-on already owned")
	ErrInvalidAddOnType  = errors.New("invalid add-on type")
	ErrCannotStackAddOn  = errors.New("this add-on cannot be stacked")
)

// GetAvailableAddOns returns all active add-ons
func GetAvailableAddOns(db *gorm.DB) ([]models.PlanAddOn, error) {
	var addOns []models.PlanAddOn
	err := db.Where("is_active = ?", true).
		Order("display_order ASC, name ASC").
		Find(&addOns).Error
	return addOns, err
}

// GetAvailableAddOnsByType returns active add-ons of a specific type
func GetAvailableAddOnsByType(db *gorm.DB, addonType string) ([]models.PlanAddOn, error) {
	var addOns []models.PlanAddOn
	err := db.Where("is_active = ? AND type = ?", true, addonType).
		Order("display_order ASC, name ASC").
		Find(&addOns).Error
	return addOns, err
}

// GetFirmAddOns returns all active add-ons for a firm
func GetFirmAddOns(db *gorm.DB, firmID string) ([]models.FirmAddOn, error) {
	var addOns []models.FirmAddOn
	err := db.Preload("AddOn").
		Where("firm_id = ? AND is_active = ?", firmID, true).
		Find(&addOns).Error
	return addOns, err
}

// GetFirmAddOnsByType returns firm's add-ons of a specific type
func GetFirmAddOnsByType(db *gorm.DB, firmID string, addonType string) ([]models.FirmAddOn, error) {
	var addOns []models.FirmAddOn
	err := db.Preload("AddOn").
		Joins("JOIN plan_addons ON firm_addons.add_on_id = plan_addons.id").
		Where("firm_addons.firm_id = ? AND firm_addons.is_active = ? AND plan_addons.type = ?",
			firmID, true, addonType).
		Find(&addOns).Error
	return addOns, err
}

// PurchaseAddOn adds an add-on to a firm's subscription
func PurchaseAddOn(db *gorm.DB, firmID string, addOnID string, quantity int, purchasedByUserID *string) error {
	// Get the add-on
	var addOn models.PlanAddOn
	if err := db.First(&addOn, "id = ? AND is_active = ?", addOnID, true).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAddOnNotFound
		}
		return err
	}

	// Check if firm already owns this add-on (for non-stackable add-ons like templates)
	if addOn.Type == models.AddOnTypeTemplates {
		var existing models.FirmAddOn
		err := db.Where("firm_id = ? AND add_on_id = ? AND is_active = ?", firmID, addOnID, true).First(&existing).Error
		if err == nil {
			return ErrAddOnAlreadyOwned
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}

	now := time.Now()

	firmAddOn := &models.FirmAddOn{
		FirmID:            firmID,
		AddOnID:           addOnID,
		Quantity:          quantity,
		IsActive:          true,
		PurchasedByUserID: purchasedByUserID,
	}

	if addOn.IsRecurring {
		firmAddOn.StartedAt = &now
		// For recurring add-ons, set expiry to 1 month
		expiresAt := now.AddDate(0, 1, 0)
		firmAddOn.ExpiresAt = &expiresAt
	} else {
		// One-time purchase (like templates)
		firmAddOn.PurchasedAt = &now
		firmAddOn.IsPermanent = true
	}

	return db.Create(firmAddOn).Error
}

// IncreaseAddOnQuantity increases the quantity of a stackable add-on
func IncreaseAddOnQuantity(db *gorm.DB, firmAddOnID string, additionalQuantity int) error {
	var firmAddOn models.FirmAddOn
	if err := db.Preload("AddOn").First(&firmAddOn, "id = ?", firmAddOnID).Error; err != nil {
		return err
	}

	// Only allow stacking for recurring add-ons
	if !firmAddOn.AddOn.IsRecurring {
		return ErrCannotStackAddOn
	}

	firmAddOn.Quantity += additionalQuantity
	return db.Save(&firmAddOn).Error
}

// CancelAddOn cancels/deactivates a firm's add-on
func CancelAddOn(db *gorm.DB, firmAddOnID string) error {
	return db.Model(&models.FirmAddOn{}).
		Where("id = ?", firmAddOnID).
		Update("is_active", false).Error
}

// PurchaseTemplates is a convenience function to purchase the templates add-on
func PurchaseTemplates(db *gorm.DB, firmID string, purchasedByUserID *string) error {
	// Find the templates add-on
	var templatesAddOn models.PlanAddOn
	if err := db.Where("type = ? AND is_active = ?", models.AddOnTypeTemplates, true).First(&templatesAddOn).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("templates add-on not available")
		}
		return err
	}

	return PurchaseAddOn(db, firmID, templatesAddOn.ID, 1, purchasedByUserID)
}

// GetAddOnSummary returns a summary of firm's add-on purchases
type AddOnSummary struct {
	ExtraUsers   int
	ExtraStorage int64
	ExtraCases   int
	HasTemplates bool
	AddOns       []models.FirmAddOn
}

// GetFirmAddOnSummary calculates the total extra capacity from add-ons
func GetFirmAddOnSummary(db *gorm.DB, firmID string) (*AddOnSummary, error) {
	addOns, err := GetFirmAddOns(db, firmID)
	if err != nil {
		return nil, err
	}

	summary := &AddOnSummary{
		AddOns: addOns,
	}

	for _, fa := range addOns {
		if !fa.IsValid() {
			continue
		}

		switch fa.AddOn.Type {
		case models.AddOnTypeUsers:
			summary.ExtraUsers += fa.AddOn.UnitsIncluded * fa.Quantity
		case models.AddOnTypeStorage:
			summary.ExtraStorage += fa.AddOn.StorageBytes * int64(fa.Quantity)
		case models.AddOnTypeCases:
			summary.ExtraCases += fa.AddOn.UnitsIncluded * fa.Quantity
		case models.AddOnTypeTemplates:
			summary.HasTemplates = true
		}
	}

	return summary, nil
}

// RenewRecurringAddOns extends the expiry of recurring add-ons
// This would typically be called after successful payment
func RenewRecurringAddOns(db *gorm.DB, firmID string) error {
	now := time.Now()
	newExpiry := now.AddDate(0, 1, 0)

	return db.Model(&models.FirmAddOn{}).
		Where("firm_addons.firm_id = ? AND firm_addons.is_active = ? AND add_on_id IN (?)",
			firmID, true,
			db.Model(&models.PlanAddOn{}).Select("id").Where("is_recurring = ?", true),
		).
		Updates(map[string]interface{}{
			"expires_at": newExpiry,
		}).Error
}

// ExpireAddOns marks expired add-ons as inactive
// This should be run as a scheduled job
func ExpireAddOns(db *gorm.DB) error {
	return db.Model(&models.FirmAddOn{}).
		Where("is_active = ? AND is_permanent = ? AND expires_at < ?", true, false, time.Now()).
		Update("is_active", false).Error
}
