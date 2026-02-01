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
	t.Run("CanAddClient check", func(t *testing.T) {
		// Trial plan has MaxCases: 20
		// Should be able to add 19 clients
		for i := 0; i < 19; i++ {
			email := "client" + string(rune(i)) + "@test.com"
			db.Create(&models.User{ID: "c" + string(rune(i)), FirmID: &firmID, Role: "client", IsActive: true, Email: email})
		}

		result, err := CanAddClient(db, firmID)
		assert.NoError(t, err)
		assert.True(t, result.Allowed)

		// Add one more (20th client) - limit is 20
		db.Create(&models.User{ID: "c20", FirmID: &firmID, Role: "client", IsActive: true, Email: "c20@test.com"})

		// Attempt to add 21st
		result, err = CanAddClient(db, firmID)
		assert.ErrorIs(t, err, ErrClientLimitReached)
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
func TestSubscriptionInfo(t *testing.T) {
	db := setupSubscriptionTestDB()
	SeedDefaultPlans(db)
	SeedDefaultAddOns(db)

	firmID := "f-info"
	db.Create(&models.Firm{ID: firmID, Name: "Info Firm"})
	CreateTrialSubscription(db, firmID)

	t.Run("GetFirmSubscriptionInfo full success", func(t *testing.T) {
		info, err := GetFirmSubscriptionInfo(db, firmID)
		assert.NoError(t, err)
		assert.NotNil(t, info)
		t.Logf("DEBUG: Plan Tier: %s, TemplatesEnabled: %v", info.Plan.Tier, info.Plan.TemplatesEnabled)
		assert.Equal(t, models.SubscriptionStatusTrialing, info.Subscription.Status)
		assert.False(t, info.HasTemplates) // Trial does NOT include templates in SeedDefaultPlans
	})

	t.Run("GetFirmSubscription not found", func(t *testing.T) {
		_, err := GetFirmSubscription(db, "non-existent")
		assert.Error(t, err)
	})
}

func TestEffectiveLimitsStorageAndCases(t *testing.T) {
	db := setupSubscriptionTestDB()
	SeedDefaultPlans(db)
	SeedDefaultAddOns(db)

	firmID := "f-limits"
	db.Create(&models.Firm{ID: firmID})

	var starterPlan models.Plan
	db.Where("tier = ?", models.PlanTierStarter).First(&starterPlan)
	db.Create(&models.FirmSubscription{FirmID: firmID, PlanID: starterPlan.ID, Status: "active"})

	t.Run("Storage with add-ons", func(t *testing.T) {
		// Starter: 1GB = 1073741824 bytes
		limit := GetEffectiveStorageLimit(db, firmID, &starterPlan)
		assert.Equal(t, starterPlan.MaxStorageBytes, limit)

		// Add 10GB pack
		var storagePack models.PlanAddOn
		db.Where("type = ?", models.AddOnTypeStorage).First(&storagePack)
		db.Create(&models.FirmAddOn{FirmID: firmID, AddOnID: storagePack.ID, Quantity: 2, IsActive: true})

		limitWithAddon := GetEffectiveStorageLimit(db, firmID, &starterPlan)
		expected := starterPlan.MaxStorageBytes + (storagePack.StorageBytes * 2)
		assert.Equal(t, expected, limitWithAddon)
	})

	t.Run("Cases with add-ons", func(t *testing.T) {
		// Starter: 50 cases
		limit := GetEffectiveCaseLimit(db, firmID, &starterPlan)
		assert.Equal(t, 50, limit)

		var casePack models.PlanAddOn
		db.Where("type = ?", models.AddOnTypeCases).First(&casePack)
		db.Create(&models.FirmAddOn{FirmID: firmID, AddOnID: casePack.ID, Quantity: 1, IsActive: true})

		limitWithAddon := GetEffectiveCaseLimit(db, firmID, &starterPlan)
		assert.Equal(t, 50+casePack.UnitsIncluded, limitWithAddon)
	})
}

func TestTemplateAccess(t *testing.T) {
	db := setupSubscriptionTestDB()
	SeedDefaultPlans(db)
	SeedDefaultAddOns(db)

	firmID := "f-templates"
	db.Create(&models.Firm{ID: firmID})

	var starterPlan models.Plan
	db.Where("tier = ?", models.PlanTierStarter).First(&starterPlan)
	db.Create(&models.FirmSubscription{FirmID: firmID, PlanID: starterPlan.ID, Status: "active"})

	t.Run("No templates in trial", func(t *testing.T) {
		fTrial := "f-trial-templates"
		db.Create(&models.Firm{ID: fTrial, Name: "Trial Firm"})
		CreateTrialSubscription(db, fTrial)

		allowed, err := CanAccessTemplates(db, fTrial)
		assert.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("Templates via add-on", func(t *testing.T) {
		var templateAddon models.PlanAddOn
		db.Where("type = ?", models.AddOnTypeTemplates).First(&templateAddon)
		db.Create(&models.FirmAddOn{FirmID: firmID, AddOnID: templateAddon.ID, Quantity: 1, IsActive: true})

		allowed, err := CanAccessTemplates(db, firmID)
		assert.NoError(t, err)
		assert.True(t, allowed)
	})
}

func TestFileUploadLimit(t *testing.T) {
	db := setupSubscriptionTestDB()
	SeedDefaultPlans(db)

	firmID := "f-upload"
	db.Create(&models.Firm{ID: firmID})

	var starterPlan models.Plan
	db.Where("tier = ?", models.PlanTierStarter).First(&starterPlan)
	db.Create(&models.FirmSubscription{FirmID: firmID, PlanID: starterPlan.ID, Status: "active"})

	// Create usage
	db.Create(&models.FirmUsage{FirmID: firmID, CurrentStorageBytes: starterPlan.MaxStorageBytes - 100, LastCalculatedAt: time.Now()})

	t.Run("Within limit", func(t *testing.T) {
		res, err := CanUploadFile(db, firmID, 50)
		assert.NoError(t, err)
		assert.True(t, res.Allowed)
	})

	t.Run("Exceed limit", func(t *testing.T) {
		res, err := CanUploadFile(db, firmID, 150)
		assert.ErrorIs(t, err, ErrStorageLimitReached)
		assert.False(t, res.Allowed)
	})
}

func TestUsageUpdateFunctions(t *testing.T) {
	db := setupSubscriptionTestDB()
	firmID := "f-usage-updates"
	db.Create(&models.Firm{ID: firmID})
	db.Create(&models.FirmUsage{FirmID: firmID, CurrentUsers: 1, CurrentCases: 1, CurrentStorageBytes: 100, LastCalculatedAt: time.Now()})

	t.Run("UpdateUserUsage", func(t *testing.T) {
		err := UpdateFirmUsageAfterUserChange(db, firmID, 1)
		assert.NoError(t, err)
		usage, _ := GetOrCalculateFirmUsage(db, firmID)
		assert.Equal(t, 2, usage.CurrentUsers)
	})

	t.Run("UpdateCaseUsage", func(t *testing.T) {
		err := UpdateFirmUsageAfterCaseChange(db, firmID, -1)
		assert.NoError(t, err)
		usage, _ := GetOrCalculateFirmUsage(db, firmID)
		assert.Equal(t, 0, usage.CurrentCases)
	})

	t.Run("UpdateStorageUsage", func(t *testing.T) {
		err := UpdateFirmUsageAfterStorageChange(db, firmID, 500)
		assert.NoError(t, err)
		usage, _ := GetOrCalculateFirmUsage(db, firmID)
		assert.Equal(t, int64(600), usage.CurrentStorageBytes)
	})
}

func TestTrialEdgeCases(t *testing.T) {
	db := setupSubscriptionTestDB()
	SeedDefaultPlans(db)
	firmID := "f-trial-edges"
	db.Create(&models.Firm{ID: firmID})

	t.Run("No subscription error", func(t *testing.T) {
		_, err := CanAddUser(db, firmID)
		assert.ErrorIs(t, err, ErrNoActiveSubscription)
	})

	t.Run("Expired trial", func(t *testing.T) {
		CreateTrialSubscription(db, firmID)
		past := time.Now().AddDate(0, 0, -1)
		db.Model(&models.FirmSubscription{}).Where("firm_id = ?", firmID).Update("trial_ends_at", past)

		res, err := CanAddUser(db, firmID)
		assert.ErrorIs(t, err, ErrSubscriptionExpired)
		assert.False(t, res.Allowed)
	})

	t.Run("Extend trial not trialing", func(t *testing.T) {
		f2 := "f-no-trial"
		db.Create(&models.Firm{ID: f2})
		db.Create(&models.FirmSubscription{FirmID: f2, Status: "active"})
		err := ExtendTrial(db, f2, 5)
		assert.Error(t, err)
	})
}
