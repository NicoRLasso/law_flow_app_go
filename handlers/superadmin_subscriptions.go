package handlers

import (
	"fmt"
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/superadmin"
	"net/http"

	"github.com/labstack/echo/v4"
)

// SuperadminPlansPageHandler renders the plan management page
func SuperadminPlansPageHandler(c echo.Context) error {
	var plans []models.Plan
	if err := db.DB.Order("display_order asc").Find(&plans).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch plans")
	}

	user := middleware.GetCurrentUser(c)
	csrfToken := middleware.GetCSRFToken(c)
	component := superadmin.PlansPage(c.Request().Context(), "Plan Management | Superadmin", csrfToken, user, plans)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// SuperadminAddOnsPageHandler renders the add-on management page
func SuperadminAddOnsPageHandler(c echo.Context) error {
	var addons []models.PlanAddOn
	if err := db.DB.Order("display_order asc").Find(&addons).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch add-ons")
	}

	user := middleware.GetCurrentUser(c)
	csrfToken := middleware.GetCSRFToken(c)
	component := superadmin.AddOnsPage(c.Request().Context(), "Add-on Management | Superadmin", csrfToken, user, addons)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// SuperadminUpdateFirmSubscriptionHandler allows changing a firm's plan manually
func SuperadminUpdateFirmSubscriptionHandler(c echo.Context) error {
	firmID := c.Param("id")
	planID := c.FormValue("plan_id")

	fmt.Printf("DEBUG: Received Plan Update Request - FirmID: [%s], PlanID: [%s]\n", firmID, planID)

	if firmID == "" || planID == "" {
		fmt.Printf("DEBUG: Validation Failed - Missing IDs\n")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Firm ID and Plan ID are required"})
	}

	fmt.Printf("DEBUG: Commencing ChangeFirmPlan call\n")

	user := middleware.GetCurrentUser(c)
	var userID *string
	if user != nil {
		userID = &user.ID
	}

	err := services.ChangeFirmPlan(db.DB, firmID, planID, userID)
	if err != nil {
		fmt.Printf("DEBUG: Error updating plan: %v\n", err)
		return c.String(http.StatusInternalServerError, "<div class='text-red-400'>"+err.Error()+"</div>")
	}

	fmt.Printf("DEBUG: Plan updated successfully for Firm %s\n", firmID)

	c.Response().Header().Set("HX-Trigger", "closeModal")
	return SuperadminGetFirmsListHTMX(c)
}

// SuperadminToggleAddOnActiveHandler toggles the active status of an add-on
func SuperadminToggleAddOnActiveHandler(c echo.Context) error {
	id := c.Param("id")
	var addon models.PlanAddOn
	if err := db.DB.First(&addon, "id = ?", id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Add-on not found")
	}

	addon.IsActive = !addon.IsActive
	if err := db.DB.Save(&addon).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update add-on")
	}

	// Re-render the add-on row or the whole table - but here we return standard 200 OK
	// Ideally we should return the updated row.
	// For simplicity, let's redirect to the list which is already an HTMX optimized replacement?
	// The AddOnsPageHandler renders the full page.
	// We should probably just return the updated AddOnsPage or just the row if we had a partial.
	// Since we don't have a partial for a single row readily available as a standalone component without refactoring,
	// let's re-render the whole page or redirect. Active/Inactive usually just requires a simple toggle update.
	//
	// Better approach: Re-fetch list and render the page content again or a partial table if it existed.
	// But `AddOnsPage` is a full page.
	// Let's reload the page for now using HX-Refresh or similar, or just re-render the page function which includes the layout.
	// OR better: Create a Partial for the Addons Table?
	// For now, let's just trigger a full reload or use the existing page handler logic to return full HTML which HTMX can parse if `hx-select` is used,
	// OR just set HX-Refresh: true.

	c.Response().Header().Set("HX-Refresh", "true")
	return c.NoContent(http.StatusOK)
}
