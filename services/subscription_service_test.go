package services

import (
	"law_flow_app_go/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSubscriptionTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(
		&models.Plan{},
		&models.PlanAddOn{},
		&models.Firm{},
		&models.FirmSubscription{},
		&models.FirmAddOn{},
		&models.FirmUsage{},
		&models.User{},
		&models.Case{},
		&models.CaseDocument{},
	)
	return db
}

func TestLimitChecks(t *testing.T) {
	db := setupSubscriptionTestDB()
	SeedDefaultPlans(db)
	SeedDefaultAddOns(db)

	firmID := "f1"
	db.Create(&models.Firm{ID: firmID, Name: "Test Firm"})
	CreateTrialSubscription(db, firmID)

	t.Run("CanAddUser check", func(t *testing.T) {
		// Trial plan has MaxUsers: 2
		result, err := CanAddUser(db, firmID)
		assert.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, int64(2), result.Limit)

		// Reach limit
		db.Model(&models.FirmUsage{}).Where("firm_id = ?", firmID).Update("current_users", 2)
		result, err = CanAddUser(db, firmID)
		assert.ErrorIs(t, err, ErrUserLimitReached)
		assert.False(t, result.Allowed)
	})

	t.Run("CanAddCase check", func(t *testing.T) {
		// Trial plan has MaxCases: 20
		db.Model(&models.FirmUsage{}).Where("firm_id = ?", firmID).Update("current_cases", 19)
		result, err := CanAddCase(db, firmID)
		assert.NoError(t, err)
		assert.True(t, result.Allowed)

		db.Model(&models.FirmUsage{}).Where("firm_id = ?", firmID).Update("current_cases", 20)
		result, err = CanAddCase(db, firmID)
		assert.ErrorIs(t, err, ErrCaseLimitReached)
		assert.False(t, result.Allowed)
	})
}

func TestEffectiveLimitsWithAddOns(t *testing.T) {
	db := setupSubscriptionTestDB()
	SeedDefaultPlans(db)
	SeedDefaultAddOns(db)

	firmID := "f1"
	db.Create(&models.Firm{ID: firmID, Name: "Test Firm"})

	var starterPlan models.Plan
	db.Where("tier = ?", models.PlanTierStarter).First(&starterPlan)
	db.Create(&models.FirmSubscription{FirmID: firmID, PlanID: starterPlan.ID, Status: "active"})

	// Starter MaxUsers: 5
	assert.Equal(t, 5, GetEffectiveUserLimit(db, firmID, &starterPlan))

	// Purchase 2 User Packs (+5 each = +10)
	var userPack models.PlanAddOn
	db.Where("type = ?", models.AddOnTypeUsers).First(&userPack)
	db.Create(&models.FirmAddOn{FirmID: firmID, AddOnID: userPack.ID, Quantity: 2, IsActive: true})

	assert.Equal(t, 15, GetEffectiveUserLimit(db, firmID, &starterPlan))
}

func TestRecalculateUsage(t *testing.T) {
	db := setupSubscriptionTestDB()
	firmID := "f1"
	db.Create(&models.Firm{ID: firmID})

	// Add users
	db.Create(&models.User{ID: "u1", FirmID: &firmID, Role: "admin", IsActive: true, Email: "u1@test.com"})
	db.Create(&models.User{ID: "u2", FirmID: &firmID, Role: "lawyer", IsActive: true, Email: "u2@test.com"})
	db.Create(&models.User{ID: "u3", FirmID: &firmID, Role: "client", IsActive: true, Email: "u3@test.com"}) // Clients don't count

	// Add cases
	db.Create(&models.Case{ID: "c1", FirmID: firmID, IsDeleted: false, CaseNumber: "CN1"})
	db.Create(&models.Case{ID: "c2", FirmID: firmID, IsDeleted: true, CaseNumber: "CN2"}) // Deleted doesn't count

	// Add docs
	db.Create(&models.CaseDocument{ID: "d1", FirmID: firmID, FileSize: 100})
	db.Create(&models.CaseDocument{ID: "d2", FirmID: firmID, FileSize: 200})

	usage, err := RecalculateFirmUsage(db, firmID)
	assert.NoError(t, err)
	assert.Equal(t, 2, usage.CurrentUsers)
	assert.Equal(t, 1, usage.CurrentCases)
	assert.Equal(t, int64(300), usage.CurrentStorageBytes)
}

func TestChangePlan(t *testing.T) {
	db := setupSubscriptionTestDB()
	SeedDefaultPlans(db)
	firmID := "f1"
	db.Create(&models.Firm{ID: firmID})
	CreateTrialSubscription(db, firmID)

	var starterPlan models.Plan
	db.Where("tier = ?", models.PlanTierStarter).First(&starterPlan)

	err := ChangeFirmPlan(db, firmID, starterPlan.ID, stringToPtr("admin-user"))
	assert.NoError(t, err)

	var sub models.FirmSubscription
	db.Where("firm_id = ?", firmID).First(&sub)
	assert.Equal(t, starterPlan.ID, sub.PlanID)
	assert.Equal(t, models.SubscriptionStatusActive, sub.Status)
	assert.Nil(t, sub.TrialEndsAt)
}

func TestExtendTrial(t *testing.T) {
	db := setupSubscriptionTestDB()
	SeedDefaultPlans(db)
	firmID := "f1"
	db.Create(&models.Firm{ID: firmID})
	CreateTrialSubscription(db, firmID)

	var before models.FirmSubscription
	db.Where("firm_id = ?", firmID).First(&before)
	originalEnd := *before.TrialEndsAt

	err := ExtendTrial(db, firmID, 7)
	assert.NoError(t, err)

	var after models.FirmSubscription
	db.Where("firm_id = ?", firmID).First(&after)
	assert.True(t, after.TrialEndsAt.After(originalEnd))

	// Should be approx 7 days later
	diff := after.TrialEndsAt.Sub(originalEnd)
	assert.InDelta(t, (7 * 24 * time.Hour).Seconds(), diff.Seconds(), 1)
}
