package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/services"
	"net/http"

	"github.com/labstack/echo/v4"
)

// PurchaseAddOnHandler handles purchasing an add-on
func PurchaseAddOnHandler(c echo.Context) error {
	firm := middleware.GetCurrentFirm(c)
	addOnID := c.FormValue("addon_id")
	quantity := 1 // For now, handle 1 unit. Extension: allow quantity in form.

	if addOnID == "" {
		return c.HTML(http.StatusBadRequest, `<div class="text-error text-sm">Add-on ID is required</div>`)
	}

	err := services.PurchaseAddOn(db.DB, firm.ID, addOnID, quantity, &middleware.GetCurrentUser(c).ID)
	if err != nil {
		c.Logger().Errorf("Failed to purchase add-on %s for firm %s: %v", addOnID, firm.ID, err)
		return c.HTML(http.StatusInternalServerError, `<div class="text-error text-sm">Failed to purchase add-on. Please try again.</div>`)
	}

	// Recalculate usage to ensure limits are updated
	services.RecalculateFirmUsage(db.DB, firm.ID)

	// Refresh via trigger only to avoid race conditions with OOB swaps
	c.Response().Header().Set("HX-Trigger", "subscriptionUpdated")

	return c.HTML(http.StatusOK, `<div class="alert alert-success mt-4">
		<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
		<span>Add-on purchased successfully!</span>
	</div>`)
}

// CancelAddOnHandler handles canceling a recurring add-on
func CancelAddOnHandler(c echo.Context) error {
	firm := middleware.GetCurrentFirm(c)
	firmAddOnID := c.Param("id")

	if firmAddOnID == "" {
		return c.HTML(http.StatusBadRequest, `<div class="text-error text-sm">Firm Add-on ID is required</div>`)
	}

	err := services.CancelAddOn(db.DB, firmAddOnID)
	if err != nil {
		c.Logger().Errorf("Failed to cancel add-on %s for firm %s: %v", firmAddOnID, firm.ID, err)
		return c.HTML(http.StatusInternalServerError, `<div class="text-error text-sm">Failed to cancel add-on.</div>`)
	}

	// Recalculate usage
	services.RecalculateFirmUsage(db.DB, firm.ID)

	// Refresh via trigger.
	// We return NoContent because the BillingTab will refresh itself upon hearing the event.
	c.Response().Header().Set("HX-Trigger", "subscriptionUpdated")
	return c.NoContent(http.StatusOK)
}
