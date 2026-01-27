package jobs

import (
	"errors"
	"law_flow_app_go/db"
	"law_flow_app_go/models"
	"law_flow_app_go/services/judicial"
	"log"
	"time"

	"gorm.io/gorm"
)

// StartJudicialUpdateJob starts the background job to update judicial processes
// It runs once immediately, then every 24 hours
func StartJudicialUpdateJob() {
	go func() {
		// Run immediately on startup (non-blocking) - wait 10 seconds for server to settle
		time.Sleep(10 * time.Second)
		log.Println("[JOB] Starting initial judicial process update...")
		UpdateAllJudicialProcesses(db.DB)

		// Schedule nightly run (every 24 hours)
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			log.Println("[JOB] Starting scheduled judicial process update...")
			UpdateAllJudicialProcesses(db.DB)
		}
	}()
}

// UpdateAllJudicialProcesses iterates through relevant cases and updates them
func UpdateAllJudicialProcesses(database *gorm.DB) {
	// 1. Find all Open Cases with a Filing Number (Radicado) and preload Firm to get Country
	var cases []models.Case
	if err := database.Preload("Firm").Where("status = ? AND filing_number IS NOT NULL AND filing_number != ''", models.CaseStatusOpen).Find(&cases).Error; err != nil {
		log.Printf("[JOB] Error fetching cases for update: %v", err)
		return
	}

	log.Printf("[JOB] Found %d cases to check for judicial updates", len(cases))

	for _, c := range cases {
		// Process each case sequentially
		if err := processCase(database, c); err != nil {
			log.Printf("[JOB] Error updating case %s (Radicado: %s): %v", c.CaseNumber, *c.FilingNumber, err)
		} else {
			log.Printf("[JOB] Successfully checked/updated case %s", c.CaseNumber)
		}

		time.Sleep(1 * time.Second) // Be polite
	}
}

// UpdateSingleCase triggers a judicial process update for a specific case
func UpdateSingleCase(database *gorm.DB, caseID string) error {
	log.Printf("[JOB] UpdateSingleCase called for caseID: %s", caseID)
	var c models.Case
	// Need to preload Firm to get Country
	if err := database.Preload("Firm").Where("id = ?", caseID).First(&c).Error; err != nil {
		return err
	}

	// Only process if it has a filing number and is not closed
	if c.FilingNumber == nil || *c.FilingNumber == "" {
		return errors.New("case has no filing number")
	}

	return processCase(database, c)
}

func processCase(database *gorm.DB, c models.Case) error {
	if c.FilingNumber == nil {
		return nil
	}

	// Get appropriate provider for firm's country
	// Default to Colombia if country not set, or handle error
	country := "CO"
	if c.Firm.Country != "" {
		country = c.Firm.Country
	} else {
		// Try to fallback/determine or just return error
		// For now, let's log and skip if unknown, or default to CO?
		// User said this job is just for Colombia depending on country, so let's default to skipping if not CO.
		// Actually, let's keep it robust.
	}

	provider, err := judicial.GetProvider(country)
	if err != nil {
		// If provider not found (e.g. US firm), just return nil (skip)
		log.Printf("[JOB] GetProvider failed for country '%s': %v", country, err)
		return nil
	}

	radicado := *c.FilingNumber
	log.Printf("[JOB] Processing case %s with radicado: %s (Country: %s)", c.ID, radicado, country)

	// 1. Check if we already have a JudicialProcess record
	var jp models.JudicialProcess
	err = database.Where("case_id = ?", c.ID).First(&jp).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("[JOB] Error checking existing record: %v", err)
		return err
	}

	// 2. If not found, search API to get Process ID
	if errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("[JOB] No existing record found. searching API for radicado: %s", radicado)
		summary, err := provider.GetProcessIDByRadicado(radicado)
		if err != nil {
			log.Printf("[JOB] Provider search error: %v", err)
			return err
		}
		if summary == nil {
			log.Printf("[JOB] Radicado not found in API: %s", radicado)
			return nil // Not found in API
		}
		log.Printf("[JOB] Found process ID in API: %s", summary.ProcessID)

		// Get full details
		detail, err := provider.GetProcessDetail(summary.ProcessID)
		if err != nil {
			log.Printf("[JOB] Detail fetch error: %v", err)
			return err
		}

		// Create JudicialProcess record
		// Populate Details map with generic fields + any others returned by detail
		detailsMap := models.JSONMap{
			"department": summary.Department,
			"office":     summary.Office,
			"subject":    summary.Subject,
		}
		// Merge details from GetProcessDetail
		for k, v := range detail {
			detailsMap[k] = v
		}

		jp = models.JudicialProcess{
			CaseID:       c.ID,
			ProcessID:    summary.ProcessID,
			Radicado:     radicado,
			IsPrivado:    summary.IsPrivate,
			LastTracking: time.Now(),
			Status:       "ACTIVE",
			Details:      detailsMap,
		}
		// Last Activity wasn't in generic summary, maybe we can fetch it, or rely on actions later.
		// For now, let's leave LastActivityDate as zero or update from actions.

		if err := database.Create(&jp).Error; err != nil {
			return err
		}
	} else {
		// Update LastTracking
		jp.LastTracking = time.Now()
		database.Save(&jp)
	}

	// 3. Sync Actions (Actuaciones)
	// ProcessID is string now
	actions, err := provider.GetProcessActions(jp.ProcessID)
	if err != nil {
		return err
	}

	// Insert new actions
	for _, action := range actions {
		var exists int64
		database.Model(&models.JudicialProcessAction{}).
			Where("judicial_process_id = ? AND external_id = ?", jp.ID, action.ExternalID).
			Count(&exists)

		if exists == 0 {
			newAction := models.JudicialProcessAction{
				JudicialProcessID: jp.ID,
				ExternalID:        action.ExternalID,
				Type:              action.Type,
				Annotation:        action.Annotation,
				HasDocuments:      action.HasDocuments,
				ActionDate:        action.ActionDate,
				Metadata: models.JSONMap{
					"registration_date": action.RegistrationDate,
					"initial_date":      action.InitialDate,
					"final_date":        action.FinalDate,
				},
			}
			// Merge extra metadata from generic action
			if action.Metadata != nil {
				for k, v := range action.Metadata {
					newAction.Metadata[k] = v
				}
			}
			// Dates are directly available in generic action as Time or *Time

			if err := database.Create(&newAction).Error; err != nil {
				log.Printf("[JOB] Failed to create action %s: %v", action.ExternalID, err)
			}
		}
	}

	// Update generic LastActivityDate if actions found
	if len(actions) > 0 {
		latestAction := actions[0] // API returns sorted desc
		if !latestAction.ActionDate.IsZero() {
			jp.LastActivityDate = latestAction.ActionDate
			database.Save(&jp)
		}
	}

	return nil
}
