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
