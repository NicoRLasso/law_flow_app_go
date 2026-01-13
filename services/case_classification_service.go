package services

import (
	"law_flow_app_go/models"
	"log"

	"gorm.io/gorm"
)

// GetCaseDomains fetches active case domains for a firm
func GetCaseDomains(db *gorm.DB, firmID string) ([]models.CaseDomain, error) {
	var domains []models.CaseDomain

	err := db.
		Where("firm_id = ?", firmID).
		Where("is_active = ?", true).
		Order("`order` ASC, name ASC").
		Find(&domains).Error

	return domains, err
}

// GetCaseBranches fetches active case branches for a domain
func GetCaseBranches(db *gorm.DB, firmID string, domainID string) ([]models.CaseBranch, error) {
	var branches []models.CaseBranch

	err := db.
		Where("firm_id = ?", firmID).
		Where("domain_id = ?", domainID).
		Where("is_active = ?", true).
		Order("`order` ASC, name ASC").
		Find(&branches).Error

	return branches, err
}

// GetCaseSubtypes fetches active case subtypes for a branch
func GetCaseSubtypes(db *gorm.DB, firmID string, branchID string) ([]models.CaseSubtype, error) {
	var subtypes []models.CaseSubtype

	err := db.
		Where("firm_id = ?", firmID).
		Where("branch_id = ?", branchID).
		Where("is_active = ?", true).
		Order("`order` ASC, name ASC").
		Find(&subtypes).Error

	return subtypes, err
}

// ValidateCaseClassification validates that the classification hierarchy is valid
func ValidateCaseClassification(db *gorm.DB, firmID string, domainID *string, branchID *string, subtypeID *string) bool {
	// If no classification provided, it's valid (optional)
	if domainID == nil && branchID == nil && subtypeID == nil {
		return true
	}

	// If subtype is provided, branch and domain must be provided
	if subtypeID != nil {
		if branchID == nil || domainID == nil {
			return false
		}

		var subtype models.CaseSubtype
		err := db.Preload("Branch").Preload("Branch.Domain").
			First(&subtype, "id = ? AND firm_id = ? AND is_active = ?", *subtypeID, firmID, true).Error
		if err != nil {
			return false
		}

		// Verify branch and domain match
		if subtype.BranchID != *branchID || subtype.Branch.DomainID != *domainID {
			return false
		}

		return true
	}

	// If branch is provided, domain must be provided
	if branchID != nil {
		if domainID == nil {
			return false
		}

		var branch models.CaseBranch
		err := db.First(&branch, "id = ? AND firm_id = ? AND domain_id = ? AND is_active = ?",
			*branchID, firmID, *domainID, true).Error
		return err == nil
	}

	// If only domain is provided, verify it exists
	if domainID != nil {
		var domain models.CaseDomain
		err := db.First(&domain, "id = ? AND firm_id = ? AND is_active = ?", *domainID, firmID, true).Error
		return err == nil
	}

	return true
}

// SeedCaseClassifications seeds default case classifications for a firm based on country
func SeedCaseClassifications(db *gorm.DB, firmID string, country string) error {
	switch country {
	case "Colombia":
		return seedColombianCaseClassifications(db, firmID, country)
	default:
		log.Printf("No case classifications to seed for country: %s", country)
		return nil
	}
}

// seedColombianCaseClassifications seeds the complete Colombian legal classification catalog
func seedColombianCaseClassifications(db *gorm.DB, firmID string, country string) error {
	// Seed domains first
	domains := []models.CaseDomain{
		{FirmID: firmID, Country: country, Code: "PUBLICO", Name: "Derecho Público", Order: 10, IsActive: true, IsSystem: true},
		{FirmID: firmID, Country: country, Code: "PRIVADO", Name: "Derecho Privado", Order: 20, IsActive: true, IsSystem: true},
		{FirmID: firmID, Country: country, Code: "SOCIAL", Name: "Derecho Social", Order: 30, IsActive: true, IsSystem: true},
		{FirmID: firmID, Country: country, Code: "TRANSVERSAL_MODERNO", Name: "Ramas Transversales y Modernas", Order: 40, IsActive: true, IsSystem: true},
	}

	domainMap := make(map[string]string) // code -> ID
	for _, domain := range domains {
		if err := db.Create(&domain).Error; err != nil {
			return err
		}
		domainMap[domain.Code] = domain.ID
	}

	// Seed branches
	branches := []struct {
		Code       string
		Name       string
		DomainCode string
		Order      int
	}{
		// Derecho Público
		{"CONSTITUCIONAL", "Derecho Constitucional", "PUBLICO", 10},
		{"ADMINISTRATIVO", "Derecho Administrativo", "PUBLICO", 20},
		{"PENAL", "Derecho Penal", "PUBLICO", 30},
		{"FINANCIERO_TRIBUTARIO", "Derecho Financiero y Tributario", "PUBLICO", 40},
		{"PROCESAL", "Derecho Procesal", "PUBLICO", 50},
		// Derecho Privado
		{"CIVIL", "Derecho Civil", "PRIVADO", 10},
		{"COMERCIAL", "Derecho Comercial", "PRIVADO", 20},
		{"INTERNACIONAL_PRIVADO", "Derecho Internacional Privado", "PRIVADO", 30},
		// Derecho Social
		{"LABORAL", "Derecho Laboral", "SOCIAL", 10},
		{"AMBIENTAL", "Derecho Ambiental", "SOCIAL", 20},
		{"AGRARIO", "Derecho Agrario", "SOCIAL", 30},
		// Ramas Transversales y Modernas
		{"DIGITAL", "Derecho Digital", "TRANSVERSAL_MODERNO", 10},
		{"SALUD", "Derecho de la Salud", "TRANSVERSAL_MODERNO", 20},
		{"MINERO_ENERGETICO", "Derecho Minero-Energético", "TRANSVERSAL_MODERNO", 30},
	}

	branchMap := make(map[string]string) // code -> ID
	for _, b := range branches {
		branch := models.CaseBranch{
			FirmID:   firmID,
			DomainID: domainMap[b.DomainCode],
			Country:  country,
			Code:     b.Code,
			Name:     b.Name,
			Order:    b.Order,
			IsActive: true,
			IsSystem: true,
		}
		if err := db.Create(&branch).Error; err != nil {
			return err
		}
		branchMap[branch.Code] = branch.ID
	}

	// Seed subtypes
	subtypes := []struct {
		Code       string
		Name       string
		BranchCode string
		Order      int
	}{
		// Derecho Constitucional
		{"CONSTITUCIONAL_DERECHOS_FUNDAMENTALES", "Derechos Fundamentales (Acción de Tutela)", "CONSTITUCIONAL", 10},
		{"CONSTITUCIONAL_ESTRUCTURA_PODER_PUBLICO", "Estructura y Ramas del Poder Público", "CONSTITUCIONAL", 20},
		{"CONSTITUCIONAL_ELECTORAL", "Derecho Electoral", "CONSTITUCIONAL", 30},
		// Derecho Administrativo
		{"ADMINISTRATIVO_CONTRATACION_ESTATAL", "Contratación Estatal (Ley 80)", "ADMINISTRATIVO", 10},
		{"ADMINISTRATIVO_RESPONSABILIDAD_ESTADO", "Responsabilidad Extracontractual del Estado", "ADMINISTRATIVO", 20},
		{"ADMINISTRATIVO_URBANISTICO_TIERRAS", "Derecho Urbanístico y de Tierras", "ADMINISTRATIVO", 30},
		{"ADMINISTRATIVO_DISCIPLINARIO", "Derecho Disciplinario", "ADMINISTRATIVO", 40},
		// Derecho Penal
		{"PENAL_ESPECIAL", "Derecho Penal Especial", "PENAL", 10},
		{"PENAL_CORPORATIVO", "Derecho Penal Corporativo", "PENAL", 20},
		{"PENAL_JUSTICIA_TRANSICIONAL", "Justicia Transicional (JEP)", "PENAL", 30},
		// Derecho Financiero y Tributario
		{"FIN_TRIB_FISCAL", "Derecho Fiscal (Impuestos y DIAN)", "FINANCIERO_TRIBUTARIO", 10},
		{"FIN_TRIB_ADUANERO_CAMBIARIO", "Derecho Aduanero y Cambiario", "FINANCIERO_TRIBUTARIO", 20},
		// Derecho Procesal
		{"PROCESAL_CIVIL", "Procedimiento Civil", "PROCESAL", 10},
		{"PROCESAL_PENAL", "Procedimiento Penal", "PROCESAL", 20},
		{"PROCESAL_LABORAL", "Procedimiento Laboral", "PROCESAL", 30},
		{"PROCESAL_ADMINISTRATIVO", "Procedimiento Administrativo", "PROCESAL", 40},
		// Derecho Civil
		{"CIVIL_PERSONAS_FAMILIA", "Personas y Familia", "CIVIL", 10},
		{"CIVIL_BIENES_PROPIEDAD", "Bienes y Propiedad", "CIVIL", 20},
		{"CIVIL_OBLIGACIONES_CONTRATOS", "Obligaciones y Contratos", "CIVIL", 30},
		// Derecho Comercial
		{"COMERCIAL_SOCIETARIO", "Societario", "COMERCIAL", 10},
		{"COMERCIAL_PROPIEDAD_INTELECTUAL", "Propiedad Intelectual", "COMERCIAL", 20},
		{"COMERCIAL_CONSUMIDOR", "Consumidor", "COMERCIAL", 30},
		{"COMERCIAL_INSOLVENCIA", "Insolvencia", "COMERCIAL", 40},
		// Derecho Internacional Privado
		{"INT_PRIVADO_CONFLICTOS_LEYES", "Conflictos de Leyes Internacionales", "INTERNACIONAL_PRIVADO", 10},
		// Derecho Laboral
		{"LABORAL_INDIVIDUAL", "Laboral Individual", "LABORAL", 10},
		{"LABORAL_COLECTIVO", "Laboral Colectivo", "LABORAL", 20},
		{"LABORAL_SEGURIDAD_SOCIAL", "Seguridad Social", "LABORAL", 30},
		// Derecho Ambiental
		{"AMBIENTAL_RECURSOS_NATURALES", "Protección de Recursos Naturales", "AMBIENTAL", 10},
		{"AMBIENTAL_LICENCIAS", "Licencias Ambientales", "AMBIENTAL", 20},
		// Derecho Agrario
		{"AGRARIO_PROPIEDAD_RURAL", "Propiedad Rural", "AGRARIO", 10},
		{"AGRARIO_EXPLOTACION_CAMPO", "Explotación del Campo", "AGRARIO", 20},
		// Derecho Digital
		{"DIGITAL_IA", "Inteligencia Artificial", "DIGITAL", 10},
		{"DIGITAL_PROTECCION_DATOS", "Protección de Datos", "DIGITAL", 20},
		{"DIGITAL_COMERCIO_ELECTRONICO", "Comercio Electrónico", "DIGITAL", 30},
		// Derecho de la Salud
		{"SALUD_RECLAMOS_EPS", "Reclamos EPS", "SALUD", 10},
		{"SALUD_PRESTADORES", "Prestadores de Salud", "SALUD", 20},
		// Derecho Minero-Energético
		{"MINERO_TRANSICION_ENERGETICA", "Transición Energética", "MINERO_ENERGETICO", 10},
		{"MINERO_ENERGIAS_RENOVABLES", "Energías Renovables", "MINERO_ENERGETICO", 20},
	}

	for _, s := range subtypes {
		subtype := models.CaseSubtype{
			FirmID:   firmID,
			BranchID: branchMap[s.BranchCode],
			Country:  country,
			Code:     s.Code,
			Name:     s.Name,
			Order:    s.Order,
			IsActive: true,
			IsSystem: true,
		}
		if err := db.Create(&subtype).Error; err != nil {
			return err
		}
	}

	log.Printf("Successfully seeded Colombian case classifications for firm %s: 4 domains, 14 branches, %d subtypes", firmID, len(subtypes))
	return nil
}
