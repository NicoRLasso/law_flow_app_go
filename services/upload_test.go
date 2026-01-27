package services

import (
	"bytes"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createMockFileHeader(filename string, content []byte, contentType string) *multipart.FileHeader {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", filename)
	part.Write(content)
	writer.Close()

	reader := multipart.NewReader(body, writer.Boundary())
	form, _ := reader.ReadForm(10 * 1024 * 1024)
	return form.File["file"][0]
}

func TestValidateDocumentUpload(t *testing.T) {
	t.Run("Valid PDF", func(t *testing.T) {
		content := append([]byte("%PDF-1.4\n"), make([]byte, 100)...)
		file := createMockFileHeader("test.pdf", content, "application/pdf")
		err := ValidateDocumentUpload(file)
		assert.NoError(t, err)
	})

	t.Run("Valid PNG", func(t *testing.T) {
		content := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 100)...)
		file := createMockFileHeader("test.png", content, "image/png")
		err := ValidateDocumentUpload(file)
		assert.NoError(t, err)
	})

	t.Run("Valid DOCX", func(t *testing.T) {
		content := append([]byte("PK\x03\x04"), make([]byte, 100)...)
		file := createMockFileHeader("test.docx", content, "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
		err := ValidateDocumentUpload(file)
		assert.NoError(t, err)
	})

	t.Run("File too large", func(t *testing.T) {
		content := make([]byte, 11*1024*1024) // 11MB
		file := createMockFileHeader("large.pdf", content, "application/pdf")
		err := ValidateDocumentUpload(file)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds the maximum limit")
	})

	t.Run("Invalid extension", func(t *testing.T) {
		file := createMockFileHeader("test.exe", []byte("fake"), "application/x-msdownload")
		err := ValidateDocumentUpload(file)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file type not allowed")
	})

	t.Run("Mismatched content (PDF extension but text)", func(t *testing.T) {
		file := createMockFileHeader("fake.pdf", []byte("this is just text"), "text/plain")
		err := ValidateDocumentUpload(file)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid PDF file content")
	})
}

func TestSaveAndDeleteCaseDocument(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "upload_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	content := []byte("%PDF-1.4\nSome content")
	file := createMockFileHeader("input.pdf", content, "application/pdf")

	firmID := "firm123"
	caseID := "case456"

	var result *UploadResult
	t.Run("SaveCaseDocument", func(t *testing.T) {
		res, err := SaveCaseDocument(file, tempDir, firmID, caseID)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "input.pdf", res.FileOriginalName)

		// Verify file exists on disk
		_, err = os.Stat(res.FilePath)
		assert.NoError(t, err)

		// Verify depth
		expectedDir := filepath.Join(tempDir, "firms", firmID, "cases", caseID)
		assert.Contains(t, res.FilePath, expectedDir)

		result = res
	})

	t.Run("DeleteUploadedFile", func(t *testing.T) {
		err := DeleteUploadedFile(result.FilePath)
		assert.NoError(t, err)

		_, err = os.Stat(result.FilePath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("Delete non-existent file", func(t *testing.T) {
		err := DeleteUploadedFile("non_existent_path")
		assert.Error(t, err) // os.Remove returns error if not exists and not handled
	})
}
