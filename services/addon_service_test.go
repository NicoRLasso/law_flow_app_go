package services

import (
	"law_flow_app_go/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB initializes an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Migrate schemas
	err = db.AutoMigrate(&models.Firm{}, &models.PlanAddOn{}, &models.FirmAddOn{})
	assert.NoError(t, err)

	return db
}

func TestGetAvailableAddOns(t *testing.T) {
	db := setupTestDB(t)

	// Seed data
	active := models.PlanAddOn{Name: "Active Addon", IsActive: true, DisplayOrder: 1}
	inactive := models.PlanAddOn{Name: "Inactive Addon", IsActive: false, DisplayOrder: 2}
	db.Create(&active)
	db.Create(&inactive)
	db.Model(&inactive).Update("IsActive", false)

	// Test
	addOns, err := GetAvailableAddOns(db)
	assert.NoError(t, err)
	assert.Len(t, addOns, 1)
	assert.Equal(t, active.Name, addOns[0].Name)
}

func TestGetAvailableAddOnsByType(t *testing.T) {
	db := setupTestDB(t)

	// Seed data
	users := models.PlanAddOn{Name: "Users Addon", Type: models.AddOnTypeUsers, IsActive: true}
	storage := models.PlanAddOn{Name: "Storage Addon", Type: models.AddOnTypeStorage, IsActive: true}
	db.Create(&users)
	db.Create(&storage)

	// Test
	addOns, err := GetAvailableAddOnsByType(db, models.AddOnTypeUsers)
	assert.NoError(t, err)
	assert.Len(t, addOns, 1)
	assert.Equal(t, users.Name, addOns[0].Name)
}

func TestPurchaseAddOn_Recurring(t *testing.T) {
	db := setupTestDB(t)

	// Seed data
	firm := models.Firm{Name: "Test Firm"}
	db.Create(&firm)
	addOn := models.PlanAddOn{Name: "Recurring Addon", IsRecurring: true, IsActive: true}
	db.Create(&addOn)

	// Test
	err := PurchaseAddOn(db, firm.ID, addOn.ID, 2, nil)
	assert.NoError(t, err)

	var firmAddOn models.FirmAddOn
	err = db.Where("firm_id = ? AND add_on_id = ?", firm.ID, addOn.ID).First(&firmAddOn).Error
	assert.NoError(t, err)
	assert.Equal(t, 2, firmAddOn.Quantity)
	assert.True(t, firmAddOn.IsActive)
	assert.NotNil(t, firmAddOn.StartedAt)
	assert.NotNil(t, firmAddOn.ExpiresAt)
}

func TestPurchaseAddOn_Templates_OneTime(t *testing.T) {
	db := setupTestDB(t)

	// Seed data
	firm := models.Firm{Name: "Test Firm"}
	db.Create(&firm)
	addOn := models.PlanAddOn{Name: "Templates", Type: models.AddOnTypeTemplates, IsRecurring: false, IsActive: true}
	db.Create(&addOn)
	db.Model(&addOn).Update("IsRecurring", false)

	// Test Purchase
	err := PurchaseAddOn(db, firm.ID, addOn.ID, 1, nil)
	assert.NoError(t, err)

	var firmAddOn models.FirmAddOn
	err = db.Where("firm_id = ? AND add_on_id = ?", firm.ID, addOn.ID).First(&firmAddOn).Error
	assert.NoError(t, err)
	assert.True(t, firmAddOn.IsPermanent)
	assert.NotNil(t, firmAddOn.PurchasedAt)

	// Test Duplicate Purchase prevention
	err = PurchaseAddOn(db, firm.ID, addOn.ID, 1, nil)
	assert.ErrorIs(t, err, ErrAddOnAlreadyOwned)
}

func TestIncreaseAddOnQuantity(t *testing.T) {
	db := setupTestDB(t)

	// Seed data
	firm := models.Firm{Name: "Test Firm"}
	db.Create(&firm)
	addOn := models.PlanAddOn{Name: "Stackable Addon", IsRecurring: true, IsActive: true}
	db.Create(&addOn)

	// Initial Purchase
	PurchaseAddOn(db, firm.ID, addOn.ID, 1, nil)
	var firmAddOn models.FirmAddOn
	db.Where("firm_id = ? AND add_on_id = ?", firm.ID, addOn.ID).First(&firmAddOn)

	// Test Increase
	err := IncreaseAddOnQuantity(db, firmAddOn.ID, 2)
	assert.NoError(t, err)

	var updated models.FirmAddOn
	db.First(&updated, "id = ?", firmAddOn.ID)
	assert.Equal(t, 3, updated.Quantity)
}

func TestIncreaseAddOnQuantity_NonStackable(t *testing.T) {
	db := setupTestDB(t)

	// Seed data
	firm := models.Firm{Name: "Test Firm"}
	db.Create(&firm)
	addOn := models.PlanAddOn{Name: "Non-Stackable", IsRecurring: false, IsActive: true}
	db.Create(&addOn)
	db.Model(&addOn).Update("IsRecurring", false)

	// Purchase manually since Helper might check type logic? Actually helper logic is inside PurchaseAddOn which sets IsPermanent/etc.
	// But IncreaseAddOnQuantity checks firmAddOn.AddOn.IsRecurring.

	PurchaseAddOn(db, firm.ID, addOn.ID, 1, nil)
	var firmAddOn models.FirmAddOn
	db.Where("firm_id = ? AND add_on_id = ?", firm.ID, addOn.ID).First(&firmAddOn)

	// Test Increase
	err := IncreaseAddOnQuantity(db, firmAddOn.ID, 1)
	assert.ErrorIs(t, err, ErrCannotStackAddOn)
}

func TestRenewRecurringAddOns(t *testing.T) {
	db := setupTestDB(t)

	// Seed data
	firm := models.Firm{Name: "Test Firm"}
	db.Create(&firm)
	addOn := models.PlanAddOn{Name: "Recurring", IsRecurring: true, IsActive: true}
	db.Create(&addOn)

	// Create an expired/expiring addon
	oldExpiry := time.Now().AddDate(0, -1, 0)
	firmAddOn := models.FirmAddOn{
		FirmID:    firm.ID,
		AddOnID:   addOn.ID,
		IsActive:  true,
		ExpiresAt: &oldExpiry,
	}
	db.Create(&firmAddOn)

	// Test Renew
	err := RenewRecurringAddOns(db, firm.ID)
	assert.NoError(t, err)

	var updated models.FirmAddOn
	db.First(&updated, "id = ?", firmAddOn.ID)
	assert.True(t, updated.ExpiresAt.After(time.Now()))
}
func TestGetFirmAddOnsByType(t *testing.T) {
	db := setupTestDB(t)
	firmID := "f1"
	db.Create(&models.Firm{ID: firmID})

	a1 := models.PlanAddOn{Name: "U1", Type: models.AddOnTypeUsers}
	db.Create(&a1)
	db.Create(&models.FirmAddOn{FirmID: firmID, AddOnID: a1.ID, IsActive: true})

	addons, err := GetFirmAddOnsByType(db, firmID, models.AddOnTypeUsers)
	assert.NoError(t, err)
	assert.Len(t, addons, 1)
}

func TestCancelAddOn(t *testing.T) {
	db := setupTestDB(t)
	fa := models.FirmAddOn{IsActive: true}
	db.Create(&fa)

	err := CancelAddOn(db, fa.ID)
	assert.NoError(t, err)

	var updated models.FirmAddOn
	db.First(&updated, "id = ?", fa.ID)
	assert.False(t, updated.IsActive)
}

func TestPurchaseTemplates(t *testing.T) {
	db := setupTestDB(t)
	firmID := "f-templ"
	db.Create(&models.Firm{ID: firmID})

	addon := models.PlanAddOn{Name: "Templates", Type: models.AddOnTypeTemplates, PriceOneTime: 2000, IsActive: true}
	db.Create(&addon)

	err := PurchaseTemplates(db, firmID, nil)
	assert.NoError(t, err)

	// Second purchase should fail
	err = PurchaseTemplates(db, firmID, nil)
	assert.ErrorIs(t, err, ErrAddOnAlreadyOwned)
}

func TestExpireAddOns(t *testing.T) {
	db := setupTestDB(t)
	past := time.Now().AddDate(0, 0, -1)
	fa := models.FirmAddOn{IsActive: true, ExpiresAt: &past, IsPermanent: false}
	db.Create(&fa)

	err := ExpireAddOns(db)
	assert.NoError(t, err)

	var updated models.FirmAddOn
	db.First(&updated, "id = ?", fa.ID)
	assert.False(t, updated.IsActive)
}

func TestGetFirmAddOnSummary(t *testing.T) {
	db := setupTestDB(t)
	firmID := "f-summary"
	db.Create(&models.Firm{ID: firmID})

	uAddon := models.PlanAddOn{Type: models.AddOnTypeUsers, UnitsIncluded: 5, IsActive: true}
	db.Create(&uAddon)
	db.Create(&models.FirmAddOn{FirmID: firmID, AddOnID: uAddon.ID, Quantity: 1, IsActive: true})

	summary, err := GetFirmAddOnSummary(db, firmID)
	assert.NoError(t, err)
	assert.Equal(t, 5, summary.ExtraUsers)
}
