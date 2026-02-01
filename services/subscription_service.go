package services

import (
	"errors"
	"fmt"
	"law_flow_app_go/models"
	"time"

	"gorm.io/gorm"
)

// Limit check error types
var (
	ErrUserLimitReached     = errors.New("user limit reached for current plan")
	ErrStorageLimitReached  = errors.New("storage limit reached for current plan")
	ErrCaseLimitReached     = errors.New("case limit reached for current plan")
	ErrClientLimitReached   = errors.New("client limit reached for current plan")
	ErrTemplatesDisabled    = errors.New("templates feature not available on current plan")
	ErrSubscriptionExpired  = errors.New("subscription has expired")
	ErrNoActiveSubscription = errors.New("no active subscription found")
)

// LimitCheckResult contains the result of a limit check
type LimitCheckResult struct {
	Allowed         bool
	CurrentUsage    int64
	Limit           int64
	PercentageUsed  float64
	Message         string
	TranslationKey  string
	TranslationArgs map[string]interface{}
}

// SubscriptionInfo contains subscription details for display
type SubscriptionInfo struct {
	Subscription     *models.FirmSubscription
	Plan             *models.Plan
	Usage            *models.FirmUsage
	EffectiveUsers   int
	EffectiveStorage int64
	EffectiveCases   int
	UsersPercent     float64
	StoragePercent   float64
	CasesPercent     float64
	TrialDaysLeft    int
	ShowTrialWarning bool
	HasTemplates     bool
	ActiveAddOns     []models.FirmAddOn
}

// GetFirmSubscription retrieves the subscription for a firm
func GetFirmSubscription(db *gorm.DB, firmID string) (*models.FirmSubscription, error) {
	var subscription models.FirmSubscription
	err := db.Preload("Plan").Where("firm_id = ?", firmID).First(&subscription).Error
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

// GetFirmSubscriptionInfo retrieves full subscription info including usage and effective limits
func GetFirmSubscriptionInfo(db *gorm.DB, firmID string) (*SubscriptionInfo, error) {
	subscription, err := GetFirmSubscription(db, firmID)
	if err != nil {
		return nil, err
	}

	usage, err := GetOrCalculateFirmUsage(db, firmID)
	if err != nil {
		return nil, err
	}

	// Calculate effective limits (plan + add-ons)
	effectiveUsers := GetEffectiveUserLimit(db, firmID, &subscription.Plan)
	effectiveStorage := GetEffectiveStorageLimit(db, firmID, &subscription.Plan)
	effectiveCases := GetEffectiveCaseLimit(db, firmID, &subscription.Plan)
	hasTemplates := HasTemplatesAccess(db, firmID, &subscription.Plan)
	activeAddOns, _ := GetFirmAddOns(db, firmID)

	info := &SubscriptionInfo{
		Subscription:     subscription,
		Plan:             &subscription.Plan,
		Usage:            usage,
		EffectiveUsers:   effectiveUsers,
		EffectiveStorage: effectiveStorage,
		EffectiveCases:   effectiveCases,
		TrialDaysLeft:    subscription.TrialDaysRemaining(),
		ShowTrialWarning: subscription.ShouldShowTrialWarning(),
		HasTemplates:     hasTemplates,
		ActiveAddOns:     activeAddOns,
	}

	// Calculate percentages
	if effectiveUsers > 0 {
		info.UsersPercent = float64(usage.CurrentUsers) / float64(effectiveUsers) * 100
	}
	if effectiveStorage > 0 {
		info.StoragePercent = float64(usage.CurrentStorageBytes) / float64(effectiveStorage) * 100
	}
	if effectiveCases > 0 {
		info.CasesPercent = float64(usage.CurrentCases) / float64(effectiveCases) * 100
	}

	return info, nil
}

// GetEffectiveUserLimit returns the total user limit (plan + add-ons)
func GetEffectiveUserLimit(db *gorm.DB, firmID string, plan *models.Plan) int {
	if plan.IsUnlimitedUsers() {
		return -1
	}

	baseLimit := plan.MaxUsers

	// Add purchased add-ons
	var addOnUsers int
	db.Model(&models.FirmAddOn{}).
		Joins("JOIN plan_addons ON firm_addons.add_on_id = plan_addons.id").
		Where("firm_addons.firm_id = ? AND plan_addons.type = ? AND firm_addons.is_active = ?",
			firmID, models.AddOnTypeUsers, true).
		Select("COALESCE(SUM(plan_addons.units_included * firm_addons.quantity), 0)").
		Scan(&addOnUsers)

	return baseLimit + addOnUsers
}

// GetEffectiveStorageLimit returns the total storage limit (plan + add-ons)
func GetEffectiveStorageLimit(db *gorm.DB, firmID string, plan *models.Plan) int64 {
	if plan.IsUnlimitedStorage() {
		return -1
	}

	baseLimit := plan.MaxStorageBytes

	// Add purchased add-ons
	var addOnStorage int64
	db.Model(&models.FirmAddOn{}).
		Joins("JOIN plan_addons ON firm_addons.add_on_id = plan_addons.id").
		Where("firm_addons.firm_id = ? AND plan_addons.type = ? AND firm_addons.is_active = ?",
			firmID, models.AddOnTypeStorage, true).
		Select("COALESCE(SUM(plan_addons.storage_bytes * firm_addons.quantity), 0)").
		Scan(&addOnStorage)

	return baseLimit + addOnStorage
}

// GetEffectiveCaseLimit returns the total case limit (plan + add-ons)
func GetEffectiveCaseLimit(db *gorm.DB, firmID string, plan *models.Plan) int {
	if plan.IsUnlimitedCases() {
		return -1
	}

	baseLimit := plan.MaxCases

	// Add purchased add-ons
	var addOnCases int
	db.Model(&models.FirmAddOn{}).
		Joins("JOIN plan_addons ON firm_addons.add_on_id = plan_addons.id").
		Where("firm_addons.firm_id = ? AND plan_addons.type = ? AND firm_addons.is_active = ?",
			firmID, models.AddOnTypeCases, true).
		Select("COALESCE(SUM(plan_addons.units_included * firm_addons.quantity), 0)").
		Scan(&addOnCases)

	return baseLimit + addOnCases
}

// HasTemplatesAccess checks if firm has access to templates (via plan or add-on)
func HasTemplatesAccess(db *gorm.DB, firmID string, plan *models.Plan) bool {
	// Check if plan includes templates
	if plan.TemplatesEnabled {
		return true
	}

	// Check if purchased as add-on
	var count int64
	db.Model(&models.FirmAddOn{}).
		Joins("JOIN plan_addons ON firm_addons.add_on_id = plan_addons.id").
		Where("firm_addons.firm_id = ? AND plan_addons.type = ? AND firm_addons.is_active = ?",
			firmID, models.AddOnTypeTemplates, true).
		Count(&count)

	return count > 0
}

// CanAddUser checks if a firm can add another user
func CanAddUser(db *gorm.DB, firmID string) (*LimitCheckResult, error) {
	subscription, err := GetFirmSubscription(db, firmID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoActiveSubscription
		}
		return nil, err
	}

	if !subscription.IsActive() {
		return &LimitCheckResult{
			Allowed:        false,
			Message:        "Subscription is not active",
			TranslationKey: "subscription.errors.subscription_expired",
		}, ErrSubscriptionExpired
	}

	// Check if trial has expired
	if subscription.HasTrialExpired() {
		return &LimitCheckResult{
			Allowed:        false,
			Message:        "Trial period has expired",
			TranslationKey: "subscription.errors.trial_expired",
		}, ErrSubscriptionExpired
	}

	effectiveLimit := GetEffectiveUserLimit(db, firmID, &subscription.Plan)

	// Unlimited users
	if effectiveLimit == -1 {
		return &LimitCheckResult{Allowed: true, Limit: -1}, nil
	}

	usage, err := GetOrCalculateFirmUsage(db, firmID)
	if err != nil {
		return nil, err
	}

	result := &LimitCheckResult{
		CurrentUsage:   int64(usage.CurrentUsers),
		Limit:          int64(effectiveLimit),
		PercentageUsed: float64(usage.CurrentUsers) / float64(effectiveLimit) * 100,
	}

	if usage.CurrentUsers >= effectiveLimit {
		result.Allowed = false
		result.Message = fmt.Sprintf("User limit reached (%d/%d). Please upgrade your plan or purchase additional users.",
			usage.CurrentUsers, effectiveLimit)
		result.TranslationKey = "subscription.errors.user_limit_reached"
		result.TranslationArgs = map[string]interface{}{
			"current": usage.CurrentUsers,
			"limit":   effectiveLimit,
		}
		return result, ErrUserLimitReached
	}

	result.Allowed = true
	return result, nil
}

// CanAddClient checks if a firm can add another client (limit linked to max cases)
func CanAddClient(db *gorm.DB, firmID string) (*LimitCheckResult, error) {
	subscription, err := GetFirmSubscription(db, firmID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoActiveSubscription
		}
		return nil, err
	}

	if !subscription.IsActive() {
		return &LimitCheckResult{
			Allowed:        false,
			Message:        "Subscription is not active",
			TranslationKey: "subscription.errors.subscription_expired",
		}, ErrSubscriptionExpired
	}

	// Check if trial has expired
	if subscription.HasTrialExpired() {
		return &LimitCheckResult{
			Allowed:        false,
			Message:        "Trial period has expired",
			TranslationKey: "subscription.errors.trial_expired",
		}, ErrSubscriptionExpired
	}

	// Client limit equals Case Limit
	effectiveLimit := GetEffectiveCaseLimit(db, firmID, &subscription.Plan)

	// Unlimited clients (if cases are unlimited)
	if effectiveLimit == -1 {
		return &LimitCheckResult{Allowed: true, Limit: -1}, nil
	}

	// Count existing clients
	var currentClients int64
	if err := db.Model(&models.User{}).Where("firm_id = ? AND role = ? AND is_active = ?", firmID, "client", true).Count(&currentClients).Error; err != nil {
		return nil, err
	}

	result := &LimitCheckResult{
		CurrentUsage:   currentClients,
		Limit:          int64(effectiveLimit),
		PercentageUsed: float64(currentClients) / float64(effectiveLimit) * 100,
	}

	if currentClients >= int64(effectiveLimit) {
		result.Allowed = false
		result.Message = fmt.Sprintf("Client limit reached (%d/%d). Client limit is equal to your Case limit. Please upgrade your plan or purchase additional cases.",
			currentClients, effectiveLimit)
		result.TranslationKey = "subscription.errors.client_limit_reached"
		result.TranslationArgs = map[string]interface{}{
			"current": currentClients,
			"limit":   effectiveLimit,
		}
		return result, ErrClientLimitReached
	}

	result.Allowed = true
	return result, nil
}

// CanAddCase checks if a firm can create another case
func CanAddCase(db *gorm.DB, firmID string) (*LimitCheckResult, error) {
	subscription, err := GetFirmSubscription(db, firmID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoActiveSubscription
		}
		return nil, err
	}

	if !subscription.IsActive() {
		return &LimitCheckResult{
			Allowed:        false,
			Message:        "Subscription is not active",
			TranslationKey: "subscription.errors.subscription_expired",
		}, ErrSubscriptionExpired
	}

	// Check trial expiration
	if subscription.HasTrialExpired() {
		return &LimitCheckResult{
			Allowed:        false,
			Message:        "Trial period has expired",
			TranslationKey: "subscription.errors.trial_expired",
		}, ErrSubscriptionExpired
	}

	effectiveLimit := GetEffectiveCaseLimit(db, firmID, &subscription.Plan)

	if effectiveLimit == -1 {
		return &LimitCheckResult{Allowed: true, Limit: -1}, nil
	}

	usage, err := GetOrCalculateFirmUsage(db, firmID)
	if err != nil {
		return nil, err
	}

	result := &LimitCheckResult{
		CurrentUsage:   int64(usage.CurrentCases),
		Limit:          int64(effectiveLimit),
		PercentageUsed: float64(usage.CurrentCases) / float64(effectiveLimit) * 100,
	}

	if usage.CurrentCases >= effectiveLimit {
		result.Allowed = false
		result.Message = fmt.Sprintf("Case limit reached (%d/%d). Please upgrade your plan or purchase additional cases.",
			usage.CurrentCases, effectiveLimit)
		result.TranslationKey = "subscription.errors.case_limit_reached"
		result.TranslationArgs = map[string]interface{}{
			"current": usage.CurrentCases,
			"limit":   effectiveLimit,
		}
		return result, ErrCaseLimitReached
	}

	result.Allowed = true
	return result, nil
}

// CanUploadFile checks if a firm can upload a file of given size
func CanUploadFile(db *gorm.DB, firmID string, fileSizeBytes int64) (*LimitCheckResult, error) {
	subscription, err := GetFirmSubscription(db, firmID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoActiveSubscription
		}
		return nil, err
	}

	if !subscription.IsActive() {
		return &LimitCheckResult{Allowed: false, Message: "Subscription is not active"}, ErrSubscriptionExpired
	}

	// Check trial expiration
	if subscription.HasTrialExpired() {
		return &LimitCheckResult{Allowed: false, Message: "Trial period has expired"}, ErrSubscriptionExpired
	}

	effectiveLimit := GetEffectiveStorageLimit(db, firmID, &subscription.Plan)

	if effectiveLimit == -1 {
		return &LimitCheckResult{Allowed: true, Limit: -1}, nil
	}

	usage, err := GetOrCalculateFirmUsage(db, firmID)
	if err != nil {
		return nil, err
	}

	projectedUsage := usage.CurrentStorageBytes + fileSizeBytes

	result := &LimitCheckResult{
		CurrentUsage:   usage.CurrentStorageBytes,
		Limit:          effectiveLimit,
		PercentageUsed: float64(projectedUsage) / float64(effectiveLimit) * 100,
	}

	if projectedUsage > effectiveLimit {
		result.Allowed = false
		result.Message = fmt.Sprintf("Storage limit would be exceeded. Current: %s, Limit: %s",
			models.FormatBytes(usage.CurrentStorageBytes), models.FormatBytes(effectiveLimit))
		result.TranslationKey = "subscription.errors.storage_limit_reached"
		result.TranslationArgs = map[string]interface{}{
			"current": models.FormatBytes(usage.CurrentStorageBytes),
			"limit":   models.FormatBytes(effectiveLimit),
		}
		return result, ErrStorageLimitReached
	}

	result.Allowed = true
	return result, nil
}

// CanAccessTemplates checks if templates feature is enabled for the firm
func CanAccessTemplates(db *gorm.DB, firmID string) (bool, error) {
	subscription, err := GetFirmSubscription(db, firmID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, ErrNoActiveSubscription
		}
		return false, err
	}

	if !subscription.IsActive() {
		return false, ErrSubscriptionExpired
	}

	// Check trial expiration
	if subscription.HasTrialExpired() {
		return false, ErrSubscriptionExpired
	}

	return HasTemplatesAccess(db, firmID, &subscription.Plan), nil
}

// CreateTrialSubscription creates a trial subscription for a new firm
func CreateTrialSubscription(db *gorm.DB, firmID string) error {
	// Get default trial plan
	var trialPlan models.Plan
	if err := db.Where("is_trial_plan = ? AND is_active = ?", true, true).First(&trialPlan).Error; err != nil {
		return fmt.Errorf("no active trial plan found: %w", err)
	}

	trialEndDate := time.Now().AddDate(0, 0, trialPlan.TrialDays)

	subscription := &models.FirmSubscription{
		FirmID:      firmID,
		PlanID:      trialPlan.ID,
		Status:      models.SubscriptionStatusTrialing,
		StartedAt:   time.Now(),
		TrialEndsAt: &trialEndDate,
	}

	return db.Create(subscription).Error
}

// UpdateFirmUsageAfterUserChange updates usage cache after user addition/removal
func UpdateFirmUsageAfterUserChange(db *gorm.DB, firmID string, delta int) error {
	return db.Model(&models.FirmUsage{}).
		Where("firm_id = ?", firmID).
		UpdateColumns(map[string]interface{}{
			"current_users":      gorm.Expr("current_users + ?", delta),
			"last_calculated_at": time.Now(),
		}).Error
}

// UpdateFirmUsageAfterStorageChange updates usage cache after file upload/delete
func UpdateFirmUsageAfterStorageChange(db *gorm.DB, firmID string, deltaBytes int64) error {
	return db.Model(&models.FirmUsage{}).
		Where("firm_id = ?", firmID).
		UpdateColumns(map[string]interface{}{
			"current_storage_bytes": gorm.Expr("current_storage_bytes + ?", deltaBytes),
			"last_calculated_at":    time.Now(),
		}).Error
}

// UpdateFirmUsageAfterCaseChange updates usage cache after case creation/deletion
func UpdateFirmUsageAfterCaseChange(db *gorm.DB, firmID string, delta int) error {
	return db.Model(&models.FirmUsage{}).
		Where("firm_id = ?", firmID).
		UpdateColumns(map[string]interface{}{
			"current_cases":      gorm.Expr("current_cases + ?", delta),
			"last_calculated_at": time.Now(),
		}).Error
}

// GetOrCalculateFirmUsage gets cached usage or recalculates if stale
func GetOrCalculateFirmUsage(db *gorm.DB, firmID string) (*models.FirmUsage, error) {
	var usage models.FirmUsage
	err := db.Where("firm_id = ?", firmID).First(&usage).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Calculate and create
		return RecalculateFirmUsage(db, firmID)
	}

	if err != nil {
		return nil, err
	}

	// Check if cache is stale (older than 1 hour)
	if usage.IsStale() {
		return RecalculateFirmUsage(db, firmID)
	}

	return &usage, nil
}

// RecalculateFirmUsage calculates fresh usage from source tables
func RecalculateFirmUsage(db *gorm.DB, firmID string) (*models.FirmUsage, error) {
	// Count users (excluding clients and superadmins - only count billable users)
	var userCount int64
	db.Model(&models.User{}).
		Where("firm_id = ? AND role IN (?, ?, ?) AND is_active = ?",
			firmID, "admin", "lawyer", "staff", true).
		Count(&userCount)

	// Sum storage from documents
	var storageBytes int64
	db.Model(&models.CaseDocument{}).
		Where("firm_id = ?", firmID).
		Select("COALESCE(SUM(file_size), 0)").
		Scan(&storageBytes)

	// Count cases (non-deleted)
	var caseCount int64
	db.Model(&models.Case{}).
		Where("firm_id = ? AND is_deleted = ?", firmID, false).
		Count(&caseCount)

	usage := &models.FirmUsage{
		FirmID:              firmID,
		CurrentUsers:        int(userCount),
		CurrentStorageBytes: storageBytes,
		CurrentCases:        int(caseCount),
		LastCalculatedAt:    time.Now(),
	}

	// Upsert - update if exists, create if not
	var existingUsage models.FirmUsage
	err := db.Where("firm_id = ?", firmID).First(&existingUsage).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Create new
		if err := db.Create(usage).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		// Update existing
		existingUsage.CurrentUsers = usage.CurrentUsers
		existingUsage.CurrentStorageBytes = usage.CurrentStorageBytes
		existingUsage.CurrentCases = usage.CurrentCases
		existingUsage.LastCalculatedAt = usage.LastCalculatedAt
		if err := db.Save(&existingUsage).Error; err != nil {
			return nil, err
		}
		usage = &existingUsage
	}

	return usage, nil
}

// ChangeFirmPlan changes a firm's subscription to a different plan
func ChangeFirmPlan(db *gorm.DB, firmID string, newPlanID string, changedByUserID *string) error {
	subscription, err := GetFirmSubscription(db, firmID)
	if err != nil {
		return err
	}

	now := time.Now()
	oldPlanID := subscription.PlanID
	subscription.PreviousPlanID = &oldPlanID
	subscription.PlanID = newPlanID
	subscription.LastPlanChangeAt = &now
	subscription.ChangedByUserID = changedByUserID

	// If changing from trial to paid, update status
	if subscription.IsTrialing() {
		subscription.Status = models.SubscriptionStatusActive
		subscription.TrialEndsAt = nil
		subscription.CurrentPeriodStart = &now
		periodEnd := now.AddDate(0, 1, 0) // 1 month
		subscription.CurrentPeriodEnd = &periodEnd
	}

	return db.Omit("Plan", "Firm").Save(subscription).Error
}

// ExtendTrial extends the trial period for a firm
func ExtendTrial(db *gorm.DB, firmID string, additionalDays int) error {
	subscription, err := GetFirmSubscription(db, firmID)
	if err != nil {
		return err
	}

	if !subscription.IsTrialing() {
		return errors.New("firm is not on trial")
	}

	if subscription.TrialEndsAt == nil {
		newEnd := time.Now().AddDate(0, 0, additionalDays)
		subscription.TrialEndsAt = &newEnd
	} else {
		newEnd := subscription.TrialEndsAt.AddDate(0, 0, additionalDays)
		subscription.TrialEndsAt = &newEnd
	}

	return db.Save(subscription).Error
}
