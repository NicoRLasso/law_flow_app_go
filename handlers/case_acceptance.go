package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"law_flow_app_go/config"
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/partials"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

// StartCaseAcceptanceHandler renders the initial acceptance modal (Step 1)
func StartCaseAcceptanceHandler(c echo.Context) error {
	id := c.Param("id")

	// Fetch request with firm-scoping
	var request models.CaseRequest
	query := middleware.GetFirmScopedQuery(c, db.DB)
	if err := query.Preload("DocumentTypeOption").First(&request, "id = ?", id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Request not found")
	}

	// Verify status is pending
	if request.Status != models.StatusPending {
		return echo.NewHTTPError(http.StatusBadRequest, "Only pending requests can be accepted")
	}

	// Render the acceptance modal
	component := partials.CaseAcceptanceModal(c.Request().Context(), request)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// ProcessClientStepHandler checks if client exists and returns Step 2
func ProcessClientStepHandler(c echo.Context) error {
	id := c.Param("id")
	firm := middleware.GetCurrentFirm(c)

	// Parse client role from form (required)
	clientRole := strings.TrimSpace(c.FormValue("client_role"))
	if clientRole == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Client role selection is required")
	}
	if !models.IsValidClientRole(clientRole) {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid client role selection")
	}

	// Fetch request
	var request models.CaseRequest
	query := middleware.GetFirmScopedQuery(c, db.DB)
	if err := query.First(&request, "id = ?", id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Request not found")
	}

	// Check if client email exists
	var existingClient models.User
	isNewClient := true
	err := db.DB.Where("firm_id = ? AND email = ? AND role = ?", firm.ID, request.Email, "client").
		First(&existingClient).Error
	if err == nil {
		isNewClient = false
	}

	// Fetch active lawyers for the firm
	var lawyers []models.User
	c.Logger().Infof("Fetching lawyers for firm_id: %s", firm.ID)
	if err := db.DB.Where("firm_id = ? AND is_active = ? AND role IN ?", firm.ID, true, []string{"lawyer", "admin"}).
		Order("name ASC").
		Find(&lawyers).Error; err != nil {
		c.Logger().Errorf("Error fetching lawyers: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch lawyers")
	}
	c.Logger().Infof("Found %d lawyers", len(lawyers))

	// Render Step 2
	component := partials.LawyerSelectionStep(c.Request().Context(), lawyers, isNewClient, request, clientRole)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// GetLawyerListHandler returns JSON list of active lawyers
func GetLawyerListHandler(c echo.Context) error {
	firm := middleware.GetCurrentFirm(c)

	var lawyers []models.User
	if err := db.DB.Where("firm_id = ? AND is_active = ? AND role IN ?", firm.ID, true, []string{"lawyer", "admin"}).
		Select("id", "name", "email", "role").
		Order("name ASC").
		Find(&lawyers).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch lawyers")
	}

	return c.JSON(http.StatusOK, lawyers)
}

// AssignLawyerStepHandler validates lawyer and returns Step 3
func AssignLawyerStepHandler(c echo.Context) error {
	id := c.Param("id")
	firm := middleware.GetCurrentFirm(c)

	// Parse lawyer ID from form
	lawyerID := strings.TrimSpace(c.FormValue("lawyer_id"))
	if lawyerID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Lawyer selection is required")
	}

	// Parse client role from form (passed from previous step)
	clientRole := strings.TrimSpace(c.FormValue("client_role"))
	if clientRole == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Client role is required")
	}

	// Validate lawyer exists and is active
	var lawyer models.User
	if err := db.DB.Where("id = ? AND firm_id = ? AND is_active = ? AND role IN ?",
		lawyerID, firm.ID, true, []string{"lawyer", "admin"}).
		First(&lawyer).Error; err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid lawyer selection")
	}

	// Fetch request
	var request models.CaseRequest
	query := middleware.GetFirmScopedQuery(c, db.DB)
	if err := query.First(&request, "id = ?", id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Request not found")
	}

	// Fetch classification options for the firm's country
	var domains []models.CaseDomain
	if err := db.DB.Where("firm_id = ? AND is_active = ?", firm.ID, true).
		Order("`order` ASC, name ASC").
		Find(&domains).Error; err != nil {
		c.Logger().Errorf("Failed to fetch domains: %v", err)
	}

	// Render Step 3
	component := partials.ClassificationStep(c.Request().Context(), domains, request, lawyerID, clientRole)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// GetClassificationOptionsHandler returns JSON of classification options
func GetClassificationOptionsHandler(c echo.Context) error {
	firm := middleware.GetCurrentFirm(c)
	domainID := c.QueryParam("domain_id")
	branchID := c.QueryParam("branch_id")

	response := make(map[string]interface{})

	// Fetch domains if no domain_id provided
	if domainID == "" {
		var domains []models.CaseDomain
		if err := db.DB.Where("firm_id = ? AND is_active = ?", firm.ID, true).
			Order("`order` ASC, name ASC").
			Find(&domains).Error; err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch domains")
		}
		response["domains"] = domains
		return c.JSON(http.StatusOK, response)
	}

	// Fetch branches for domain
	if branchID == "" {
		var branches []models.CaseBranch
		if err := db.DB.Where("domain_id = ? AND is_active = ?", domainID, true).
			Order("`order` ASC, name ASC").
			Find(&branches).Error; err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch branches")
		}
		response["branches"] = branches
		return c.JSON(http.StatusOK, response)
	}

	// Fetch subtypes for branch
	var subtypes []models.CaseSubtype
	if err := db.DB.Where("branch_id = ? AND is_active = ?", branchID, true).
		Order("`order` ASC, name ASC").
		Find(&subtypes).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch subtypes")
	}
	response["subtypes"] = subtypes

	return c.JSON(http.StatusOK, response)
}

// SaveClassificationStepHandler saves classification and returns Step 4
func SaveClassificationStepHandler(c echo.Context) error {
	id := c.Param("id")
	firm := middleware.GetCurrentFirm(c)

	// Parse optional classification data
	domainID := strings.TrimSpace(c.FormValue("domain_id"))
	branchID := strings.TrimSpace(c.FormValue("branch_id"))
	lawyerID := strings.TrimSpace(c.FormValue("lawyer_id"))
	clientRole := strings.TrimSpace(c.FormValue("client_role"))
	subtypeIDs := c.Request().Form["subtype_ids[]"]

	// Validate client role
	if clientRole == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Client role is required")
	}

	// Validate classification IDs if provided
	if domainID != "" {
		var domain models.CaseDomain
		if err := db.DB.Where("id = ? AND firm_id = ?", domainID, firm.ID).First(&domain).Error; err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid domain selection")
		}
	}

	if branchID != "" {
		var branch models.CaseBranch
		if err := db.DB.Joins("JOIN case_domains ON case_domains.id = case_branches.domain_id").
			Where("case_branches.id = ? AND case_domains.firm_id = ?", branchID, firm.ID).
			First(&branch).Error; err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid branch selection")
		}
	}

	// Fetch request
	var request models.CaseRequest
	query := middleware.GetFirmScopedQuery(c, db.DB)
	if err := query.First(&request, "id = ?", id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Request not found")
	}

	// Fetch lawyer
	var lawyer models.User
	if err := db.DB.Where("id = ? AND firm_id = ?", lawyerID, firm.ID).First(&lawyer).Error; err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid lawyer")
	}

	// Check if client exists
	var existingClient models.User
	isNewClient := true
	err := db.DB.Where("firm_id = ? AND email = ? AND role = ?", firm.ID, request.Email, "client").
		First(&existingClient).Error
	if err == nil {
		isNewClient = false
	}

	// Fetch classification details if provided
	var domain *models.CaseDomain
	var branch *models.CaseBranch
	var subtypes []models.CaseSubtype

	if domainID != "" {
		domain = &models.CaseDomain{}
		db.DB.First(domain, "id = ?", domainID)
	}

	if branchID != "" {
		branch = &models.CaseBranch{}
		db.DB.First(branch, "id = ?", branchID)
	}

	if len(subtypeIDs) > 0 {
		db.DB.Where("id IN ?", subtypeIDs).Find(&subtypes)
	}

	// Render Step 4 (Review)
	component := partials.ReviewConfirmStep(c.Request().Context(), request, lawyer, isNewClient, domain, branch, subtypes, domainID, branchID, subtypeIDs, lawyerID, clientRole)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// FinalizeCaseCreationHandler executes the transaction to create the case
func FinalizeCaseCreationHandler(c echo.Context) error {
	id := c.Param("id")
	firm := middleware.GetCurrentFirm(c)
	currentUser := middleware.GetCurrentUser(c)
	cfg := c.Get("config").(*config.Config)

	// Parse form data
	lawyerID := strings.TrimSpace(c.FormValue("lawyer_id"))
	clientRole := strings.TrimSpace(c.FormValue("client_role"))
	domainID := strings.TrimSpace(c.FormValue("domain_id"))
	branchID := strings.TrimSpace(c.FormValue("branch_id"))
	subtypeIDs := c.Request().Form["subtype_ids[]"]

	if lawyerID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Lawyer assignment is required")
	}

	if clientRole == "" || !models.IsValidClientRole(clientRole) {
		return echo.NewHTTPError(http.StatusBadRequest, "Valid client role is required")
	}

	// Begin transaction
	tx := db.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Fetch request with lock
	var request models.CaseRequest
	if err := tx.Where("id = ? AND firm_id = ?", id, firm.ID).First(&request).Error; err != nil {
		tx.Rollback()
		return echo.NewHTTPError(http.StatusNotFound, "Request not found")
	}

	// Verify still pending
	if request.Status != models.StatusPending {
		tx.Rollback()
		return echo.NewHTTPError(http.StatusBadRequest, "Request has already been processed")
	}

	// Check if client exists or create new one
	var client models.User
	isNewClient := false
	err := tx.Where("firm_id = ? AND email = ? AND role = ?", firm.ID, request.Email, "client").
		First(&client).Error

	if err != nil {
		// Create new client user
		isNewClient = true

		// Generate random password
		randomBytes := make([]byte, 32)
		if _, err := rand.Read(randomBytes); err != nil {
			tx.Rollback()
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate password")
		}
		randomPassword := base64.URLEncoding.EncodeToString(randomBytes)

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(randomPassword), bcrypt.DefaultCost)
		if err != nil {
			tx.Rollback()
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to hash password")
		}

		client = models.User{
			Name:     request.Name,
			Email:    request.Email,
			Password: string(hashedPassword),
			FirmID:   &firm.ID,
			Role:     "client",
			IsActive: true,
		}

		// Set phone and document if available
		if request.Phone != "" {
			client.PhoneNumber = &request.Phone
		}
		if request.DocumentNumber != "" {
			client.DocumentNumber = &request.DocumentNumber
		}
		if request.DocumentTypeID != nil {
			client.DocumentTypeID = request.DocumentTypeID
		}

		if err := tx.Create(&client).Error; err != nil {
			tx.Rollback()
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create client user")
		}
	}

	// Generate unique case number
	caseNumber, err := services.EnsureUniqueCaseNumber(tx, firm.ID)
	if err != nil {
		tx.Rollback()
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to generate case number: %v", err))
	}

	// Create case
	newCase := models.Case{
		FirmID:               firm.ID,
		ClientID:             client.ID,
		AssignedToID:         &lawyerID,
		CaseNumber:           caseNumber,
		CaseType:             "General", // Default type
		Description:          request.Description,
		Status:               models.CaseStatusOpen,
		ClientRole:           &clientRole,
		CreatedFromRequestID: &request.ID,
	}

	// Set optional classification
	if domainID != "" {
		newCase.DomainID = &domainID
	}
	if branchID != "" {
		newCase.BranchID = &branchID
	}

	if err := tx.Create(&newCase).Error; err != nil {
		tx.Rollback()
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create case")
	}

	// Link subtypes if provided
	if len(subtypeIDs) > 0 {
		var subtypes []models.CaseSubtype
		if err := tx.Where("id IN ?", subtypeIDs).Find(&subtypes).Error; err != nil {
			tx.Rollback()
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch subtypes")
		}

		if err := tx.Model(&newCase).Association("Subtypes").Append(subtypes); err != nil {
			tx.Rollback()
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to link subtypes")
		}
	}

	// Update request status
	request.Status = models.StatusAccepted
	request.ReviewedByID = &currentUser.ID
	reviewTime := time.Now()
	request.ReviewedAt = &reviewTime

	if err := tx.Save(&request).Error; err != nil {
		tx.Rollback()
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update request status")
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to commit transaction")
	}

	// Transfer document from request to case (outside transaction, non-critical)
	if err := services.TransferRequestDocumentToCase(db.DB, &request, newCase.ID, currentUser.ID); err != nil {
		// Log error but don't fail the request
		c.Logger().Errorf("Failed to transfer document: %v", err)
	}

	// Send emails asynchronously (don't block on errors)
	go func() {
		// Send welcome email to new clients
		if isNewClient {
			welcomeEmail := services.BuildCaseAcceptanceEmail(client.Email, client.Name, firm.Name, caseNumber)
			services.SendEmailAsync(cfg, welcomeEmail)
		}

		// Send assignment email to lawyer
		var lawyer models.User
		if err := db.DB.First(&lawyer, "id = ?", lawyerID).Error; err == nil {
			assignmentEmail := services.BuildLawyerAssignmentEmail(lawyer.Email, lawyer.Name, caseNumber, client.Name)
			services.SendEmailAsync(cfg, assignmentEmail)
		}
	}()

	// Return success response
	if c.Request().Header.Get("HX-Request") == "true" {
		// Trigger page reload via HX-Redirect
		c.Response().Header().Set("HX-Redirect", "/case-requests")
		return c.NoContent(http.StatusOK)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":     "Case created successfully",
		"case_number": caseNumber,
		"case_id":     newCase.ID,
	})
}

// CancelAcceptanceHandler clears any session data (placeholder for now)
func CancelAcceptanceHandler(c echo.Context) error {
	// Currently we're not using sessions, but this is here for future implementation
	// if we decide to store intermediate state in sessions
	return c.NoContent(http.StatusNoContent)
}
