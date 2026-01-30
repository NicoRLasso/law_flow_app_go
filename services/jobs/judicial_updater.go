package jobs

import (
	"errors"
	"fmt"
	"law_flow_app_go/models"
	"law_flow_app_go/services/judicial"
	"log"
	"reflect"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// StartJudicialUpdateJob starts the background job to update judicial processes
// It runs once immediately, then every 24 hours
func StartScheduler(database *gorm.DB) {

	loc, _ := time.LoadLocation("America/Bogota")
	c := cron.New(cron.WithLocation(loc))

	_, err := c.AddFunc("0 0 * * *", func() {
		log.Println("[CRON] Ejecutando UpdateAllJudicialProcesses a medianoche...")
		UpdateAllJudicialProcesses(database)
	})

	if err != nil {
		log.Fatalf("[CRON] Error al programar la tarea: %v", err)
	}

	c.Start()
	log.Println("[CRON] Planificador de tareas iniciado correctamente.")
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
	isNewTracking := false
	err = database.Where("case_id = ?", c.ID).First(&jp).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("[JOB] Error checking existing record: %v", err)
		return err
	}

	// 2. If not found, search API to get Process ID
	if errors.Is(err, gorm.ErrRecordNotFound) {
		isNewTracking = true
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

	importedCount := 0

	// Insert/Update actions
	for _, action := range actions {
		var existingAction models.JudicialProcessAction
		result := database.Where("judicial_process_id = ? AND external_id = ?", jp.ID, action.ExternalID).First(&existingAction)

		if result.Error == nil {
			// UPDATE LOGIC: Check changes BEFORE updating

			// Construct target metadata for comparison
			targetMetadata := make(models.JSONMap)
			// Start with existing metadata keys if needed, or build fresh?
			// To ensure clean updates from API, we usually rebuild or merge carefully.
			// Let's copy existing first to preserve non-API keys if any.
			if existingAction.Metadata != nil {
				for k, v := range existingAction.Metadata {
					targetMetadata[k] = v
				}
			}
			// Update with API values
			targetMetadata["registration_date"] = action.RegistrationDate
			targetMetadata["initial_date"] = action.InitialDate
			targetMetadata["final_date"] = action.FinalDate
			if action.Metadata != nil {
				for k, v := range action.Metadata {
					targetMetadata[k] = v
				}
			}

			// Detect changes
			hasChanges := false
			if existingAction.Type != action.Type {
				hasChanges = true
			}
			if existingAction.Annotation != action.Annotation {
				hasChanges = true
			}
			if existingAction.HasDocuments != action.HasDocuments {
				hasChanges = true
			}
			if !existingAction.ActionDate.Equal(action.ActionDate) {
				hasChanges = true
			}
			if !reflect.DeepEqual(existingAction.Metadata, targetMetadata) {
				hasChanges = true
			}

			if hasChanges {
				// Update fields
				existingAction.Type = action.Type
				existingAction.Annotation = action.Annotation
				existingAction.HasDocuments = action.HasDocuments
				existingAction.ActionDate = action.ActionDate
				existingAction.Metadata = targetMetadata

				if err := database.Save(&existingAction).Error; err != nil {
					log.Printf("[JOB] Failed to update action %s: %v", action.ExternalID, err)
				} else {
					// Notify update if not initial sync
					if !isNewTracking {
						createJudicialUpdateNotification(database, c, existingAction, fmt.Sprintf("Actuación actualizada: %s", action.Type))
					}
				}
			}
		} else if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// CREATE: New action
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

			if err := database.Create(&newAction).Error; err != nil {
				log.Printf("[JOB] Failed to create action %s: %v", action.ExternalID, err)
			} else {
				importedCount++
				// Create notification ONLY if this is NOT the initial tracking setup
				if !isNewTracking {
					createJudicialUpdateNotification(database, c, newAction, fmt.Sprintf("Nueva actuación: %s", action.Type))
				}
			}
		} else {
			log.Printf("[JOB] Error checking action existence: %v", result.Error)
		}
	}

	// If this was the first sync and we imported data, send a SUMMARY notification
	if isNewTracking && importedCount > 0 {
		summaryNotification := models.Notification{
			FirmID:    c.FirmID,
			CaseID:    &c.ID,
			Type:      models.NotificationTypeSystem, // Or a specific type for Linkage
			Title:     "Proceso Vinculado Exitosamente",
			Message:   fmt.Sprintf("Se ha conectado con la Rama Judicial y se han importado %d actuaciones históricas.", importedCount),
			LinkURL:   fmt.Sprintf("/cases/%s", c.ID),
			CreatedAt: time.Now(),
		}
		if c.AssignedToID != nil {
			summaryNotification.UserID = c.AssignedToID
		}
		database.Create(&summaryNotification)
	}

	return nil
}

func createJudicialUpdateNotification(db *gorm.DB, c models.Case, action models.JudicialProcessAction, title string) {
	notification := models.Notification{
		FirmID:                  c.FirmID,
		CaseID:                  &c.ID,
		JudicialProcessActionID: &action.ID,
		Type:                    models.NotificationTypeJudicialUpdate,
		Title:                   title,
		Message:                 action.Annotation,
		LinkURL:                 fmt.Sprintf("/cases/%s", c.ID),
	}

	// Target assigned lawyer if exists
	if c.AssignedToID != nil {
		notification.UserID = c.AssignedToID
	}

	db.Create(&notification)
}
