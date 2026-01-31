package handlers

import (
	"encoding/json"
	"law_flow_app_go/models"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCasesPageHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-c1", Name: "Case Firm"}
	database.Create(firm)

	admin := &models.User{ID: "admin-c1", Name: "Admin", Email: "admin-c1@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)

	t.Run("Success", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/cases", nil)
		c.Set("user", admin)
		c.Set("firm", firm)

		err := CasesPageHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Cases")
	})
}

func TestGetCasesHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-c2", Name: "Case Firm 2"}
	database.Create(firm)

	admin := &models.User{ID: "admin-c2", Name: "Admin 2", Email: "admin-c2@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)

	client := &models.User{ID: "client-c2", Name: "Client 2", Email: "client-c2@test.com", FirmID: stringToPtr(firm.ID), Role: "client"}
	database.Create(client)

	lawyer := &models.User{ID: "lawyer-c2", Name: "Lawyer 2", Email: "lawyer-c2@test.com", FirmID: stringToPtr(firm.ID), Role: "lawyer"}
	database.Create(lawyer)

	// Create some cases
	database.Create(&models.Case{
		ID:         "case-c2-1",
		FirmID:     firm.ID,
		CaseNumber: "CASE-C2-1",
		Status:     models.CaseStatusOpen,
		ClientID:   client.ID,
		OpenedAt:   time.Now(),
	})

	database.Create(&models.Case{
		ID:           "case-c2-2",
		FirmID:       firm.ID,
		CaseNumber:   "CASE-C2-2",
		Status:       models.CaseStatusOpen,
		ClientID:     client.ID,
		AssignedToID: stringToPtr(lawyer.ID),
		OpenedAt:     time.Now(),
	})

	t.Run("Admin sees all", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/api/cases", nil)
		c.Set("user", admin)
		c.Set("firm", firm)

		err := GetCasesHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		data := resp["data"].([]interface{})
		assert.Len(t, data, 2)
	})

	t.Run("Client sees own", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/api/cases", nil)
		c.Set("user", client)
		c.Set("firm", firm)

		err := GetCasesHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		data := resp["data"].([]interface{})
		assert.Len(t, data, 2) // Both cases belong to this client
	})

	t.Run("Lawyer sees assigned", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/api/cases", nil)
		c.Set("user", lawyer)
		c.Set("firm", firm)

		err := GetCasesHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		data := resp["data"].([]interface{})
		assert.Len(t, data, 1) // Only case 2 is assigned to lawyer
	})

	t.Run("Filters for Admin", func(t *testing.T) {
		q := url.Values{}
		q.Add("status", models.CaseStatusOpen)
		q.Add("keyword", "CASE-C2-1")

		_, c, rec := setupEcho(http.MethodGet, "/api/cases?"+q.Encode(), nil)
		c.Set("user", admin)
		c.Set("firm", firm)

		err := GetCasesHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		data := resp["data"].([]interface{})
		assert.Len(t, data, 1)
	})
}

func TestCreateCaseHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-c3", Name: "Case Firm 3"}
	database.Create(firm)

	plan := &models.Plan{ID: "plan-pro", MaxCases: 10}
	database.Create(plan)
	database.Create(&models.FirmSubscription{FirmID: firm.ID, PlanID: plan.ID, Status: "active"})
	database.Create(&models.FirmUsage{FirmID: firm.ID, CurrentCases: 0})

	admin := &models.User{ID: "admin-c3", Name: "Admin 3", Email: "admin3-c@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)

	client := &models.User{ID: "client-c3", Name: "Client 3", Email: "client3-c@test.com", FirmID: stringToPtr(firm.ID), Role: "client"}
	database.Create(client)

	domain := &models.CaseDomain{ID: "domain-1", FirmID: firm.ID, Name: "Domain 1", IsActive: true}
	database.Create(domain)
	branch := &models.CaseBranch{ID: "branch-1", FirmID: firm.ID, DomainID: domain.ID, Name: "Branch 1", IsActive: true}
	database.Create(branch)

	t.Run("Success", func(t *testing.T) {
		f := url.Values{}
		f.Add("client_id", client.ID)
		f.Add("client_role", "demandante")
		f.Add("description", "New case description")
		f.Add("domain_id", domain.ID)
		f.Add("branch_id", branch.ID)
		f.Add("assigned_to_id", admin.ID)
		f.Add("filing_number", "12345")

		_, c, rec := setupEcho(http.MethodPost, "/api/cases", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", admin)
		c.Set("firm", firm)

		err := CreateCaseHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
func TestGetCaseDetailHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-cd1", Name: "Detail Firm"}
	database.Create(firm)
	admin := &models.User{ID: "admin-cd1", Name: "Admin", Email: "admin-cd1@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)
	client := &models.User{ID: "client-cd1", Name: "Client", Email: "client-cd1@test.com", FirmID: stringToPtr(firm.ID), Role: "client"}
	database.Create(client)

	caseRecord := &models.Case{
		ID:         "case-cd1",
		FirmID:     firm.ID,
		CaseNumber: "CASE-CD1",
		ClientID:   client.ID,
		OpenedAt:   time.Now(),
	}
	database.Create(caseRecord)

	t.Run("Success", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/cases/case-cd1", nil)
		c.SetParamNames("id")
		c.SetParamValues("case-cd1")
		c.Set("user", admin)
		c.Set("firm", firm)

		err := GetCaseDetailHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "CASE-CD1")
	})
}

func TestGetLawyersForFilterHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-lf1", Name: "Lawyer Filter Firm"}
	database.Create(firm)
	lawyer := &models.User{ID: "lawyer-lf1", Name: "Lawyer", Email: "lawyer-lf1@test.com", FirmID: stringToPtr(firm.ID), Role: "lawyer", IsActive: true}
	database.Create(lawyer)

	t.Run("Success", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/api/cases/lawyers", nil)
		c.Set("user", lawyer)
		c.Set("firm", firm)

		err := GetLawyersForFilterHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var users []models.User
		json.Unmarshal(rec.Body.Bytes(), &users)
		assert.Len(t, users, 1)
	})
}

func TestGetCaseDocumentsHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-doc1", Name: "Doc Firm"}
	database.Create(firm)
	admin := &models.User{ID: "admin-doc1", Name: "Admin", Email: "admin-doc1@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)
	client := &models.User{ID: "client-doc1", Name: "Client", Email: "client-doc1@test.com", FirmID: stringToPtr(firm.ID), Role: "client"}
	database.Create(client)

	caseRecord := &models.Case{ID: "case-doc1", FirmID: firm.ID, CaseNumber: "CASE-DOC1", ClientID: client.ID, OpenedAt: time.Now()}
	database.Create(caseRecord)

	database.Create(&models.CaseDocument{
		ID:               "doc-1",
		FirmID:           firm.ID,
		CaseID:           stringToPtr(caseRecord.ID),
		FileOriginalName: "test.pdf",
		UploadedByID:     stringToPtr(admin.ID),
	})

	t.Run("Success", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/api/cases/case-doc1/documents", nil)
		c.SetParamNames("id")
		c.SetParamValues("case-doc1")
		c.Set("user", admin)
		c.Set("firm", firm)

		err := GetCaseDocumentsHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		data := resp["data"].([]interface{})
		assert.Len(t, data, 1)
	})
}

func TestCreateCaseModalHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-modal1", Name: "Modal Firm"}
	database.Create(firm)
	admin := &models.User{ID: "admin-modal1", Name: "Admin", Email: "admin-m1@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)

	t.Run("Success", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/cases/new", nil)
		c.Set("user", admin)
		c.Set("firm", firm)

		err := CreateCaseModalHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "modal")
	})
}

func TestToggleDocumentVisibilityHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-tv1", Name: "Toggle Firm"}
	database.Create(firm)
	admin := &models.User{ID: "admin-tv1", Name: "Admin", Email: "admin-tv1@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)

	caseRecord := &models.Case{ID: "case-tv1", FirmID: firm.ID, CaseNumber: "CASE-TV1", OpenedAt: time.Now()}
	database.Create(caseRecord)

	doc := &models.CaseDocument{
		ID:               "doc-tv1",
		FirmID:           firm.ID,
		CaseID:           stringToPtr(caseRecord.ID),
		FileName:         "toggle.pdf",
		FileOriginalName: "toggle.pdf",
		FilePath:         "path/to/toggle.pdf",
		FileSize:         1024,
		IsPublic:         false,
	}
	database.Create(doc)

	t.Run("Toggle Success", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodPost, "/api/cases/case-tv1/documents/doc-tv1/toggle-visibility", nil)
		c.Request().Header.Set("HX-Request", "true")
		c.SetParamNames("id", "docId")
		c.SetParamValues("case-tv1", "doc-tv1")
		c.Set("user", admin)
		c.Set("firm", firm)

		err := ToggleDocumentVisibilityHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var updated models.CaseDocument
		database.First(&updated, "id = ?", "doc-tv1")
		assert.True(t, updated.IsPublic)
	})
}
