package services

import (
	"fmt"
	"law_flow_app_go/models"
	"log"
	"time"

	"gorm.io/gorm"
)

const (
	GB = 1024 * 1024 * 1024 // 1 GB in bytes
)

// SeedDefaultPlans creates the default subscription plans
func SeedDefaultPlans(db *gorm.DB) error {
	plans := []models.Plan{
		{
			Name:             "Trial",
			Tier:             models.PlanTierTrial,
			Description:      "30-day free trial to explore the platform",
			PriceMonthly:     0,
			PriceYearly:      0,
			MaxUsers:         2,
			MaxStorageBytes:  1 * GB, // 1 GB
			MaxCases:         20,
			TemplatesEnabled: false, // Trial does NOT include templates
			TrialDays:        30,
			IsTrialPlan:      true,
			IsActive:         true,
			IsDefault:        true,
			DisplayOrder:     0,
		},
		{
			Name:             "Starter",
			Tier:             models.PlanTierStarter,
			Description:      "Perfect for small law firms",
			PriceMonthly:     3000,  // $30/month
			PriceYearly:      30000, // $300/year (2 months free)
			MaxUsers:         5,
			MaxStorageBytes:  5 * GB, // 5 GB
			MaxCases:         50,
			TemplatesEnabled: true,
			TrialDays:        0,
			IsTrialPlan:      false,
			IsActive:         true,
			IsDefault:        false,
			DisplayOrder:     1,
		},
		{
			Name:             "Professional",
			Tier:             models.PlanTierProfessional,
			Description:      "For growing law firms with more needs",
			PriceMonthly:     5000,  // $50/month
			PriceYearly:      50000, // $500/year (2 months free)
			MaxUsers:         10,
			MaxStorageBytes:  10 * GB, // 10 GB
			MaxCases:         150,
			TemplatesEnabled: true,
			TrialDays:        0,
			IsTrialPlan:      false,
			IsActive:         true,
			IsDefault:        false,
			DisplayOrder:     2,
		},
		{
			Name:             "Enterprise",
			Tier:             models.PlanTierEnterprise,
			Description:      "For large firms with high-volume needs",
			PriceMonthly:     12000,  // $120/month
			PriceYearly:      120000, // $1200/year (2 months free)
			MaxUsers:         15,
			MaxStorageBytes:  20 * GB, // 20 GB
			MaxCases:         500,
			TemplatesEnabled: true,
			TrialDays:        0,
			IsTrialPlan:      false,
			IsActive:         true,
			IsDefault:        false,
			DisplayOrder:     3,
		},
	}

	for _, plan := range plans {
		var existing models.Plan
		if err := db.Where("tier = ?", plan.Tier).First(&existing).Error; err == nil {
			log.Printf("[SEED] Plan %s already exists, skipping", plan.Name)
			continue
		}

		if err := db.Create(&plan).Error; err != nil {
			return fmt.Errorf("failed to create plan %s: %w", plan.Name, err)
		}
		log.Printf("[SEED] Created plan: %s", plan.Name)
	}

	return nil
}

// SeedDefaultAddOns creates the default purchasable add-ons
func SeedDefaultAddOns(db *gorm.DB) error {
	addOns := []models.PlanAddOn{
		{
			Name:          "Extra Users Pack",
			Description:   "Add 5 additional team members to your firm",
			Type:          models.AddOnTypeUsers,
			UnitsIncluded: 5,
			StorageBytes:  0,
			PriceMonthly:  1500, // $15/month
			PriceOneTime:  0,
			IsRecurring:   true,
			IsActive:      false,
			DisplayOrder:  1,
		},
		{
			Name:          "Extra Storage Pack",
			Description:   "Add 5 GB of additional document storage",
			Type:          models.AddOnTypeStorage,
			UnitsIncluded: 0,
			StorageBytes:  5 * GB, // 5 GB
			PriceMonthly:  1000,   // $10/month
			PriceOneTime:  0,
			IsRecurring:   true,
			IsActive:      false,
			DisplayOrder:  2,
		},
		{
			Name:          "Extra Cases Pack",
			Description:   "Add 50 additional case slots",
			Type:          models.AddOnTypeCases,
			UnitsIncluded: 50,
			StorageBytes:  0,
			PriceMonthly:  1000, // $10/month
			PriceOneTime:  0,
			IsRecurring:   true,
			IsActive:      false,
			DisplayOrder:  3,
		},
		{
			Name:          "Document Templates",
			Description:   "Unlock access to document templates for generating legal documents",
			Type:          models.AddOnTypeTemplates,
			UnitsIncluded: 0,
			StorageBytes:  0,
			PriceMonthly:  0,
			PriceOneTime:  2000, // $20 one-time
			IsRecurring:   false,
			IsActive:      false,
			DisplayOrder:  4,
		},
	}

	for _, addOn := range addOns {
		var existing models.PlanAddOn
		if err := db.Where("type = ? AND name = ?", addOn.Type, addOn.Name).First(&existing).Error; err == nil {
			log.Printf("[SEED] Add-on %s already exists, skipping", addOn.Name)
			continue
		}

		if err := db.Create(&addOn).Error; err != nil {
			return fmt.Errorf("failed to create add-on %s: %w", addOn.Name, err)
		}
		log.Printf("[SEED] Created add-on: %s", addOn.Name)
	}

	return nil
}

// MigrateExistingFirmsToTrial assigns trial subscriptions to existing firms without subscriptions
func MigrateExistingFirmsToTrial(db *gorm.DB) error {
	// Get default trial plan
	var trialPlan models.Plan
	if err := db.Where("is_trial_plan = ? AND is_active = ?", true, true).First(&trialPlan).Error; err != nil {
		log.Printf("[MIGRATION] No active trial plan found, skipping firm migration")
		return nil
	}

	// Find firms without subscriptions
	var firms []models.Firm
	if err := db.Where("id NOT IN (SELECT firm_id FROM firm_subscriptions WHERE deleted_at IS NULL)").Find(&firms).Error; err != nil {
		return err
	}

	if len(firms) == 0 {
		log.Printf("[MIGRATION] No firms without subscriptions found")
		return nil
	}

	trialEndDate := time.Now().AddDate(0, 0, 30) // 30 days from now

	for _, firm := range firms {
		subscription := &models.FirmSubscription{
			FirmID:      firm.ID,
			PlanID:      trialPlan.ID,
			Status:      models.SubscriptionStatusTrialing,
			StartedAt:   time.Now(),
			TrialEndsAt: &trialEndDate,
		}

		if err := db.Create(subscription).Error; err != nil {
			log.Printf("[MIGRATION] Failed to create subscription for firm %s: %v", firm.ID, err)
			continue
		}

		// Initialize usage tracking
		if _, err := RecalculateFirmUsage(db, firm.ID); err != nil {
			log.Printf("[MIGRATION] Failed to initialize usage for firm %s: %v", firm.ID, err)
		}

		log.Printf("[MIGRATION] Assigned trial subscription to firm: %s (%s)", firm.Name, firm.ID)
	}

	log.Printf("[MIGRATION] Completed migrating %d firms to trial subscriptions", len(firms))
	return nil
}

// InitializeSubscriptionSystem runs all setup tasks for the subscription system
func InitializeSubscriptionSystem(db *gorm.DB) error {
	log.Println("[SUBSCRIPTION] Initializing subscription system...")

	// Seed plans
	if err := SeedDefaultPlans(db); err != nil {
		return fmt.Errorf("failed to seed plans: %w", err)
	}

	// Seed add-ons
	if err := SeedDefaultAddOns(db); err != nil {
		return fmt.Errorf("failed to seed add-ons: %w", err)
	}

	// Migrate existing firms
	if err := MigrateExistingFirmsToTrial(db); err != nil {
		return fmt.Errorf("failed to migrate existing firms: %w", err)
	}

	log.Println("[SUBSCRIPTION] Subscription system initialized successfully")
	return nil
}
