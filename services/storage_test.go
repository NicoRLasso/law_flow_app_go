package services

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLocalStorage(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage := NewLocalStorage(tempDir)
	ctx := context.Background()
	content := "hello storage"
	key := "test/file.txt"
	contentType := "text/plain"
	size := int64(len(content))

	t.Run("UploadReader creates file", func(t *testing.T) {
		reader := strings.NewReader(content)
		result, err := storage.UploadReader(ctx, reader, key, contentType, size)
		assert.NoError(t, err)
		assert.Equal(t, key, result.Key)
		assert.Equal(t, size, result.FileSize)

		// Verify file exists
		_, err = os.Stat(filepath.Join(tempDir, key))
		assert.NoError(t, err)
	})

	t.Run("Get retrieves file content", func(t *testing.T) {
		reader, retrievedType, err := storage.Get(ctx, key)
		assert.NoError(t, err)
		defer reader.Close()

		got, _ := io.ReadAll(reader)
		assert.Equal(t, content, string(got))
		assert.Equal(t, "application/octet-stream", retrievedType) // LocalStorage defaults to octet-stream for .txt
	})

	t.Run("Get detects MIME types correctly", func(t *testing.T) {
		pdfKey := "test/doc.pdf"
		storage.UploadReader(ctx, strings.NewReader("%PDF-1.4"), pdfKey, "application/pdf", 8)

		_, retrievedType, err := storage.Get(ctx, pdfKey)
		assert.NoError(t, err)
		assert.Equal(t, "application/pdf", retrievedType)

		jpgKey := "test/image.jpg"
		storage.UploadReader(ctx, strings.NewReader("fake-jpg"), jpgKey, "image/jpeg", 8)
		_, retrievedType, err = storage.Get(ctx, jpgKey)
		assert.NoError(t, err)
		assert.Equal(t, "image/jpeg", retrievedType)
	})

	t.Run("Delete removes file", func(t *testing.T) {
		err := storage.Delete(ctx, key)
		assert.NoError(t, err)

		_, err = os.Stat(filepath.Join(tempDir, key))
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("URLs and paths", func(t *testing.T) {
		expected := "/" + filepath.Join(tempDir, "some/key")
		url := storage.GetPublicURL("some/key")
		assert.Equal(t, expected, url)

		signed, err := storage.GetSignedURL(ctx, "some/key", time.Hour)
		assert.NoError(t, err)
		assert.Equal(t, expected, signed)
	})
}

func TestKeyGeneration(t *testing.T) {
	firmID := "f1"
	caseID := "c1"
	filename := "contract.pdf"

	t.Run("GenerateStorageKey", func(t *testing.T) {
		key := GenerateStorageKey("prefix", filename)
		assert.True(t, strings.HasPrefix(key, "prefix/"))
		assert.True(t, strings.HasSuffix(key, ".pdf"))
		// Check for UUID-like part
		parts := strings.Split(filepath.Base(key), "_")
		assert.Len(t, parts, 2)
	})

	t.Run("GenerateCaseDocumentKey", func(t *testing.T) {
		key := GenerateCaseDocumentKey(firmID, caseID, filename)
		assert.Contains(t, key, "firms/f1/cases/c1")
		assert.True(t, strings.HasSuffix(key, ".pdf"))
	})

	t.Run("GenerateFirmLogoKey", func(t *testing.T) {
		key := GenerateFirmLogoKey(firmID, filename)
		assert.Equal(t, "logos/f1.pdf", key)
	})

	t.Run("GenerateGeneratedDocumentKey", func(t *testing.T) {
		key := GenerateGeneratedDocumentKey(firmID, caseID, filename)
		assert.Contains(t, key, "firms/f1/cases/c1/generated")
	})
}

func TestIsConfigured(t *testing.T) {
	ls := NewLocalStorage("/tmp")
	assert.True(t, ls.IsConfigured())

	r2 := &R2Storage{bucket: "test-bucket", client: nil}
	assert.False(t, r2.IsConfigured())
}
