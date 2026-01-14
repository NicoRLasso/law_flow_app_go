package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/templates/pages"
	"law_flow_app_go/templates/partials"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// CasesPageHandler renders the cases page
func CasesPageHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)

	component := pages.Cases("Cases | Law Flow", user, firm)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// GetCasesHandler returns a list of cases with filtering and pagination
func GetCasesHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)

	// Get query parameters for filtering
	status := c.QueryParam("status")
	assignedTo := c.QueryParam("assigned_to")
	dateFrom := c.QueryParam("date_from")
	dateTo := c.QueryParam("date_to")
	keyword := c.QueryParam("keyword")

	// Get pagination parameters
	page := 1
	limit := 20
	if pageParam := c.QueryParam("page"); pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		}
	}
	if limitParam := c.QueryParam("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Build firm-scoped query
	query := middleware.GetFirmScopedQuery(c, db.DB)

	// Apply role-based filter
	if currentUser.Role == "lawyer" {
		// Lawyers only see cases assigned to them
		query = query.Where("assigned_to_id = ?", currentUser.ID)
	}
	// Admins see all cases (no additional filter)

	// Apply status filter
	if status != "" && models.IsValidCaseStatus(status) {
		query = query.Where("status = ?", status)
	}

	// Apply assigned_to filter (admin only)
	if assignedTo != "" && currentUser.Role == "admin" {
		query = query.Where("assigned_to_id = ?", assignedTo)
	}

	// Apply date range filters
	if dateFrom != "" {
		if parsedDate, err := time.Parse("2006-01-02", dateFrom); err == nil {
			query = query.Where("opened_at >= ?", parsedDate)
		}
	}
	if dateTo != "" {
		if parsedDate, err := time.Parse("2006-01-02", dateTo); err == nil {
			// Add 24 hours to include the entire day
			endOfDay := parsedDate.Add(24 * time.Hour)
			query = query.Where("opened_at < ?", endOfDay)
		}
	}

	// Apply keyword search
	if keyword != "" {
		keyword = "%" + keyword + "%"
		query = query.Where(
			db.DB.Where("case_number LIKE ?", keyword).
				Or("title LIKE ?", keyword).
				Or("description LIKE ?", keyword).
				Or("EXISTS (SELECT 1 FROM users WHERE users.id = cases.client_id AND users.name LIKE ?)", keyword),
		)
	}

	// Get total count
	var total int64
	if err := query.Model(&models.Case{}).Count(&total).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to count cases")
	}

	// Calculate pagination
	offset := (page - 1) * limit
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	// Fetch paginated cases with preloading
	var cases []models.Case
	if err := query.
		Preload("Client").
		Preload("AssignedTo").
		Preload("Domain").
		Preload("Branch").
		Preload("Subtypes").
		Preload("Documents").
		Order("opened_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&cases).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch cases")
	}

	// Check if HTMX request
	if c.Request().Header.Get("HX-Request") == "true" {
		component := partials.CaseTable(cases, page, totalPages, limit, int(total))
		return component.Render(c.Request().Context(), c.Response().Writer)
	}

	// Return JSON with pagination metadata
	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": cases,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// GetCaseDetailHandler returns a case detail page
func GetCaseDetailHandler(c echo.Context) error {
	id := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)

	// Build firm-scoped query
	query := middleware.GetFirmScopedQuery(c, db.DB)

	// Apply role-based filter
	if currentUser.Role == "lawyer" {
		// Lawyers only see cases assigned to them
		query = query.Where("assigned_to_id = ?", currentUser.ID)
	}

	// Fetch case with all relationships
	var caseRecord models.Case
	if err := query.
		Preload("Client").
		Preload("AssignedTo").
		Preload("Domain").
		Preload("Branch").
		Preload("Subtypes").
		Preload("Documents").
		First(&caseRecord, "id = ?", id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Case not found")
	}

	// Render detail page
	component := pages.CaseDetail("Case Details | Law Flow", currentUser, currentFirm, caseRecord)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// GetLawyersForFilterHandler returns a list of lawyers for the filter dropdown (admin only)
func GetLawyersForFilterHandler(c echo.Context) error {
	// Build firm-scoped query
	query := middleware.GetFirmScopedQuery(c, db.DB)

	// Fetch active lawyers and admins
	var users []models.User
	if err := query.
		Where("role IN (?, ?)", "lawyer", "admin").
		Where("is_active = ?", true).
		Order("name ASC").
		Find(&users).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch lawyers")
	}

	// Return JSON
	return c.JSON(http.StatusOK, users)
}
