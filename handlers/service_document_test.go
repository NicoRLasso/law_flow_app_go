package handlers

import (
	"context"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetServiceDocumentsHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-sdc1", Name: "Doc Firm"}
	database.Create(firm)
	admin := &models.User{ID: "admin-sdc1", Name: "Admin", Email: "admin-sdc1@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)
	client := &models.User{ID: "client-sdc1", Name: "Client", Email: "client-sdc1@test.com", FirmID: stringToPtr(firm.ID), Role: "client"}
	database.Create(client)

	service := &models.LegalService{
		ID:            "service-sdc1",
		FirmID:        firm.ID,
		ServiceNumber: "SVC-2026-00010",
		Title:         "Doc Service",
		ClientID:      client.ID,
		CreatedAt:     time.Now(),
	}
	database.Create(service)

	database.Create(&models.ServiceDocument{
		ID:               "doc-1",
		FirmID:           firm.ID,
		ServiceID:        service.ID,
		FileOriginalName: "test.pdf",
		IsPublic:         true,
		UploadedByID:     &admin.ID,
	})

	database.Create(&models.ServiceDocument{
		ID:               "doc-2",
		FirmID:           firm.ID,
		ServiceID:        service.ID,
		FileOriginalName: "private.pdf",
		IsPublic:         false,
		UploadedByID:     &admin.ID,
	})

	t.Run("Admin sees all", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/api/services/service-sdc1/documents", nil)
		c.SetParamNames("id")
		c.SetParamValues("service-sdc1")
		c.Set("user", admin)
		c.Set("firm", firm)

		err := GetServiceDocumentsHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "test.pdf")
		assert.Contains(t, rec.Body.String(), "private.pdf")
	})

	t.Run("Client sees only public", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/api/services/service-sdc1/documents", nil)
		c.SetParamNames("id")
		c.SetParamValues("service-sdc1")
		c.Set("user", client)
		c.Set("firm", firm)

		err := GetServiceDocumentsHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "test.pdf")
		assert.NotContains(t, rec.Body.String(), "private.pdf")
	})
}

func TestToggleServiceDocumentVisibilityHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-sdc2", Name: "Toggle Firm"}
	database.Create(firm)
	admin := &models.User{ID: "admin-sdc2", Name: "Admin", Email: "admin-sdc2@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)

	doc := &models.ServiceDocument{
		ID:               "doc-toggle",
		FirmID:           firm.ID,
		ServiceID:        "service-1",
		FileOriginalName: "toggle.pdf",
		IsPublic:         false,
		UploadedByID:     &admin.ID,
	}
	database.Create(doc)

	t.Run("Success", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodPost, "/api/services/service-1/documents/doc-toggle/toggle-visibility", nil)
		c.SetParamNames("id", "did")
		c.SetParamValues("service-1", "doc-toggle")
		c.Set("user", admin)
		c.Set("firm", firm)

		err := ToggleServiceDocumentVisibilityHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		database.First(doc, "id = ?", "doc-toggle")
		assert.True(t, doc.IsPublic)
	})
}

func TestDownloadServiceDocumentHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-sdc3", Name: "Download Firm"}
	database.Create(firm)
	admin := &models.User{ID: "admin-sdc3", Name: "Admin", Email: "admin-sdc3@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)

	doc := &models.ServiceDocument{
		ID:               "doc-dl",
		FirmID:           firm.ID,
		ServiceID:        "service-1",
		FileOriginalName: "download.pdf",
		FilePath:         "firms/firm-sdc3/services/service-1/download.pdf",
		IsPublic:         true,
	}
	database.Create(doc)

	// Mock file in storage
	content := "test pdf content"
	_, _ = services.Storage.UploadReader(context.Background(), strings.NewReader(content), doc.FilePath, "application/pdf", int64(len(content)))

	t.Run("Success", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/api/services/service-1/documents/doc-dl/download", nil)
		c.SetParamNames("id", "did")
		c.SetParamValues("service-1", "doc-dl")
		c.Set("user", admin)
		c.Set("firm", firm)

		err := DownloadServiceDocumentHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/pdf", rec.Header().Get("Content-Type"))
	})
}

func TestDeleteServiceDocumentHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-sdc4", Name: "Delete Firm"}
	database.Create(firm)
	admin := &models.User{ID: "admin-sdc4", Name: "Admin", Email: "admin-sdc4@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)

	// Create the service first
	service := &models.LegalService{
		ID:            "service-1",
		FirmID:        firm.ID,
		ServiceNumber: "SVC-2026-00099",
		Title:         "Delete Test Service",
		ClientID:      admin.ID,
		CreatedAt:     time.Now(),
	}
	database.Create(service)

	doc := &models.ServiceDocument{
		ID:               "doc-del",
		FirmID:           firm.ID,
		ServiceID:        "service-1",
		FileOriginalName: "delete.pdf",
		FilePath:         "firms/firm-sdc4/services/service-1/delete.pdf",
	}
	database.Create(doc)

	// Mock file in storage
	content := "to be deleted"
	_, _ = services.Storage.UploadReader(context.Background(), strings.NewReader(content), doc.FilePath, "application/pdf", int64(len(content)))

	t.Run("Success", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodDelete, "/api/services/service-1/documents/doc-del", nil)
		c.SetParamNames("id", "did")
		c.SetParamValues("service-1", "doc-del")
		c.Set("user", admin)
		c.Set("firm", firm)

		err := DeleteServiceDocumentHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var deleted models.ServiceDocument
		err = database.First(&deleted, "id = ?", "doc-del").Error
		assert.Error(t, err) // Should be not found
	})
}

func TestViewServiceDocumentHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-sdc5", Name: "View Firm"}
	database.Create(firm)
	admin := &models.User{ID: "admin-sdc5", Name: "Admin", Email: "admin-sdc5@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)

	doc := &models.ServiceDocument{
		ID:               "doc-view",
		FirmID:           firm.ID,
		ServiceID:        "service-1",
		FileOriginalName: "view.pdf",
		FilePath:         "firms/firm-sdc5/services/service-1/view.pdf",
		IsPublic:         true,
	}
	database.Create(doc)

	// Mock file in storage
	content := "pdf content"
	_, _ = services.Storage.UploadReader(context.Background(), strings.NewReader(content), doc.FilePath, "application/pdf", int64(len(content)))

	t.Run("Success", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/api/services/service-1/documents/doc-view/view", nil)
		c.SetParamNames("id", "did")
		c.SetParamValues("service-1", "doc-view")
		c.Set("user", admin)
		c.Set("firm", firm)

		err := ViewServiceDocumentHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Disposition"), "inline")
	})
}
