package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/services"
	"net/http"

	"github.com/labstack/echo/v4"
)

// GetCountriesHandler returns all active countries
// GET /api/geography/countries
func GetCountriesHandler(c echo.Context) error {
	countries, err := services.GetActiveCountries(db.DB)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch countries")
	}

	// Return as JSON for HTMX to process
	return c.JSON(http.StatusOK, countries)
}

// GetDepartmentsHandler returns all departments for a country
// GET /api/geography/departments?country_id=xxx or ?country_code=xxx
func GetDepartmentsHandler(c echo.Context) error {
	countryID := c.QueryParam("country_id")
	countryCode := c.QueryParam("country_code")

	var departments []struct {
		ID   string `json:"id"`
		Code string `json:"code"`
		Name string `json:"name"`
	}

	if countryID != "" {
		depts, err := services.GetDepartmentsByCountry(db.DB, countryID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch departments")
		}
		for _, d := range depts {
			departments = append(departments, struct {
				ID   string `json:"id"`
				Code string `json:"code"`
				Name string `json:"name"`
			}{d.ID, d.Code, d.Name})
		}
	} else if countryCode != "" {
		depts, err := services.GetDepartmentsByCountryCode(db.DB, countryCode)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch departments")
		}
		for _, d := range depts {
			departments = append(departments, struct {
				ID   string `json:"id"`
				Code string `json:"code"`
				Name string `json:"name"`
			}{d.ID, d.Code, d.Name})
		}
	} else {
		return echo.NewHTTPError(http.StatusBadRequest, "country_id or country_code is required")
	}

	return c.JSON(http.StatusOK, departments)
}

// GetCitiesHandler returns all cities for a department
// GET /api/geography/cities?department_id=xxx
func GetCitiesHandler(c echo.Context) error {
	departmentID := c.QueryParam("department_id")
	if departmentID == "" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, `<option value="">Select a city</option>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "department_id is required")
	}

	cities, err := services.GetCitiesByDepartment(db.DB, departmentID)
	if err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, `<option value="">Error loading cities</option>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch cities")
	}

	// Return HTML options for HTMX
	if c.Request().Header.Get("HX-Request") == "true" {
		html := `<option value="">Select a city</option>`
		for _, city := range cities {
			html += `<option value="` + city.ID + `" data-code="` + city.Code + `">` + city.Name + ` (` + city.Code + `)</option>`
		}
		return c.HTML(http.StatusOK, html)
	}

	// Return JSON for API requests
	var result []struct {
		ID   string `json:"id"`
		Code string `json:"code"`
		Name string `json:"name"`
	}
	for _, city := range cities {
		result = append(result, struct {
			ID   string `json:"id"`
			Code string `json:"code"`
			Name string `json:"name"`
		}{city.ID, city.Code, city.Name})
	}

	return c.JSON(http.StatusOK, result)
}

// GetEntitiesHandler returns all legal entities for a city
// GET /api/geography/entities?city_id=xxx
func GetEntitiesHandler(c echo.Context) error {
	cityID := c.QueryParam("city_id")
	if cityID == "" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, `<option value="">Select an entity</option>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "city_id is required")
	}

	entities, err := services.GetEntitiesByCity(db.DB, cityID)
	if err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, `<option value="">Error loading entities</option>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch entities")
	}

	// Return HTML options for HTMX
	if c.Request().Header.Get("HX-Request") == "true" {
		html := `<option value="">Select an entity</option>`
		for _, entity := range entities {
			html += `<option value="` + entity.ID + `" data-code="` + entity.Code + `">` + entity.Name + `</option>`
		}
		return c.HTML(http.StatusOK, html)
	}

	var result []struct {
		ID   string `json:"id"`
		Code string `json:"code"`
		Name string `json:"name"`
	}
	for _, entity := range entities {
		result = append(result, struct {
			ID   string `json:"id"`
			Code string `json:"code"`
			Name string `json:"name"`
		}{entity.ID, entity.Code, entity.Name})
	}

	return c.JSON(http.StatusOK, result)
}

// GetSpecialtiesHandler returns all specialties for an entity
// GET /api/geography/specialties?entity_id=xxx
func GetSpecialtiesHandler(c echo.Context) error {
	entityID := c.QueryParam("entity_id")
	if entityID == "" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, `<option value="">Select a specialty</option>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "entity_id is required")
	}

	specialties, err := services.GetSpecialtiesByEntity(db.DB, entityID)
	if err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, `<option value="">Error loading specialties</option>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch specialties")
	}

	// Return HTML options for HTMX
	if c.Request().Header.Get("HX-Request") == "true" {
		html := `<option value="">Select a specialty</option>`
		for _, s := range specialties {
			html += `<option value="` + s.ID + `" data-code="` + s.Code + `">` + s.Name + `</option>`
		}
		return c.HTML(http.StatusOK, html)
	}

	var result []struct {
		ID   string `json:"id"`
		Code string `json:"code"`
		Name string `json:"name"`
	}
	for _, s := range specialties {
		result = append(result, struct {
			ID   string `json:"id"`
			Code string `json:"code"`
			Name string `json:"name"`
		}{s.ID, s.Code, s.Name})
	}

	return c.JSON(http.StatusOK, result)
}

// GetCourtOfficesHandler returns all court offices for a specialty
// GET /api/geography/court-offices?specialty_id=xxx
func GetCourtOfficesHandler(c echo.Context) error {
	specialtyID := c.QueryParam("specialty_id")
	if specialtyID == "" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, `<option value="">Select a court office</option>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "specialty_id is required")
	}

	offices, err := services.GetCourtOfficesBySpecialty(db.DB, specialtyID)
	if err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, `<option value="">Error loading court offices</option>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch court offices")
	}

	// Return HTML options for HTMX
	if c.Request().Header.Get("HX-Request") == "true" {
		html := `<option value="">Select a court office</option>`
		for _, o := range offices {
			html += `<option value="` + o.Code + `" data-code="` + o.Code + `">` + o.Name + ` (` + o.Code + `)</option>`
		}
		return c.HTML(http.StatusOK, html)
	}

	var result []struct {
		ID   string `json:"id"`
		Code string `json:"code"`
		Name string `json:"name"`
	}
	for _, o := range offices {
		result = append(result, struct {
			ID   string `json:"id"`
			Code string `json:"code"`
			Name string `json:"name"`
		}{o.ID, o.Code, o.Name})
	}

	return c.JSON(http.StatusOK, result)
}
