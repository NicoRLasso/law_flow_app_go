package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/components"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

var searchService *services.SearchService

// InitSearchService initializes the search service
func InitSearchService() {
	searchService = services.NewSearchService(db.DB)
}

// SearchCasesHandler handles case searches
// GET /api/search?q=keyword&limit=10
func SearchCasesHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)

	if currentUser == nil || currentFirm == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Not authenticated")
	}

	// Get parameters
	query := c.QueryParam("q")
	if query == "" || len(query) < 2 {
		// Return empty if query too short
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, "")
		}
		return c.JSON(http.StatusOK, []services.SearchResult{})
	}

	limit := 10
	if limitParam := c.QueryParam("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	// Perform search with role filter
	results, err := searchService.SearchWithRoleFilter(
		c.Request().Context(),
		currentFirm.ID,
		currentUser.ID,
		currentUser.Role,
		query,
		limit,
	)
	if err != nil {
		c.Logger().Error("Search failed:", err)
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, `<div class="p-4 text-center text-error text-sm">Error en la busqueda</div>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Search failed")
	}

	// If HTMX request, return HTML
	if c.Request().Header.Get("HX-Request") == "true" {
		component := components.SearchResults(c.Request().Context(), results, query)
		return component.Render(c.Request().Context(), c.Response().Writer)
	}

	// Return JSON
	return c.JSON(http.StatusOK, map[string]interface{}{
		"results": results,
		"query":   query,
		"count":   len(results),
	})
}
