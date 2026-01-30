package services

import (
	"encoding/json"
	"fmt"
	"io"
	"law_flow_app_go/models"
	"log"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"
)

const ramaJudicialBaseURL = "https://consultaprocesos.ramajudicial.gov.co:448/api/v2/Lista"

// Colombian departments data
var colombiaDepartments = []struct {
	Code string
	Name string
}{
	{"91", "AMAZONAS"},
	{"05", "ANTIOQUIA"},
	{"81", "ARAUCA"},
	{"08", "ATLÁNTICO"},
	{"11", "BOGOTÁ"},
	{"13", "BOLÍVAR"},
	{"15", "BOYACÁ"},
	{"17", "CALDAS"},
	{"18", "CAQUETÁ"},
	{"85", "CASANARE"},
	{"19", "CAUCA"},
	{"20", "CESAR"},
	{"27", "CHOCÓ"},
	{"23", "CÓRDOBA"},
	{"25", "CUNDINAMARCA"},
	{"94", "GUAINÍA"},
	{"95", "GUAVIARE"},
	{"41", "HUILA"},
	{"44", "LA GUAJIRA"},
	{"47", "MAGDALENA"},
	{"50", "META"},
	{"52", "NARIÑO"},
	{"54", "NORTE DE SANTANDER"},
	{"86", "PUTUMAYO"},
	{"63", "QUINDÍO"},
	{"66", "RISARALDA"},
	{"88", "SAN ANDRÉS"},
	{"68", "SANTANDER"},
	{"70", "SUCRE"},
	{"73", "TOLIMA"},
	{"76", "VALLE DEL CAUCA"},
	{"97", "VAUPÉS"},
	{"99", "VICHADA"},
}

// RamaJudicialItem represents an item from the Rama Judicial API
type RamaJudicialItem struct {
	Codigo string `json:"codigo"`
	Nombre string `json:"nombre"`
}

// SeedGeography seeds countries and Colombia's departments
func SeedGeography(db *gorm.DB) error {
	log.Println("Seeding geography data...")

	// Check if Colombia already exists
	var colombia models.Country
	result := db.Where("code = ?", "COL").First(&colombia)

	if result.Error == gorm.ErrRecordNotFound {
		// Create Colombia
		colombia = models.Country{
			Code:     "COL",
			Name:     "Colombia",
			IsActive: true,
		}
		if err := db.Create(&colombia).Error; err != nil {
			return fmt.Errorf("failed to create Colombia country: %w", err)
		}
		log.Println("Created Colombia country")
	} else if result.Error != nil {
		return fmt.Errorf("failed to check for Colombia: %w", result.Error)
	}

	// Seed departments
	for _, dept := range colombiaDepartments {
		var existing models.Department
		if err := db.Where("country_id = ? AND code = ?", colombia.ID, dept.Code).First(&existing).Error; err == gorm.ErrRecordNotFound {
			newDept := models.Department{
				CountryID: colombia.ID,
				Code:      dept.Code,
				Name:      dept.Name,
				IsActive:  true,
			}
			if err := db.Create(&newDept).Error; err != nil {
				log.Printf("Failed to create department %s: %v", dept.Name, err)
			}
		}
	}

	log.Println("Geography seeding completed")
	return nil
}

// SeedCitiesForDepartment fetches and seeds cities for a specific department from the Rama Judicial API
func SeedCitiesForDepartment(db *gorm.DB, departmentID string, departmentCode string) error {
	// Check if cities already exist for this department
	var count int64
	if err := db.Model(&models.City{}).Where("department_id = ?", departmentID).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to count existing cities: %w", err)
	}

	if count > 0 {
		log.Printf("Cities already seeded for department %s, skipping", departmentCode)
		return nil
	}

	log.Printf("Fetching cities for department %s from Rama Judicial API...", departmentCode)

	items, err := fetchFromRamaJudicial(fmt.Sprintf("%s/Ciudades/%s", ramaJudicialBaseURL, departmentCode))
	if err != nil {
		return fmt.Errorf("failed to fetch cities: %w", err)
	}

	for _, item := range items {
		code := strings.TrimSpace(item.Codigo)
		if code == "" {
			continue // Skip "Seleccione..." option
		}

		city := models.City{
			DepartmentID: departmentID,
			Code:         code,
			Name:         strings.TrimSpace(item.Nombre),
			IsActive:     true,
		}
		if err := db.Create(&city).Error; err != nil {
			log.Printf("Failed to create city %s: %v", item.Nombre, err)
		}
	}

	log.Printf("Seeded %d cities for department %s", len(items)-1, departmentCode) // -1 for "Seleccione..."
	return nil
}

// SeedEntitiesForCity fetches and seeds legal entities for a specific city
func SeedEntitiesForCity(db *gorm.DB, cityID string, cityCode string) error {
	var count int64
	if err := db.Model(&models.LegalEntity{}).Where("city_id = ?", cityID).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to count existing entities: %w", err)
	}

	if count > 0 {
		return nil // Already seeded
	}

	log.Printf("Fetching entities for city %s from Rama Judicial API...", cityCode)

	items, err := fetchFromRamaJudicial(fmt.Sprintf("%s/Entidades/%s", ramaJudicialBaseURL, cityCode))
	if err != nil {
		return fmt.Errorf("failed to fetch entities: %w", err)
	}

	for _, item := range items {
		code := strings.TrimSpace(item.Codigo)
		if code == "" {
			continue
		}

		entity := models.LegalEntity{
			CityID:   cityID,
			Code:     code,
			Name:     strings.TrimSpace(item.Nombre),
			IsActive: true,
		}
		if err := db.Create(&entity).Error; err != nil {
			log.Printf("Failed to create entity %s: %v", item.Nombre, err)
		}
	}

	return nil
}

// SeedSpecialtiesForEntity fetches and seeds specialties for a specific entity
func SeedSpecialtiesForEntity(db *gorm.DB, entityID string, entityCode string) error {
	var count int64
	if err := db.Model(&models.LegalSpecialty{}).Where("entity_id = ?", entityID).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to count existing specialties: %w", err)
	}

	if count > 0 {
		return nil // Already seeded
	}

	log.Printf("Fetching specialties for entity %s from Rama Judicial API...", entityCode)

	items, err := fetchFromRamaJudicial(fmt.Sprintf("%s/Especialidades/%s", ramaJudicialBaseURL, entityCode))
	if err != nil {
		return fmt.Errorf("failed to fetch specialties: %w", err)
	}

	for _, item := range items {
		code := strings.TrimSpace(item.Codigo)
		if code == "" {
			continue
		}

		specialty := models.LegalSpecialty{
			EntityID: entityID,
			Code:     code,
			Name:     strings.TrimSpace(item.Nombre),
			IsActive: true,
		}
		if err := db.Create(&specialty).Error; err != nil {
			log.Printf("Failed to create specialty %s: %v", item.Nombre, err)
		}
	}

	return nil
}

// SeedCourtOfficesForSpecialty fetches and seeds court offices for a specific specialty
func SeedCourtOfficesForSpecialty(db *gorm.DB, specialtyID string, specialtyCode string) error {
	var count int64
	if err := db.Model(&models.CourtOffice{}).Where("specialty_id = ?", specialtyID).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to count existing court offices: %w", err)
	}

	if count > 0 {
		return nil // Already seeded
	}

	log.Printf("Fetching court offices for specialty %s from Rama Judicial API...", specialtyCode)

	items, err := fetchFromRamaJudicial(fmt.Sprintf("%s/Despachos/%s", ramaJudicialBaseURL, specialtyCode))
	if err != nil {
		return fmt.Errorf("failed to fetch court offices: %w", err)
	}

	for _, item := range items {
		code := strings.TrimSpace(item.Codigo)
		if code == "" {
			continue
		}

		office := models.CourtOffice{
			SpecialtyID: specialtyID,
			Code:        code,
			Name:        strings.TrimSpace(item.Nombre),
			IsActive:    true,
		}
		if err := db.Create(&office).Error; err != nil {
			log.Printf("Failed to create court office %s: %v", item.Nombre, err)
		}
	}

	return nil
}

// fetchFromRamaJudicial makes an HTTP request to the Rama Judicial API
func fetchFromRamaJudicial(url string) ([]RamaJudicialItem, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var items []RamaJudicialItem
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return items, nil
}

// GetActiveCountries returns all active countries
func GetActiveCountries(db *gorm.DB) ([]models.Country, error) {
	var countries []models.Country
	if err := db.Where("is_active = ?", true).Order("name").Find(&countries).Error; err != nil {
		return nil, err
	}
	return countries, nil
}

// GetCountryByID returns a country by its ID
func GetCountryByID(db *gorm.DB, id string) (*models.Country, error) {
	var country models.Country
	if err := db.First(&country, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &country, nil
}

// GetDepartmentsByCountry returns all departments for a country
func GetDepartmentsByCountry(db *gorm.DB, countryID string) ([]models.Department, error) {
	var departments []models.Department
	if err := db.Where("country_id = ? AND is_active = ?", countryID, true).Order("name").Find(&departments).Error; err != nil {
		return nil, err
	}
	return departments, nil
}

// GetDepartmentsByCountryCode returns all departments for a country by country code
func GetDepartmentsByCountryCode(db *gorm.DB, countryCode string) ([]models.Department, error) {
	var country models.Country
	if err := db.Where("code = ?", countryCode).First(&country).Error; err != nil {
		return nil, err
	}
	return GetDepartmentsByCountry(db, country.ID)
}

// GetCitiesByDepartment returns all cities for a department, seeding from API if needed
func GetCitiesByDepartment(db *gorm.DB, departmentID string) ([]models.City, error) {
	// First, check if we need to seed
	var dept models.Department
	if err := db.First(&dept, "id = ?", departmentID).Error; err != nil {
		return nil, err
	}

	// Seed if needed (will be skipped if already seeded)
	if err := SeedCitiesForDepartment(db, departmentID, dept.Code); err != nil {
		log.Printf("Warning: failed to seed cities for department %s: %v", dept.Code, err)
	}

	var cities []models.City
	if err := db.Where("department_id = ? AND is_active = ?", departmentID, true).Order("name").Find(&cities).Error; err != nil {
		return nil, err
	}
	return cities, nil
}

// GetEntitiesByCity returns all legal entities for a city, seeding from API if needed
func GetEntitiesByCity(db *gorm.DB, cityID string) ([]models.LegalEntity, error) {
	var city models.City
	if err := db.First(&city, "id = ?", cityID).Error; err != nil {
		return nil, err
	}

	if err := SeedEntitiesForCity(db, cityID, city.Code); err != nil {
		log.Printf("Warning: failed to seed entities for city %s: %v", city.Code, err)
	}

	var entities []models.LegalEntity
	if err := db.Where("city_id = ? AND is_active = ?", cityID, true).Order("name").Find(&entities).Error; err != nil {
		return nil, err
	}
	return entities, nil
}

// GetSpecialtiesByEntity returns all specialties for an entity, seeding from API if needed
func GetSpecialtiesByEntity(db *gorm.DB, entityID string) ([]models.LegalSpecialty, error) {
	var entity models.LegalEntity
	if err := db.First(&entity, "id = ?", entityID).Error; err != nil {
		return nil, err
	}

	if err := SeedSpecialtiesForEntity(db, entityID, entity.Code); err != nil {
		log.Printf("Warning: failed to seed specialties for entity %s: %v", entity.Code, err)
	}

	var specialties []models.LegalSpecialty
	if err := db.Where("entity_id = ? AND is_active = ?", entityID, true).Order("name").Find(&specialties).Error; err != nil {
		return nil, err
	}
	return specialties, nil
}

// GetCourtOfficesBySpecialty returns all court offices for a specialty, seeding from API if needed
func GetCourtOfficesBySpecialty(db *gorm.DB, specialtyID string) ([]models.CourtOffice, error) {
	var specialty models.LegalSpecialty
	if err := db.First(&specialty, "id = ?", specialtyID).Error; err != nil {
		return nil, err
	}

	if err := SeedCourtOfficesForSpecialty(db, specialtyID, specialty.Code); err != nil {
		log.Printf("Warning: failed to seed court offices for specialty %s: %v", specialty.Code, err)
	}

	var offices []models.CourtOffice
	if err := db.Where("specialty_id = ? AND is_active = ?", specialtyID, true).Order("name").Find(&offices).Error; err != nil {
		return nil, err
	}
	return offices, nil
}
