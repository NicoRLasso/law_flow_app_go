package handlers

import (
	"fmt"
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/pages"
	"law_flow_app_go/templates/partials"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// ServicesPageHandler renders the services page
func ServicesPageHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)
	csrfToken := middleware.GetCSRFToken(c)

	// Note: pages.Services will be created in Phase 4
	// For now this assumes the function signature
	component := pages.Services(c.Request().Context(), "Services | LexLegal Cloud", csrfToken, user, firm)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// GetServicesHandler returns a list of services with filtering and pagination
func GetServicesHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)

	// Get query parameters for filtering
	filters := services.ServiceFilters{
		Status:        c.QueryParam("status"),
		ServiceTypeID: c.QueryParam("service_type_id"),
		ClientID:      c.QueryParam("client_id"),
		AssignedToID:  c.QueryParam("assigned_to_id"),
		Priority:      c.QueryParam("priority"),
		Keyword:       c.QueryParam("keyword"),
	}

	// Date filters
	if dateFrom := c.QueryParam("date_from"); dateFrom != "" {
		if t, err := time.Parse("2006-01-02", dateFrom); err == nil {
			filters.DateFrom = &t
		}
	}
	if dateTo := c.QueryParam("date_to"); dateTo != "" {
		if t, err := time.Parse("2006-01-02", dateTo); err == nil {
			// Add 24 hours to include the entire day
			t = t.Add(24 * time.Hour)
			filters.DateTo = &t
		}
	}

	// Role-based restrictions
	if currentUser.Role == "client" {
		filters.ClientID = currentUser.ID
	}
	// Lawyers see all firm services or just theirs?
	// Following case logic: Lawyers see assigned or all.
	// For now, let's allow lawyers to see all services in firm, filtering by assigned_to if needed.
	// If the requirement implies strict access control like cases, we might need to adjust.
	// Plan says "GetServicesByFirm", so we use that.

	// Pagination
	page := 1
	limit := 10
	if p, err := strconv.Atoi(c.QueryParam("page")); err == nil && p > 0 {
		page = p
	}
	if l, err := strconv.Atoi(c.QueryParam("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	servicesList, total, err := services.GetServicesByFirm(db.DB, currentFirm.ID, filters, page, limit)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch services")
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	// Check if HTMX request
	if c.Request().Header.Get("HX-Request") == "true" {
		// Note: partials.ServiceTable will be created in Phase 4
		component := partials.ServiceTable(c.Request().Context(), servicesList, page, totalPages, limit, int(total))
		return component.Render(c.Request().Context(), c.Response().Writer)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": servicesList,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// GetServiceDetailHandler renders the service detail page
func GetServiceDetailHandler(c echo.Context) error {
	id := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)

	service, err := services.GetServiceByID(db.DB, currentFirm.ID, id)
	if err != nil {
		if err == services.ErrServiceNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Service not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve service")
	}

	// Preload Milestones and Activities for timeline
	if err := db.DB.Where("service_id = ?", service.ID).Find(&service.Milestones).Error; err != nil {
		fmt.Printf("Error loading milestones: %v\n", err)
	}
	if err := db.DB.Where("service_id = ?", service.ID).Find(&service.Activities).Error; err != nil {
		fmt.Printf("Error loading activities: %v\n", err)
	}

	// Build timeline events (empty for now, will be loaded via HTMX)
	timeline := []models.TimelineEvent{}

	// Security check for clients
	if currentUser.Role == "client" && service.ClientID != currentUser.ID {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	csrfToken := middleware.GetCSRFToken(c)

	// Fetch expense categories for the modal
	var expenseCategories []models.ChoiceOption
	if err := db.DB.Joins("JOIN choice_categories cc ON cc.id = choice_options.category_id").
		Where("cc.key = ? AND choice_options.is_active = ?", models.ChoiceCategoryKeyExpenseCategory, true).
		Order("choice_options.label ASC").
		Find(&expenseCategories).Error; err != nil {
		// Log error but continue
		fmt.Printf("Error fetching expense categories: %v\n", err)
	}

	// Fetch total expenses for summary
	totalExpenses, _ := services.GetServiceTotalExpenses(db.DB, service.ID)

	// Note: pages.ServiceDetail update to include totalExpenses and timeline
	component := pages.ServiceDetail(c.Request().Context(), "Service Details | LexLegal Cloud", csrfToken, currentUser, currentFirm, service, expenseCategories, totalExpenses, timeline)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// GetServiceTimelineHandler returns timeline events for a service with pagination
func GetServiceTimelineHandler(c echo.Context) error {
	id := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)

	service, err := services.GetServiceByID(db.DB, currentFirm.ID, id)
	if err != nil {
		if err == services.ErrServiceNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Service not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve service")
	}

	// Security check for clients
	if currentUser.Role == "client" && service.ClientID != currentUser.ID {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	// Preload Milestones and Activities for timeline
	if err := db.DB.Where("service_id = ?", service.ID).Find(&service.Milestones).Error; err != nil {
		fmt.Printf("Error loading milestones: %v\n", err)
	}
	if err := db.DB.Where("service_id = ?", service.ID).Find(&service.Activities).Error; err != nil {
		fmt.Printf("Error loading activities: %v\n", err)
	}

	// Pagination
	page := 1
	limit := 6
	if p, err := strconv.Atoi(c.QueryParam("page")); err == nil && p > 0 {
		page = p
	}

	// Build timeline events
	allEvents := buildServiceTimeline(service)

	// Calculate pagination
	total := len(allEvents)
	totalPages := (total + limit - 1) / limit

	// Get page slice
	start := (page - 1) * limit
	end := start + limit
	if end > total {
		end = total
	}

	var events []models.TimelineEvent
	if start < total {
		events = allEvents[start:end]
	} else {
		events = []models.TimelineEvent{}
	}

	component := partials.ServiceTimelineList(c.Request().Context(), events, page, totalPages, total, id)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// buildServiceTimeline creates a sorted timeline of service events
func buildServiceTimeline(service *models.LegalService) []models.TimelineEvent {
	var events []models.TimelineEvent

	// Add service created event
	events = append(events, models.TimelineEvent{
		Date:        service.CreatedAt,
		Type:        "service_created",
		Title:       "Service Created",
		Description: "Service was created",
	})

	// Add service started event
	if service.StartedAt != nil {
		events = append(events, models.TimelineEvent{
			Date:        *service.StartedAt,
			Type:        "service_started",
			Title:       "Service Started",
			Description: "Service work began",
		})
	}

	// Add service completed event
	if service.CompletedAt != nil {
		events = append(events, models.TimelineEvent{
			Date:        *service.CompletedAt,
			Type:        "service_completed",
			Title:       "Service Completed",
			Description: "Service was completed",
			IsCompleted: true,
		})
	}

	// Add estimated due date as event
	if service.EstimatedDueDate != nil {
		events = append(events, models.TimelineEvent{
			Date:        *service.EstimatedDueDate,
			Type:        "estimated_due",
			Title:       "Estimated Due Date",
			Description: "Target completion date",
		})
	}

	// Add milestones
	for _, milestone := range service.Milestones {
		desc := ""
		if milestone.Description != nil {
			desc = *milestone.Description
		}
		dueDate := time.Now()
		if milestone.DueDate != nil {
			dueDate = *milestone.DueDate
		}
		events = append(events, models.TimelineEvent{
			Date:        dueDate,
			Type:        "milestone",
			Title:       milestone.Title,
			Description: desc,
			Status:      milestone.Status,
			IsCompleted: milestone.Status == "COMPLETED",
		})
	}

	// Add activities
	for _, activity := range service.Activities {
		if activity.OccurredAt != nil {
			events = append(events, models.TimelineEvent{
				Date:         *activity.OccurredAt,
				Type:         "activity",
				Title:        activity.Title,
				Description:  activity.Content,
				ActivityType: activity.ActivityType,
			})
		}
	}

	// Sort events by date (most recent first)
	sort.Slice(events, func(i, j int) bool {
		return events[i].Date.After(events[j].Date)
	})

	return events
}

// CreateServiceModalHandler renders the modal to create a new service
func CreateServiceModalHandler(c echo.Context) error {
	currentFirm := middleware.GetCurrentFirm(c)
	currentUser := middleware.GetCurrentUser(c)

	// Fetch clients
	var clients []models.User
	if err := db.DB.Where("firm_id = ? AND role = ?", currentFirm.ID, "client").Order("name ASC").Find(&clients).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch clients")
	}

	// Fetch lawyers (for assignment)
	var lawyers []models.User
	if err := db.DB.Where("firm_id = ? AND role IN (?, ?)", currentFirm.ID, "lawyer", "admin").Order("name ASC").Find(&lawyers).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch lawyers")
	}

	// Fetch service types
	var serviceTypes []models.ChoiceOption
	if err := db.DB.Joins("JOIN choice_categories cc ON cc.id = choice_options.category_id").
		Where("cc.key = ? AND choice_options.is_active = ?", models.ChoiceCategoryKeyServiceType, true).
		Order("choice_options.label ASC").
		Find(&serviceTypes).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch service types")
	}

	// Note: partials.ServiceCreateModal will be created in Phase 4
	component := partials.ServiceCreateModal(c.Request().Context(), currentUser, clients, lawyers, serviceTypes)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// CreateServiceHandler handles the creation of a new service
func CreateServiceHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)

	// Check subscription limits
	limitCheck, err := services.CanAddService(db.DB, currentFirm.ID)
	if err != nil {
		if err == services.ErrServiceLimitReached {
			return c.HTML(http.StatusForbidden, fmt.Sprintf("<div class='alert alert-error'>%s</div>", limitCheck.Message))
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check limits")
	}
	// Assuming LimitCheckResult has CanAdd boolean
	// Note: CanAddService returns LimitCheckResult which has Allowed or similar?
	// Based on handlers/case.go usage: limitResult.Message is used.

	// Parse form
	title := c.FormValue("title")
	clientID := c.FormValue("client_id")
	serviceTypeID := c.FormValue("service_type_id")
	description := c.FormValue("description")
	objective := c.FormValue("objective")
	assignedToID := c.FormValue("assigned_to_id")
	priority := c.FormValue("priority")

	if title == "" || clientID == "" || serviceTypeID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing required fields")
	}

	// Length Validation
	if len(title) > 255 {
		return echo.NewHTTPError(http.StatusBadRequest, "Title must be less than 255 characters")
	}
	if len(description) > 5000 {
		return echo.NewHTTPError(http.StatusBadRequest, "Description must be less than 5000 characters")
	}
	if len(objective) > 5000 {
		return echo.NewHTTPError(http.StatusBadRequest, "Objective must be less than 5000 characters")
	}

	// Generate number
	serviceNumber, err := services.EnsureUniqueServiceNumber(db.DB, currentFirm.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate service number")
	}

	// Create service
	service := models.LegalService{
		FirmID:        currentFirm.ID,
		ServiceNumber: serviceNumber,
		Title:         title,
		Description:   description,
		ServiceTypeID: &serviceTypeID,
		ClientID:      clientID,
		Objective:     objective,
		Status:        models.ServiceStatusIntake, // Default status
		AssignedToID:  nil,
		Priority:      models.ServicePriorityNormal,
	}

	if assignedToID != "" {
		service.AssignedToID = &assignedToID
	}
	if priority != "" && models.IsValidServicePriority(priority) {
		service.Priority = priority
	}

	now := time.Now()
	service.StatusChangedAt = &now
	service.StatusChangedBy = &currentUser.ID

	tx := db.DB.Begin()
	if err := tx.Create(&service).Error; err != nil {
		tx.Rollback()
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create service")
	}

	// Create default milestones
	if err := services.CreateDefaultMilestones(tx, &service); err != nil {
		tx.Rollback()
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create milestones")
	}

	tx.Commit()

	// Audit Log
	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(db.DB, auditCtx, models.AuditActionCreate,
		"LegalService", service.ID, service.ServiceNumber,
		"Service created", nil, service)

	// Redirect or return success
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Redirect", "/services/"+service.ID)
		return c.NoContent(http.StatusOK)
	}

	return c.Redirect(http.StatusSeeOther, "/services/"+service.ID)
}

// GetUpdateServiceModalHandler renders the modal to edit a service
func GetUpdateServiceModalHandler(c echo.Context) error {
	id := c.Param("id")
	currentFirm := middleware.GetCurrentFirm(c)
	currentUser := middleware.GetCurrentUser(c)

	service, err := services.GetServiceByID(db.DB, currentFirm.ID, id)
	if err != nil {
		if err == services.ErrServiceNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Service not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve service")
	}

	// Fetch clients
	var clients []models.User
	if err := db.DB.Where("firm_id = ? AND role = ?", currentFirm.ID, "client").Order("name ASC").Find(&clients).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch clients")
	}

	// Fetch lawyers (for assignment)
	var lawyers []models.User
	if err := db.DB.Where("firm_id = ? AND role IN (?, ?)", currentFirm.ID, "lawyer", "admin").Order("name ASC").Find(&lawyers).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch lawyers")
	}

	// Fetch service types
	var serviceTypes []models.ChoiceOption
	if err := db.DB.Joins("JOIN choice_categories cc ON cc.id = choice_options.category_id").
		Where("cc.key = ? AND choice_options.is_active = ?", models.ChoiceCategoryKeyServiceType, true).
		Order("choice_options.label ASC").
		Find(&serviceTypes).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch service types")
	}

	component := partials.ServiceEditModal(c.Request().Context(), currentUser, service, clients, lawyers, serviceTypes)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// DeleteServiceConfirmHandler renders the deletion confirmation modal
func DeleteServiceConfirmHandler(c echo.Context) error {
	id := c.Param("id")
	currentFirm := middleware.GetCurrentFirm(c)

	service, err := services.GetServiceByID(db.DB, currentFirm.ID, id)
	if err != nil {
		if err == services.ErrServiceNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Service not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve service")
	}

	component := partials.ServiceDeleteConfirmModal(c.Request().Context(), *service)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// DeleteServiceHandler handles the deletion of a service
func DeleteServiceHandler(c echo.Context) error {
	id := c.Param("id")
	currentFirm := middleware.GetCurrentFirm(c)

	service, err := services.GetServiceByID(db.DB, currentFirm.ID, id)
	if err != nil {
		if err == services.ErrServiceNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Service not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve service")
	}

	if err := services.DeleteService(db.DB, currentFirm.ID, id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete service")
	}

	// Audit Log
	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(db.DB, auditCtx, models.AuditActionDelete,
		"LegalService", service.ID, service.ServiceNumber,
		"Service deleted", nil, nil)

	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Trigger", "reload-services")
		return c.NoContent(http.StatusOK)
	}

	return c.Redirect(http.StatusSeeOther, "/services")
}

// UpdateServiceHandler updates service details
func UpdateServiceHandler(c echo.Context) error {
	id := c.Param("id")
	currentFirm := middleware.GetCurrentFirm(c)

	var service models.LegalService
	if err := db.DB.Where("firm_id = ? AND id = ?", currentFirm.ID, id).First(&service).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Service not found")
	}

	// Parse fields
	service.Title = c.FormValue("title")
	service.Description = c.FormValue("description")
	service.Objective = c.FormValue("objective")

	// Length Validation
	if len(service.Title) > 255 {
		return echo.NewHTTPError(http.StatusBadRequest, "Title must be less than 255 characters")
	}
	if len(service.Description) > 5000 {
		return echo.NewHTTPError(http.StatusBadRequest, "Description must be less than 5000 characters")
	}
	if len(service.Objective) > 5000 {
		return echo.NewHTTPError(http.StatusBadRequest, "Objective must be less than 5000 characters")
	}

	priority := c.FormValue("priority")
	if priority != "" && models.IsValidServicePriority(priority) {
		service.Priority = priority
	}

	assignedToID := c.FormValue("assigned_to_id")
	if assignedToID != "" {
		service.AssignedToID = &assignedToID
	} else {
		service.AssignedToID = nil
	}

	// Handle other fields like dates if present in form
	// ...

	if err := db.DB.Save(&service).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update service")
	}

	// Audit Log
	// ...

	// Return updated view or redirect
	if c.Request().Header.Get("HX-Request") == "true" {
		// Return specific part causing update or full page?
		// Typically return the detail view or form
		c.Response().Header().Set("HX-Refresh", "true")
		return c.NoContent(http.StatusOK)
	}

	return c.JSON(http.StatusOK, service)
}

// UpdateServiceStatusHandler updates the service status
func UpdateServiceStatusHandler(c echo.Context) error {
	id := c.Param("id")
	status := c.FormValue("status")
	currentFirm := middleware.GetCurrentFirm(c)
	currentUser := middleware.GetCurrentUser(c)

	var service models.LegalService
	if err := db.DB.Where("firm_id = ? AND id = ?", currentFirm.ID, id).First(&service).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Service not found")
	}

	if err := services.UpdateServiceStatus(db.DB, id, status, currentUser.ID); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Audit Log
	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(db.DB, auditCtx, models.AuditActionUpdate,
		"LegalService", service.ID, service.ServiceNumber,
		"Status updated to "+status, nil, nil)

	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Refresh", "true")
		return c.NoContent(http.StatusOK)
	}

	return c.JSON(http.StatusOK, map[string]string{"status": status})
}

// GetServiceHandler returns a single service as JSON
func GetServiceHandler(c echo.Context) error {
	serviceID := c.Param("id")
	firm := c.Get("firm").(*models.Firm)

	service, err := services.GetServiceByID(db.DB, firm.ID, serviceID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Service not found")
	}

	return c.JSON(http.StatusOK, service)
}
