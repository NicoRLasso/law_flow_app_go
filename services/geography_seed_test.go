package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"law_flow_app_go/models"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// MockRoundTripper allows mocking HTTP responses
type MockRoundTripper struct {
	Response *http.Response
	Error    error
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.Response, nil
}

// TestMain sets up and tears down the test environment.
// It temporarily replaces the global httpClient in geography_seed.go with a mock.
func TestMain(m *testing.M) {
	// Save original httpClient and replace with our mock transport
	originalTransport := httpClient.Transport
	httpClient.Transport = &MockRoundTripper{} // Initialize with mock
	
	// Run all tests
	code := m.Run()
	
	// Restore original httpClient
	httpClient.Transport = originalTransport
	os.Exit(code)
}

// setupGeographyTestDB creates an in-memory SQLite database for testing
func setupGeographyTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(
		&models.Country{},
		&models.Department{},
		&models.City{},
		&models.LegalEntity{},
		&models.LegalSpecialty{},
		&models.CourtOffice{},
	)
	return db
}

// captureOutput captures log.Println output
func captureOutput(f func()) string {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	f()
	log.SetOutput(io.Discard) // Reset to discard
	return buf.String()
}

// setMockHTTPResponse configures the MockRoundTripper with a specific response
func setMockHTTPResponse(statusCode int, body []byte, err error) {
	mockTransport, ok := httpClient.Transport.(*MockRoundTripper)
	if !ok {
		// This should not happen if TestMain correctly sets the transport
		panic("httpClient.Transport is not MockRoundTripper")
	}
	
	mockTransport.Response = nil // Clear previous response
	mockTransport.Error = nil    // Clear previous error

	if err == nil {
		mockTransport.Response = &http.Response{
			StatusCode: statusCode,
			Body:       io.NopCloser(bytes.NewReader(body)),
			Header:     make(http.Header),
			Request:    &http.Request{}, // Must provide a non-nil Request for http.Response
		}
	} else {
		mockTransport.Error = err
	}
}

func TestSeedGeography(t *testing.T) {
	db := setupGeographyTestDB()

	t.Run("Seed Colombia for the first time", func(t *testing.T) {
		output := captureOutput(func() {
			err := SeedGeography(db)
			assert.NoError(t, err)
		})

		var country models.Country
		db.Where("code = ?", "COL").First(&country)
		assert.Equal(t, "Colombia", country.Name)

		var departments []models.Department
		db.Where("country_id = ?", country.ID).Find(&departments)
		assert.Len(t, departments, len(colombiaDepartments))
		assert.Contains(t, output, "Created Colombia country")
		assert.Contains(t, output, "Seeding geography data...")
		assert.Contains(t, output, "Geography seeding completed")
	})

	t.Run("Do not re-seed if Colombia and departments exist", func(t *testing.T) {
		output := captureOutput(func() {
			err := SeedGeography(db)
			assert.NoError(t, err)
		})
		assert.Contains(t, output, "Seeding geography data...")
		assert.NotContains(t, output, "Created Colombia country") // Should not re-create
		var country models.Country
		db.Where("code = ?", "COL").First(&country)
		var departments []models.Department
		db.Where("country_id = ?", country.ID).Find(&departments)
		assert.Len(t, departments, len(colombiaDepartments)) // Still same count
	})
}

func TestSeedCitiesForDepartment(t *testing.T) {
	db := setupGeographyTestDB()
	country := models.Country{Code: "COL", Name: "Colombia", IsActive: true}
	db.Create(&country)
	department := models.Department{CountryID: country.ID, Code: "05", Name: "ANTIOQUIA", IsActive: true}
	db.Create(&department)

	mockItems := []RamaJudicialItem{
		{Codigo: "", Nombre: "Seleccione una opción"},
		{Codigo: "001", Nombre: "MEDELLÍN"},
		{Codigo: "002", Nombre: "ENVIGADO"},
	}
	mockBody, _ := json.Marshal(mockItems)

	t.Run("Seed cities for the first time", func(t *testing.T) {
		setMockHTTPResponse(http.StatusOK, mockBody, nil)
		output := captureOutput(func() {
			err := SeedCitiesForDepartment(db, department.ID, department.Code)
			assert.NoError(t, err)
		})

		var cities []models.City
		db.Where("department_id = ?", department.ID).Find(&cities)
		assert.Len(t, cities, 2)
		assert.Equal(t, "MEDELLÍN", cities[0].Name)
		assert.Contains(t, output, "Fetching cities for department 05 from Rama Judicial API...")
		assert.Contains(t, output, "Seeded 2 cities for department 05")
	})

	t.Run("Do not re-seed if cities already exist", func(t *testing.T) {
		output := captureOutput(func() {
			err := SeedCitiesForDepartment(db, department.ID, department.Code)
			assert.NoError(t, err)
		})
		assert.Contains(t, output, "Cities already seeded for department 05, skipping")
	})

	t.Run("API error handling", func(t *testing.T) {
		// Use a different department to avoid cities already existing
		department2 := models.Department{CountryID: country.ID, Code: "11", Name: "BOGOTÁ", IsActive: true}
		db.Create(&department2)

		setMockHTTPResponse(http.StatusInternalServerError, nil, fmt.Errorf("network error"))
		err := SeedCitiesForDepartment(db, department2.ID, department2.Code)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch cities: HTTP request failed")
		assert.Contains(t, err.Error(), "network error")
	})
}

func TestSeedEntitiesForCity(t *testing.T) {
	db := setupGeographyTestDB()
	department := models.Department{Code: "05", Name: "ANTIOQUIA", IsActive: true}
	db.Create(&department)
	city := models.City{DepartmentID: department.ID, Code: "001", Name: "MEDELLÍN", IsActive: true}
	db.Create(&city)

	mockItems := []RamaJudicialItem{
		{Codigo: "", Nombre: "Seleccione una opción"},
		{Codigo: "40", Nombre: "JUZGADOS CIVILES MUNICIPALES"},
		{Codigo: "03", Nombre: "TRIBUNALES SUPERIORES"},
	}
	mockBody, _ := json.Marshal(mockItems)

	t.Run("Seed entities for the first time", func(t *testing.T) {
		setMockHTTPResponse(http.StatusOK, mockBody, nil)
		output := captureOutput(func() {
			err := SeedEntitiesForCity(db, city.ID, city.Code)
			assert.NoError(t, err)
		})

		var entities []models.LegalEntity
		db.Where("city_id = ?", city.ID).Find(&entities)
		assert.Len(t, entities, 2)
		assert.Contains(t, output, "Fetching entities for city 001 from Rama Judicial API...")
	})

	t.Run("Do not re-seed if entities already exist", func(t *testing.T) {
		output := captureOutput(func() {
			err := SeedEntitiesForCity(db, city.ID, city.Code)
			assert.NoError(t, err)
		})
		assert.Empty(t, output) // No log message for skipping
	})

	t.Run("API error handling", func(t *testing.T) {
		// Use a different city to avoid entities already existing
		city2 := models.City{DepartmentID: department.ID, Code: "002", Name: "ENVIGADO", IsActive: true}
		db.Create(&city2)

		setMockHTTPResponse(http.StatusInternalServerError, nil, fmt.Errorf("network error"))
		err := SeedEntitiesForCity(db, city2.ID, city2.Code)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch entities: HTTP request failed")
		assert.Contains(t, err.Error(), "network error")
	})
}

func TestSeedSpecialtiesForEntity(t *testing.T) {
	db := setupGeographyTestDB()
	city := models.City{Code: "001", Name: "MEDELLÍN", IsActive: true}
	db.Create(&city)
	entity := models.LegalEntity{CityID: city.ID, Code: "40", Name: "JUZGADOS CIVILES MUNICIPALES", IsActive: true}
	db.Create(&entity)

	mockItems := []RamaJudicialItem{
		{Codigo: "", Nombre: "Seleccione una opción"},
		{Codigo: "03", Nombre: "CIVIL"},
		{Codigo: "01", Nombre: "FAMILIA"},
	}
	mockBody, _ := json.Marshal(mockItems)

	t.Run("Seed specialties for the first time", func(t *testing.T) {
		setMockHTTPResponse(http.StatusOK, mockBody, nil)
		output := captureOutput(func() {
			err := SeedSpecialtiesForEntity(db, entity.ID, entity.Code)
			assert.NoError(t, err)
		})

		var specialties []models.LegalSpecialty
		db.Where("entity_id = ?", entity.ID).Find(&specialties)
		assert.Len(t, specialties, 2)
		assert.Contains(t, output, "Fetching specialties for entity 40 from Rama Judicial API...")
	})

	t.Run("Do not re-seed if specialties already exist", func(t *testing.T) {
		output := captureOutput(func() {
			err := SeedSpecialtiesForEntity(db, entity.ID, entity.Code)
			assert.NoError(t, err)
		})
		assert.Empty(t, output)
	})

	t.Run("API error handling", func(t *testing.T) {
		// Use a different entity to avoid specialties already existing
		entity2 := models.LegalEntity{CityID: city.ID, Code: "03", Name: "TRIBUNALES SUPERIORES", IsActive: true}
		db.Create(&entity2)

		setMockHTTPResponse(http.StatusInternalServerError, nil, fmt.Errorf("network error"))
		err := SeedSpecialtiesForEntity(db, entity2.ID, entity2.Code)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch specialties: HTTP request failed")
		assert.Contains(t, err.Error(), "network error")
	})
}

func TestSeedCourtOfficesForSpecialty(t *testing.T) {
	db := setupGeographyTestDB()
	entity := models.LegalEntity{Code: "40", Name: "JUZGADOS CIVILES MUNICIPALES", IsActive: true}
	db.Create(&entity)
	specialty := models.LegalSpecialty{EntityID: entity.ID, Code: "03", Name: "CIVIL", IsActive: true}
	db.Create(&specialty)

	mockItems := []RamaJudicialItem{
		{Codigo: "", Nombre: "Seleccione una opción"},
		{Codigo: "890", Nombre: "JUZGADO 001 CIVIL MUNICIPAL"},
		{Codigo: "891", Nombre: "JUZGADO 002 CIVIL MUNICIPAL"},
	}
	mockBody, _ := json.Marshal(mockItems)

	t.Run("Seed court offices for the first time", func(t *testing.T) {
		setMockHTTPResponse(http.StatusOK, mockBody, nil)
		output := captureOutput(func() {
			err := SeedCourtOfficesForSpecialty(db, specialty.ID, specialty.Code)
			assert.NoError(t, err)
		})

		var offices []models.CourtOffice
		db.Where("specialty_id = ?", specialty.ID).Find(&offices)
		assert.Len(t, offices, 2)
		assert.Contains(t, output, "Fetching court offices for specialty 03 from Rama Judicial API...")
	})

	t.Run("Do not re-seed if court offices already exist", func(t *testing.T) {
		output := captureOutput(func() {
			err := SeedCourtOfficesForSpecialty(db, specialty.ID, specialty.Code)
			assert.NoError(t, err)
		})
		assert.Empty(t, output)
	})

	t.Run("API error handling", func(t *testing.T) {
		// Use a different specialty to avoid court offices already existing
		specialty2 := models.LegalSpecialty{EntityID: entity.ID, Code: "01", Name: "FAMILIA", IsActive: true}
		db.Create(&specialty2)

		setMockHTTPResponse(http.StatusInternalServerError, nil, fmt.Errorf("network error"))
		err := SeedCourtOfficesForSpecialty(db, specialty2.ID, specialty2.Code)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch court offices: HTTP request failed")
		assert.Contains(t, err.Error(), "network error")
	})
}

func TestGetFunctions(t *testing.T) {
	db := setupGeographyTestDB()
	colombia := models.Country{ID: "co-id", Code: "COL", Name: "Colombia", IsActive: true}
	usa := models.Country{ID: "us-id", Code: "USA", Name: "United States", IsActive: false}
	db.Create(&colombia)
	db.Create(&usa)

	antioquia := models.Department{ID: "ant-id", CountryID: colombia.ID, Code: "05", Name: "Antioquia", IsActive: true}
	cundinamarca := models.Department{ID: "cun-id", CountryID: colombia.ID, Code: "25", Name: "Cundinamarca", IsActive: false}
	db.Create(&antioquia)
	db.Create(&cundinamarca)

	medellin := models.City{ID: "med-id", DepartmentID: antioquia.ID, Code: "001", Name: "Medellín", IsActive: true}
	bogota := models.City{ID: "bog-id", DepartmentID: cundinamarca.ID, Code: "001", Name: "Bogotá", IsActive: false}
	db.Create(&medellin)
	db.Create(&bogota)

	civilEntities := models.LegalEntity{ID: "entity-civil-id", CityID: medellin.ID, Code: "40", Name: "Juzgados Civiles", IsActive: true}
	familyEntities := models.LegalEntity{ID: "entity-family-id", CityID: medellin.ID, Code: "03", Name: "Juzgados Familia", IsActive: false}
	db.Create(&civilEntities)
	db.Create(&familyEntities)

	civilSpecialty := models.LegalSpecialty{ID: "spec-civil-id", EntityID: civilEntities.ID, Code: "03", Name: "Civil", IsActive: true}
	penalSpecialty := models.LegalSpecialty{ID: "spec-penal-id", EntityID: civilEntities.ID, Code: "01", Name: "Penal", IsActive: false}
	db.Create(&civilSpecialty)
	db.Create(&penalSpecialty)

	courtOffice1 := models.CourtOffice{ID: "office-1-id", SpecialtyID: civilSpecialty.ID, Code: "890", Name: "Juzgado 001 Civil", IsActive: true}
	courtOffice2 := models.CourtOffice{ID: "office-2-id", SpecialtyID: civilSpecialty.ID, Code: "891", Name: "Juzgado 002 Civil", IsActive: false}
	db.Create(&courtOffice1)
	db.Create(&courtOffice2)

	t.Run("GetActiveCountries", func(t *testing.T) {
		countries, err := GetActiveCountries(db)
		assert.NoError(t, err)
		assert.Len(t, countries, 1)
		assert.Equal(t, "Colombia", countries[0].Name)
	})

	t.Run("GetCountryByID", func(t *testing.T) {
		country, err := GetCountryByID(db, colombia.ID)
		assert.NoError(t, err)
		assert.Equal(t, "Colombia", country.Name)

		_, err = GetCountryByID(db, "non-existent")
		assert.Error(t, err)
	})

	t.Run("GetDepartmentsByCountry", func(t *testing.T) {
		depts, err := GetDepartmentsByCountry(db, colombia.ID)
		assert.NoError(t, err)
		assert.Len(t, depts, 1)
		assert.Equal(t, "Antioquia", depts[0].Name)

		depts, err = GetDepartmentsByCountry(db, usa.ID) // No active depts for USA
		assert.NoError(t, err)
		assert.Len(t, depts, 0)
	})

	t.Run("GetDepartmentsByCountryCode", func(t *testing.T) {
		depts, err := GetDepartmentsByCountryCode(db, "COL")
		assert.NoError(t, err)
		assert.Len(t, depts, 1)
		assert.Equal(t, "Antioquia", depts[0].Name)

		_, err = GetDepartmentsByCountryCode(db, "NON")
		assert.Error(t, err)
	})

	t.Run("GetCitiesByDepartment", func(t *testing.T) {
		mockItems := []RamaJudicialItem{
			{Codigo: "", Nombre: "Seleccione una opción"},
			{Codigo: "001", Nombre: "MEDELLÍN"},
		}
		mockBody, _ := json.Marshal(mockItems)
		setMockHTTPResponse(http.StatusOK, mockBody, nil)

		cities, err := GetCitiesByDepartment(db, antioquia.ID)
		assert.NoError(t, err)
		assert.Len(t, cities, 1)
		assert.Equal(t, "Medellín", cities[0].Name)

		// Test with already seeded city
		cities, err = GetCitiesByDepartment(db, antioquia.ID)
		assert.NoError(t, err)
		assert.Len(t, cities, 1)
	})

	t.Run("GetEntitiesByCity", func(t *testing.T) {
		mockItems := []RamaJudicialItem{
			{Codigo: "", Nombre: "Seleccione una opción"},
			{Codigo: "40", Nombre: "JUZGADOS CIVILES MUNICIPALES"},
		}
		mockBody, _ := json.Marshal(mockItems)
		setMockHTTPResponse(http.StatusOK, mockBody, nil)

		entities, err := GetEntitiesByCity(db, medellin.ID)
		assert.NoError(t, err)
		assert.Len(t, entities, 1)
		assert.Equal(t, "Juzgados Civiles", entities[0].Name)
	})

	t.Run("GetSpecialtiesByEntity", func(t *testing.T) {
		mockItems := []RamaJudicialItem{
			{Codigo: "", Nombre: "Seleccione una opción"},
			{Codigo: "03", Nombre: "CIVIL"},
		}
		mockBody, _ := json.Marshal(mockItems)
		setMockHTTPResponse(http.StatusOK, mockBody, nil)

		specialties, err := GetSpecialtiesByEntity(db, civilEntities.ID)
		assert.NoError(t, err)
		assert.Len(t, specialties, 1)
		assert.Equal(t, "Civil", specialties[0].Name)
	})

	t.Run("GetCourtOfficesBySpecialty", func(t *testing.T) {
		mockItems := []RamaJudicialItem{
			{Codigo: "", Nombre: "Seleccione una opción"},
			{Codigo: "890", Nombre: "JUZGADO 001 CIVIL MUNICIPAL"},
		}
		mockBody, _ := json.Marshal(mockItems)
		setMockHTTPResponse(http.StatusOK, mockBody, nil)

		offices, err := GetCourtOfficesBySpecialty(db, civilSpecialty.ID)
		assert.NoError(t, err)
		assert.Len(t, offices, 1)
		assert.Equal(t, "Juzgado 001 Civil", offices[0].Name)
	})
}