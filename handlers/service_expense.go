package handlers

import (
	"fmt"
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/partials"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// GetServiceExpensesHandler returns the list of expenses for a service
func GetServiceExpensesHandler(c echo.Context) error {
	serviceID := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)

	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	limit := 10
	offset := (page - 1) * limit

	var expenses []*models.ServiceExpense
	var total int64

	query := db.DB.Where("firm_id = ? AND service_id = ?", currentFirm.ID, serviceID)

	if err := query.Model(&models.ServiceExpense{}).Count(&total).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to count expenses")
	}

	if err := query.Preload("Category").
		Order("incurred_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&expenses).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch expenses")
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}

	var totalAmount float64
	if err := db.DB.Model(&models.ServiceExpense{}).
		Where("firm_id = ? AND service_id = ? AND status != ?", currentFirm.ID, serviceID, models.ExpenseStatusRejected).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalAmount).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to calculate total amount")
	}

	component := partials.ServiceExpenseTable(c.Request().Context(), expenses, page, totalPages, limit, int(total), totalAmount, serviceID, currentUser.Role != "client")
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// CreateServiceExpenseHandler adds a new expense
func CreateServiceExpenseHandler(c echo.Context) error {
	serviceID := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)

	description := c.FormValue("description")
	amountStr := c.FormValue("amount")
	categoryID := c.FormValue("category_id")
	dateStr := c.FormValue("incurred_at")

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid amount")
	}

	date := time.Now()
	if dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			date = t
		}
	}

	// Only set CategoryID if a valid value was provided
	var categoryIDPtr *string
	if categoryID != "" {
		categoryIDPtr = &categoryID
	}

	expense := models.ServiceExpense{
		FirmID:            currentFirm.ID,
		ServiceID:         serviceID,
		Description:       description,
		Amount:            amount,
		Currency:          currentFirm.Currency,
		ExpenseCategoryID: categoryIDPtr,
		IncurredAt:        date,
		Status:            models.ExpenseStatusPending,
		RecordedByID:      currentUser.ID,
	}

	if err := db.DB.Create(&expense).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create expense")
	}

	// Audit
	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(db.DB, auditCtx, models.AuditActionCreate,
		"ServiceExpense", expense.ID, fmt.Sprintf("%.2f", amount),
		"Expense recorded", nil, expense)

	return GetServiceExpensesHandler(c)
}

// UpdateServiceExpenseHandler updates expense
func UpdateServiceExpenseHandler(c echo.Context) error {
	serviceID := c.Param("id")
	expenseID := c.Param("eid")
	currentFirm := middleware.GetCurrentFirm(c)

	var expense models.ServiceExpense
	if err := db.DB.Where("firm_id = ? AND id = ? AND service_id = ?", currentFirm.ID, expenseID, serviceID).First(&expense).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Expense not found")
	}

	expense.Description = c.FormValue("description")
	if amt, err := strconv.ParseFloat(c.FormValue("amount"), 64); err == nil {
		expense.Amount = amt
	}

	categoryID := c.FormValue("category_id")
	if categoryID != "" {
		expense.ExpenseCategoryID = &categoryID
	} else {
		expense.ExpenseCategoryID = nil
	}

	if dateStr := c.FormValue("incurred_at"); dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			expense.IncurredAt = t
		}
	}

	if err := db.DB.Save(&expense).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update expense")
	}

	return GetServiceExpensesHandler(c)
}

// ApproveServiceExpenseHandler approves/rejects expense
func ApproveServiceExpenseHandler(c echo.Context) error {
	serviceID := c.Param("id")
	expenseID := c.Param("eid")
	currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)

	action := c.FormValue("action") // "approve" or "reject" or "pay"

	var expense models.ServiceExpense
	if err := db.DB.Where("firm_id = ? AND id = ? AND service_id = ?", currentFirm.ID, expenseID, serviceID).First(&expense).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Expense not found")
	}

	now := time.Now()
	if action == "approve" {
		expense.Status = models.ExpenseStatusApproved
		expense.ApprovedAt = &now
		expense.ApprovedBy = &currentUser.ID
	} else if action == "reject" {
		expense.Status = models.ExpenseStatusRejected
	} else if action == "pay" {
		expense.Status = models.ExpenseStatusPaid
	}

	if err := db.DB.Save(&expense).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update status")
	}

	return GetServiceExpensesHandler(c)
}

// DeleteServiceExpenseHandler deletes expense
func DeleteServiceExpenseHandler(c echo.Context) error {
	serviceID := c.Param("id")
	expenseID := c.Param("eid")
	currentFirm := middleware.GetCurrentFirm(c)

	if err := db.DB.Where("firm_id = ? AND id = ? AND service_id = ?", currentFirm.ID, expenseID, serviceID).Delete(&models.ServiceExpense{}).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete expense")
	}

	return GetServiceExpensesHandler(c)
}

// GetServiceExpenseEditModalHandler returns the edit modal for an expense
func GetServiceExpenseEditModalHandler(c echo.Context) error {
	serviceID := c.Param("id")
	expenseID := c.Param("eid")
	currentFirm := middleware.GetCurrentFirm(c)

	var expense models.ServiceExpense
	if err := db.DB.Where("firm_id = ? AND id = ? AND service_id = ?", currentFirm.ID, expenseID, serviceID).First(&expense).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Expense not found")
	}

	// Fetch categories for the dropdown
	var categories []models.ChoiceOption
	if err := db.DB.Joins("JOIN choice_categories cc ON cc.id = choice_options.category_id").
		Where("cc.key = ? AND choice_options.is_active = ?", models.ChoiceCategoryKeyExpenseCategory, true).
		Order("choice_options.label ASC").
		Find(&categories).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch categories")
	}

	component := partials.EditExpenseModal(c.Request().Context(), expense, categories)
	return component.Render(c.Request().Context(), c.Response().Writer)
}
