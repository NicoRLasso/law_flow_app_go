package services

import (
	"log"
	"strings"

	"gorm.io/gorm"
)

// DropFTSTriggers removes all FTS-related triggers to prevent migration issues
func DropFTSTriggers(db *gorm.DB) {
	triggers := []string{
		"cases_fts_insert", "cases_fts_update", "cases_fts_delete",
		"case_logs_fts_insert", "case_logs_fts_update", "case_logs_fts_delete",
		"case_parties_fts_insert", "case_parties_fts_update", "case_parties_fts_delete",
		"case_documents_fts_insert", "case_documents_fts_update", "case_documents_fts_delete",
		"services_fts_insert", "services_fts_update", "services_fts_delete",
		"service_milestones_fts_insert", "service_milestones_fts_update", "service_milestones_fts_delete",
		"service_documents_fts_insert", "service_documents_fts_update", "service_documents_fts_delete",
	}

	for _, t := range triggers {
		db.Exec("DROP TRIGGER IF EXISTS " + t)
	}
}

// InitializeFTS5 creates the FTS5 virtual table and triggers if they don't exist
func InitializeFTS5(db *gorm.DB) error {
	log.Println("Initializing FTS5 search index...")

	// Create FTS5 virtual table
	// Check for existing contentless table and drop it if found
	var sql string
	db.Raw("SELECT sql FROM sqlite_master WHERE type='table' AND name='cases_fts'").Scan(&sql)
	if strings.Contains(sql, "content=''") {
		log.Println("Migrating FTS5 table to support snippets...")
		db.Exec("DROP TABLE IF EXISTS cases_fts")
		db.Exec("DROP TABLE IF EXISTS cases_fts_mapping")
	}

	// Create FTS5 virtual table
	// Note: We removed content='' to allow snippet generation
	err := db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS cases_fts USING fts5(
			case_id UNINDEXED,
			firm_id UNINDEXED,
			case_number,
			case_title,
			case_description,
			filing_number,
			client_name,
			party_name,
			log_content,
			document_content,
			tokenize='unicode61 remove_diacritics 2'
		)
	`).Error
	if err != nil {
		return err
	}

	// Create mapping table
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS cases_fts_mapping (
			rowid INTEGER PRIMARY KEY AUTOINCREMENT,
			case_id TEXT NOT NULL UNIQUE,
			firm_id TEXT NOT NULL,
			last_updated DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		return err
	}

	// Create indices for mapping table
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_fts_mapping_firm ON cases_fts_mapping(firm_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_fts_mapping_case ON cases_fts_mapping(case_id)`)

	// Create FTS5 virtual table for services
	err = db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS services_fts USING fts5(
			service_id UNINDEXED,
			firm_id UNINDEXED,
			service_number,
			service_title,
			service_description,
			service_objective,
			client_name,
			milestone_content,
			document_content,
			tokenize='unicode61 remove_diacritics 2'
		)
	`).Error
	if err != nil {
		return err
	}

	// Create mapping table for services
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS services_fts_mapping (
			rowid INTEGER PRIMARY KEY AUTOINCREMENT,
			service_id TEXT NOT NULL UNIQUE,
			firm_id TEXT NOT NULL,
			last_updated DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		return err
	}

	// Create indices for services mapping table
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_services_fts_mapping_firm ON services_fts_mapping(firm_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_services_fts_mapping_service ON services_fts_mapping(service_id)`)

	// Create triggers for cases table
	if err := createCasesTriggers(db); err != nil {
		log.Printf("[WARNING] Failed to create cases triggers: %v", err)
	}

	// Create triggers for case_logs table
	if err := createCaseLogsTriggers(db); err != nil {
		log.Printf("[WARNING] Failed to create case_logs triggers: %v", err)
	}

	// Create triggers for case_parties table
	if err := createCasePartiesTriggers(db); err != nil {
		log.Printf("[WARNING] Failed to create case_parties triggers: %v", err)
	}

	// Create triggers for case_documents table
	if err := createCaseDocumentsTriggers(db); err != nil {
		log.Printf("[WARNING] Failed to create case_documents triggers: %v", err)
	}

	// Create triggers for legal_services table
	if err := createServicesTriggers(db); err != nil {
		log.Printf("[WARNING] Failed to create services triggers: %v", err)
	}

	// Create triggers for service_milestones table
	if err := createServiceMilestonesTriggers(db); err != nil {
		log.Printf("[WARNING] Failed to create service_milestones triggers: %v", err)
	}

	// Create triggers for service_documents table
	if err := createServiceDocumentsTriggers(db); err != nil {
		log.Printf("[WARNING] Failed to create service_documents triggers: %v", err)
	}

	log.Println("FTS5 search index initialized")
	return nil
}

func createCasesTriggers(db *gorm.DB) error {
	// Drop existing triggers first (in case of schema changes)
	db.Exec(`DROP TRIGGER IF EXISTS cases_fts_insert`)
	db.Exec(`DROP TRIGGER IF EXISTS cases_fts_update`)
	db.Exec(`DROP TRIGGER IF EXISTS cases_fts_delete`)

	// Trigger: INSERT on cases
	err := db.Exec(`
		CREATE TRIGGER IF NOT EXISTS cases_fts_insert AFTER INSERT ON cases
		BEGIN
			INSERT OR IGNORE INTO cases_fts_mapping (case_id, firm_id)
			VALUES (NEW.id, NEW.firm_id);

			INSERT INTO cases_fts (
				rowid, case_id, firm_id, case_number, case_title,
				case_description, filing_number, client_name, party_name, log_content, document_content
			)
			SELECT
				m.rowid,
				NEW.id,
				NEW.firm_id,
				NEW.case_number,
				COALESCE(NEW.title, ''),
				COALESCE(NEW.description, ''),
				COALESCE(NEW.filing_number, ''),
				COALESCE((SELECT name FROM users WHERE id = NEW.client_id), ''),
				'',
				'',
				''
			FROM cases_fts_mapping m WHERE m.case_id = NEW.id;
		END
	`).Error
	if err != nil {
		return err
	}

	// Trigger: UPDATE on cases
	err = db.Exec(`
		CREATE TRIGGER IF NOT EXISTS cases_fts_update AFTER UPDATE ON cases
		WHEN OLD.title IS NOT NEW.title
		   OR OLD.description IS NOT NEW.description
		   OR OLD.case_number IS NOT NEW.case_number
		   OR OLD.filing_number IS NOT NEW.filing_number
		   OR OLD.client_id IS NOT NEW.client_id
		   OR (OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL)
		BEGIN
			DELETE FROM cases_fts WHERE rowid = (
				SELECT rowid FROM cases_fts_mapping WHERE case_id = OLD.id
			);

			-- Only re-insert if NOT deleted
			INSERT INTO cases_fts (
				rowid, case_id, firm_id, case_number, case_title,
				case_description, filing_number, client_name, party_name, log_content, document_content
			)
			SELECT
				m.rowid,
				NEW.id,
				NEW.firm_id,
				NEW.case_number,
				COALESCE(NEW.title, ''),
				COALESCE(NEW.description, ''),
				COALESCE(NEW.filing_number, ''),
				COALESCE((SELECT name FROM users WHERE id = NEW.client_id), ''),
				COALESCE((SELECT name FROM case_parties WHERE case_id = NEW.id LIMIT 1), ''),
				COALESCE((SELECT GROUP_CONCAT(COALESCE(title, '') || ' ' || COALESCE(content, ''), ' ') FROM case_logs WHERE case_id = NEW.id AND deleted_at IS NULL), ''),
				COALESCE((SELECT GROUP_CONCAT(COALESCE(description, '') || ' ' || file_original_name, ' ') FROM case_documents WHERE case_id = NEW.id AND deleted_at IS NULL), '')
			FROM cases_fts_mapping m
			WHERE m.case_id = NEW.id
			AND NEW.deleted_at IS NULL;

			-- If it was soft-deleted, we might want to also clean up mapping or keep it?
			-- RebuildFTSIndex excludes deleted_at IS NULL makers.
			-- For triggers, if it's soft-deleted, we already deleted from cases_fts.

			UPDATE cases_fts_mapping SET last_updated = CURRENT_TIMESTAMP WHERE case_id = NEW.id;
		END
	`).Error
	if err != nil {
		return err
	}

	// Trigger: DELETE on cases
	err = db.Exec(`
		CREATE TRIGGER IF NOT EXISTS cases_fts_delete AFTER DELETE ON cases
		BEGIN
			DELETE FROM cases_fts WHERE rowid = (
				SELECT rowid FROM cases_fts_mapping WHERE case_id = OLD.id
			);
			DELETE FROM cases_fts_mapping WHERE case_id = OLD.id;
		END
	`).Error

	return err
}

func createCaseLogsTriggers(db *gorm.DB) error {
	db.Exec(`DROP TRIGGER IF EXISTS case_logs_fts_insert`)
	db.Exec(`DROP TRIGGER IF EXISTS case_logs_fts_update`)
	db.Exec(`DROP TRIGGER IF EXISTS case_logs_fts_delete`)

	// Trigger: INSERT on case_logs
	err := db.Exec(`
		CREATE TRIGGER IF NOT EXISTS case_logs_fts_insert AFTER INSERT ON case_logs
		BEGIN
			UPDATE cases_fts SET log_content = (
				SELECT COALESCE(GROUP_CONCAT(COALESCE(title, '') || ' ' || COALESCE(content, ''), ' '), '')
				FROM case_logs
				WHERE case_id = NEW.case_id AND deleted_at IS NULL
			)
			WHERE rowid = (SELECT rowid FROM cases_fts_mapping WHERE case_id = NEW.case_id);
		END
	`).Error
	if err != nil {
		return err
	}

	// Trigger: UPDATE on case_logs
	err = db.Exec(`
		CREATE TRIGGER IF NOT EXISTS case_logs_fts_update AFTER UPDATE ON case_logs
		BEGIN
			UPDATE cases_fts SET log_content = (
				SELECT COALESCE(GROUP_CONCAT(COALESCE(title, '') || ' ' || COALESCE(content, ''), ' '), '')
				FROM case_logs
				WHERE case_id = NEW.case_id AND deleted_at IS NULL
			)
			WHERE rowid = (SELECT rowid FROM cases_fts_mapping WHERE case_id = NEW.case_id);
		END
	`).Error
	if err != nil {
		return err
	}

	// Trigger: DELETE on case_logs
	err = db.Exec(`
		CREATE TRIGGER IF NOT EXISTS case_logs_fts_delete AFTER DELETE ON case_logs
		BEGIN
			UPDATE cases_fts SET log_content = (
				SELECT COALESCE(GROUP_CONCAT(COALESCE(title, '') || ' ' || COALESCE(content, ''), ' '), '')
				FROM case_logs
				WHERE case_id = OLD.case_id AND deleted_at IS NULL
			)
			WHERE rowid = (SELECT rowid FROM cases_fts_mapping WHERE case_id = OLD.case_id);
		END
	`).Error

	return err
}

func createCasePartiesTriggers(db *gorm.DB) error {
	db.Exec(`DROP TRIGGER IF EXISTS case_parties_fts_insert`)
	db.Exec(`DROP TRIGGER IF EXISTS case_parties_fts_update`)
	db.Exec(`DROP TRIGGER IF EXISTS case_parties_fts_delete`)

	err := db.Exec(`
		CREATE TRIGGER IF NOT EXISTS case_parties_fts_insert AFTER INSERT ON case_parties
		BEGIN
			UPDATE cases_fts SET party_name = NEW.name
			WHERE rowid = (SELECT rowid FROM cases_fts_mapping WHERE case_id = NEW.case_id);
		END
	`).Error
	if err != nil {
		return err
	}

	err = db.Exec(`
		CREATE TRIGGER IF NOT EXISTS case_parties_fts_update AFTER UPDATE ON case_parties
		BEGIN
			UPDATE cases_fts SET party_name = NEW.name
			WHERE rowid = (SELECT rowid FROM cases_fts_mapping WHERE case_id = NEW.case_id);
		END
	`).Error
	if err != nil {
		return err
	}

	err = db.Exec(`
		CREATE TRIGGER IF NOT EXISTS case_parties_fts_delete AFTER DELETE ON case_parties
		BEGIN
			UPDATE cases_fts SET party_name = ''
			WHERE rowid = (SELECT rowid FROM cases_fts_mapping WHERE case_id = OLD.case_id);
		END
	`).Error

	return err
}

func createCaseDocumentsTriggers(db *gorm.DB) error {
	db.Exec(`DROP TRIGGER IF EXISTS case_documents_fts_insert`)
	db.Exec(`DROP TRIGGER IF EXISTS case_documents_fts_update`)
	db.Exec(`DROP TRIGGER IF EXISTS case_documents_fts_delete`)

	err := db.Exec(`
		CREATE TRIGGER IF NOT EXISTS case_documents_fts_insert AFTER INSERT ON case_documents
		WHEN NEW.case_id IS NOT NULL
		BEGIN
			UPDATE cases_fts SET document_content = (
				SELECT COALESCE(GROUP_CONCAT(COALESCE(description, '') || ' ' || file_original_name, ' '), '')
				FROM case_documents
				WHERE case_id = NEW.case_id AND deleted_at IS NULL
			)
			WHERE rowid = (SELECT rowid FROM cases_fts_mapping WHERE case_id = NEW.case_id);
		END
	`).Error
	if err != nil {
		return err
	}

	err = db.Exec(`
		CREATE TRIGGER IF NOT EXISTS case_documents_fts_update AFTER UPDATE ON case_documents
		WHEN NEW.case_id IS NOT NULL
		BEGIN
			UPDATE cases_fts SET document_content = (
				SELECT COALESCE(GROUP_CONCAT(COALESCE(description, '') || ' ' || file_original_name, ' '), '')
				FROM case_documents
				WHERE case_id = NEW.case_id AND deleted_at IS NULL
			)
			WHERE rowid = (SELECT rowid FROM cases_fts_mapping WHERE case_id = NEW.case_id);
		END
	`).Error
	if err != nil {
		return err
	}

	err = db.Exec(`
		CREATE TRIGGER IF NOT EXISTS case_documents_fts_delete AFTER DELETE ON case_documents
		WHEN OLD.case_id IS NOT NULL
		BEGIN
			UPDATE cases_fts SET document_content = (
				SELECT COALESCE(GROUP_CONCAT(COALESCE(description, '') || ' ' || file_original_name, ' '), '')
				FROM case_documents
				WHERE case_id = OLD.case_id AND deleted_at IS NULL
			)
			WHERE rowid = (SELECT rowid FROM cases_fts_mapping WHERE case_id = OLD.case_id);
		END
	`).Error

	return err
}

// RebuildFTSIndex rebuilds the FTS5 index from scratch
func RebuildFTSIndex(db *gorm.DB) error {
	log.Println("Rebuilding FTS5 index...")

	// Clear existing data for cases
	if err := db.Exec(`DELETE FROM cases_fts`).Error; err != nil {
		return err
	}
	if err := db.Exec(`DELETE FROM cases_fts_mapping`).Error; err != nil {
		return err
	}

	// Clear existing data for services
	if err := db.Exec(`DELETE FROM services_fts`).Error; err != nil {
		return err
	}
	if err := db.Exec(`DELETE FROM services_fts_mapping`).Error; err != nil {
		return err
	}

	// Insert mapping for all cases
	err := db.Exec(`
		INSERT INTO cases_fts_mapping (case_id, firm_id)
		SELECT id, firm_id FROM cases WHERE deleted_at IS NULL
	`).Error
	if err != nil {
		return err
	}

	// Populate FTS index
	err = db.Exec(`
		INSERT INTO cases_fts (
			rowid, case_id, firm_id, case_number, case_title,
			case_description, filing_number, client_name, party_name, log_content, document_content
		)
		SELECT
			m.rowid,
			c.id,
			c.firm_id,
			c.case_number,
			COALESCE(c.title, ''),
			COALESCE(c.description, ''),
			COALESCE(c.filing_number, ''),
			COALESCE(u.name, ''),
			COALESCE(cp.name, ''),
			COALESCE((
				SELECT GROUP_CONCAT(COALESCE(cl.title, '') || ' ' || COALESCE(cl.content, ''), ' ')
				FROM case_logs cl
				WHERE cl.case_id = c.id AND cl.deleted_at IS NULL
			), ''),
			COALESCE((
				SELECT GROUP_CONCAT(COALESCE(cd.description, '') || ' ' || cd.file_original_name, ' ')
				FROM case_documents cd
				WHERE cd.case_id = c.id AND cd.deleted_at IS NULL
			), '')
		FROM cases c
		INNER JOIN cases_fts_mapping m ON m.case_id = c.id
		LEFT JOIN users u ON c.client_id = u.id
		LEFT JOIN case_parties cp ON cp.case_id = c.id
		WHERE c.deleted_at IS NULL
	`).Error
	if err != nil {
		return err
	}

	// Get count for logging
	// Rebuild services index
	var services []struct {
		ID            string
		FirmID        string
		ServiceNumber string
		Title         string
		Description   string
		Objective     string
		ClientID      string
		ClientName    string
	}

	db.Table("legal_services").
		Select("legal_services.id, legal_services.firm_id, legal_services.service_number, legal_services.title, legal_services.description, legal_services.objective, legal_services.client_id, users.name as client_name").
		Joins("LEFT JOIN users ON users.id = legal_services.client_id").
		Scan(&services)

	for _, service := range services {
		// Insert into mapping
		db.Exec(`INSERT OR IGNORE INTO services_fts_mapping (service_id, firm_id) VALUES (?, ?)`,
			service.ID, service.FirmID)

		var rowid int64
		db.Raw(`SELECT rowid FROM services_fts_mapping WHERE service_id = ?`, service.ID).Scan(&rowid)

		// Get milestones
		var milestones string
		db.Raw(`SELECT GROUP_CONCAT(title || ' ' || COALESCE(description, ''), ' ') FROM service_milestones WHERE service_id = ?`, service.ID).Scan(&milestones)

		// Get documents
		var documents string
		db.Raw(`SELECT GROUP_CONCAT(file_original_name || ' ' || COALESCE(description, ''), ' ') FROM service_documents WHERE service_id = ?`, service.ID).Scan(&documents)

		// Insert into FTS
		db.Exec(`
			INSERT INTO services_fts (rowid, service_id, firm_id, service_number, service_title, service_description, service_objective, client_name, milestone_content, document_content)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			rowid, service.ID, service.FirmID, service.ServiceNumber, service.Title,
			service.Description, service.Objective, service.ClientName,
			milestones, documents)
	}

	var casesCount, servicesCount int64
	db.Table("cases_fts_mapping").Count(&casesCount)
	db.Table("services_fts_mapping").Count(&servicesCount)
	log.Printf("FTS5 index rebuilt successfully with %d cases and %d services", casesCount, servicesCount)

	return nil
}

func createServicesTriggers(db *gorm.DB) error {
	// Drop existing triggers first
	db.Exec(`DROP TRIGGER IF EXISTS services_fts_insert`)
	db.Exec(`DROP TRIGGER IF EXISTS services_fts_update`)
	db.Exec(`DROP TRIGGER IF EXISTS services_fts_delete`)

	// Trigger: INSERT on legal_services
	err := db.Exec(`
		CREATE TRIGGER IF NOT EXISTS services_fts_insert AFTER INSERT ON legal_services
		BEGIN
			INSERT OR IGNORE INTO services_fts_mapping (service_id, firm_id)
			VALUES (NEW.id, NEW.firm_id);

			INSERT INTO services_fts (
				rowid, service_id, firm_id, service_number, service_title,
				service_description, service_objective, client_name,
				milestone_content, document_content
			)
			SELECT
				(SELECT rowid FROM services_fts_mapping WHERE service_id = NEW.id),
				NEW.id,
				NEW.firm_id,
				NEW.service_number,
				NEW.title,
				NEW.description,
				COALESCE(NEW.objective, ''),
				COALESCE((SELECT name FROM users WHERE id = NEW.client_id), ''),
				'',
				'',
				''
			WHERE NEW.deleted_at IS NULL;
		END
	`).Error
	if err != nil {
		return err
	}

	// Trigger: UPDATE on legal_services
	err = db.Exec(`
		CREATE TRIGGER IF NOT EXISTS services_fts_update AFTER UPDATE ON legal_services
		BEGIN
			DELETE FROM services_fts WHERE rowid = (SELECT rowid FROM services_fts_mapping WHERE service_id = NEW.id);

			INSERT INTO services_fts (
				rowid, service_id, firm_id, service_number, service_title,
				service_description, service_objective, client_name,
				milestone_content, document_content
			)
			SELECT
				(SELECT rowid FROM services_fts_mapping WHERE service_id = NEW.id),
				NEW.id,
				NEW.firm_id,
				NEW.service_number,
				NEW.title,
				NEW.description,
				COALESCE(NEW.objective, ''),
				COALESCE((SELECT name FROM users WHERE id = NEW.client_id), ''),
				COALESCE((SELECT GROUP_CONCAT(title || ' ' || COALESCE(description, ''), ' ') FROM service_milestones WHERE service_id = NEW.id), ''),
				COALESCE((SELECT GROUP_CONCAT(file_original_name || ' ' || COALESCE(description, ''), ' ') FROM service_documents WHERE service_id = NEW.id AND deleted_at IS NULL), '')
			FROM services_fts_mapping m
			WHERE m.service_id = NEW.id
			AND NEW.deleted_at IS NULL;

			UPDATE services_fts_mapping SET last_updated = CURRENT_TIMESTAMP WHERE service_id = NEW.id;
		END
	`).Error
	if err != nil {
		return err
	}

	// Trigger: DELETE on legal_services
	err = db.Exec(`
		CREATE TRIGGER IF NOT EXISTS services_fts_delete AFTER DELETE ON legal_services
		BEGIN
			DELETE FROM services_fts WHERE rowid = (SELECT rowid FROM services_fts_mapping WHERE service_id = OLD.id);
			DELETE FROM services_fts_mapping WHERE service_id = OLD.id;
		END
	`).Error

	return err
}

func createServiceMilestonesTriggers(db *gorm.DB) error {
	db.Exec(`DROP TRIGGER IF EXISTS service_milestones_fts_insert`)
	db.Exec(`DROP TRIGGER IF EXISTS service_milestones_fts_update`)
	db.Exec(`DROP TRIGGER IF EXISTS service_milestones_fts_delete`)

	err := db.Exec(`
		CREATE TRIGGER IF NOT EXISTS service_milestones_fts_insert AFTER INSERT ON service_milestones
		BEGIN
			UPDATE services_fts SET milestone_content = (
				SELECT GROUP_CONCAT(title || ' ' || COALESCE(description, ''), ' ')
				FROM service_milestones WHERE service_id = NEW.service_id
			)
			WHERE rowid = (SELECT rowid FROM services_fts_mapping WHERE service_id = NEW.service_id);
		END
	`).Error
	if err != nil {
		return err
	}

	err = db.Exec(`
		CREATE TRIGGER IF NOT EXISTS service_milestones_fts_update AFTER UPDATE ON service_milestones
		BEGIN
			UPDATE services_fts SET milestone_content = (
				SELECT GROUP_CONCAT(title || ' ' || COALESCE(description, ''), ' ')
				FROM service_milestones WHERE service_id = NEW.service_id
			)
			WHERE rowid = (SELECT rowid FROM services_fts_mapping WHERE service_id = NEW.service_id);
		END
	`).Error
	if err != nil {
		return err
	}

	err = db.Exec(`
		CREATE TRIGGER IF NOT EXISTS service_milestones_fts_delete AFTER DELETE ON service_milestones
		BEGIN
			UPDATE services_fts SET milestone_content = (
				SELECT GROUP_CONCAT(title || ' ' || COALESCE(description, ''), ' ')
				FROM service_milestones WHERE service_id = OLD.service_id
			)
			WHERE rowid = (SELECT rowid FROM services_fts_mapping WHERE service_id = OLD.service_id);
		END
	`).Error

	return err
}

func createServiceDocumentsTriggers(db *gorm.DB) error {
	db.Exec(`DROP TRIGGER IF EXISTS service_documents_fts_insert`)
	db.Exec(`DROP TRIGGER IF EXISTS service_documents_fts_update`)
	db.Exec(`DROP TRIGGER IF EXISTS service_documents_fts_delete`)

	err := db.Exec(`
		CREATE TRIGGER IF NOT EXISTS service_documents_fts_insert AFTER INSERT ON service_documents
		BEGIN
			UPDATE services_fts SET document_content = (
				SELECT GROUP_CONCAT(file_original_name || ' ' || COALESCE(description, ''), ' ')
				FROM service_documents WHERE service_id = NEW.service_id AND deleted_at IS NULL
			)
			WHERE rowid = (SELECT rowid FROM services_fts_mapping WHERE service_id = NEW.service_id);
		END
	`).Error
	if err != nil {
		return err
	}

	err = db.Exec(`
		CREATE TRIGGER IF NOT EXISTS service_documents_fts_update AFTER UPDATE ON service_documents
		BEGIN
			UPDATE services_fts SET document_content = (
				SELECT GROUP_CONCAT(file_original_name || ' ' || COALESCE(description, ''), ' ')
				FROM service_documents WHERE service_id = NEW.service_id AND deleted_at IS NULL
			)
			WHERE rowid = (SELECT rowid FROM services_fts_mapping WHERE service_id = NEW.service_id);
		END
	`).Error
	if err != nil {
		return err
	}

	err = db.Exec(`
		CREATE TRIGGER IF NOT EXISTS service_documents_fts_delete AFTER DELETE ON service_documents
		BEGIN
			UPDATE services_fts SET document_content = (
				SELECT GROUP_CONCAT(file_original_name || ' ' || COALESCE(description, ''), ' ')
				FROM service_documents WHERE service_id = OLD.service_id AND deleted_at IS NULL
			)
			WHERE rowid = (SELECT rowid FROM services_fts_mapping WHERE service_id = OLD.service_id);
		END
	`).Error

	return err
}

// MigrateFTSData checks if FTS index is empty and populates it
func MigrateFTSData(db *gorm.DB) error {
	var ftsCount int64
	db.Table("cases_fts_mapping").Count(&ftsCount)

	var caseCount int64
	db.Table("cases").Where("deleted_at IS NULL").Count(&caseCount)

	// If FTS is empty but cases exist, rebuild
	if ftsCount == 0 && caseCount > 0 {
		log.Printf("Found %d cases but FTS index is empty. Rebuilding...", caseCount)
		return RebuildFTSIndex(db)
	}

	// If FTS is significantly out of sync, rebuild
	if caseCount > 0 && float64(ftsCount)/float64(caseCount) < 0.9 {
		log.Printf("FTS index appears out of sync (%d/%d). Rebuilding...", ftsCount, caseCount)
		return RebuildFTSIndex(db)
	}

	return nil
}
