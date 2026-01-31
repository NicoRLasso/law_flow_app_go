package services

import (
	"context"
	"io"
	"law_flow_app_go/models"
	"mime/multipart"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// MockStorageProvider is a mock implementation of StorageProvider
type MockStorageProvider struct {
	mock.Mock
}

func (m *MockStorageProvider) Upload(ctx context.Context, file *multipart.FileHeader, key string) (*StorageResult, error) {
	args := m.Called(ctx, file, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*StorageResult), args.Error(1)
}

func (m *MockStorageProvider) UploadReader(ctx context.Context, reader io.Reader, key string, contentType string, size int64) (*StorageResult, error) {
	args := m.Called(ctx, reader, key, contentType, size)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*StorageResult), args.Error(1)
}

func (m *MockStorageProvider) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockStorageProvider) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(io.ReadCloser), args.String(1), args.Error(2)
}

func (m *MockStorageProvider) GetSignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	args := m.Called(ctx, key, expiration)
	return args.String(0), args.Error(1)
}

func (m *MockStorageProvider) GetPublicURL(key string) string {
	args := m.Called(key)
	return args.String(0)
}

func (m *MockStorageProvider) IsConfigured() bool {
	args := m.Called()
	return args.Bool(0)
}

func setupDocumentTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&models.CaseDocument{}, &models.Case{}, &models.Firm{}, &models.User{})
	return db
}

func TestGetDocumentPath(t *testing.T) {
	doc := &models.CaseDocument{
		FirmID:   "firm-1",
		CaseID:   stringToPtr("case-1"),
		FileName: "test.pdf",
		FilePath: "some/path/test.pdf",
	}

	path := GetDocumentPath(doc)
	expected := filepath.Join("uploads", "firms", "firm-1", "cases", "case-1", "test.pdf")
	assert.Equal(t, expected, path)

}

func TestDeleteCaseDocument(t *testing.T) {
	db := setupDocumentTestDB()
	firmID := "firm-del"
	userID := "user-del"

	// Setup mock storage
	mStorage := new(MockStorageProvider)
	oldStorage := Storage
	Storage = mStorage
	defer func() { Storage = oldStorage }()

	filePath := "uploads/firms/firm-del/cases/c1/file.pdf"
	doc := models.CaseDocument{
		FirmID:   firmID,
		FileName: "file.pdf",
		FilePath: filePath,
	}
	db.Create(&doc)

	// Expectation
	mStorage.On("Delete", mock.Anything, filePath).Return(nil)

	err := DeleteCaseDocument(db, doc.ID, userID, firmID)
	assert.NoError(t, err)

	// Verify DB soft deleted (using db.Unscoped() to see soft-deleted records if needed, but DeleteCaseDocument does a regular delete)
	// Actually GORM Delete on a model with DeletedAt is a soft delete.
	var count int64
	db.Model(&models.CaseDocument{}).Where("id = ?", doc.ID).Count(&count)
	assert.Equal(t, int64(0), count)

	mStorage.AssertExpectations(t)
}

func TestGetCaseDocuments(t *testing.T) {
	db := setupDocumentTestDB()
	caseID := "case-list"

	db.Create(&models.CaseDocument{CaseID: &caseID, FileName: "doc1.pdf"})
	db.Create(&models.CaseDocument{CaseID: &caseID, FileName: "doc2.pdf"})
	db.Create(&models.CaseDocument{CaseID: stringToPtr("other"), FileName: "doc3.pdf"})

	docs, err := GetCaseDocuments(db, caseID)
	assert.NoError(t, err)
	assert.Len(t, docs, 2)
}
