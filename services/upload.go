package services

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// UploadResult contains information about the uploaded file
type UploadResult struct {
	FileName         string
	FileOriginalName string
	FilePath         string
	FileSize         int64
	MimeType         string
}

// ValidateDocumentUpload checks if the uploaded file is valid
// It checks file size, extension, and content type (magic bytes)
func ValidateDocumentUpload(file *multipart.FileHeader) error {
	// 1. Check file size (max 10MB)
	const maxFileSize = 10 * 1024 * 1024 // 10MB
	if file.Size > maxFileSize {
		return fmt.Errorf("file size exceeds the maximum limit of 10MB")
	}

	// 2. Check file extension
	allowedExtensions := map[string]bool{
		".pdf":  true,
		".doc":  true,
		".docx": true,
		".jpg":  true,
		".jpeg": true,
		".png":  true,
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedExtensions[ext] {
		return fmt.Errorf("file type not allowed. Allowed types: PDF, DOC, DOCX, JPG, PNG")
	}

	// 3. Check magic bytes (content type)
	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// Read first 512 bytes to detect content type
	buffer := make([]byte, 512)
	_, err = src.Read(buffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read file content: %w", err)
	}

	// Reset file pointer
	if _, err := src.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to reset file pointer: %w", err)
	}

	contentType := http.DetectContentType(buffer)

	isImage := strings.HasPrefix(contentType, "image/")
	isPDF := contentType == "application/pdf"

	// Magic bytes check for Office documents
	// DOCX (Zip): PK\x03\x04
	isDOCX := len(buffer) > 4 && string(buffer[:4]) == "PK\x03\x04"

	// DOC (OLECF): \xD0\xCF\x11\xE0\xA1\xB1\x1A\xE1
	isDOC := len(buffer) > 8 && string(buffer[:8]) == "\xD0\xCF\x11\xE0\xA1\xB1\x1A\xE1"

	if (ext == ".jpg" || ext == ".jpeg" || ext == ".png") && !isImage {
		return fmt.Errorf("invalid image file content")
	}

	if ext == ".pdf" && !isPDF {
		return fmt.Errorf("invalid PDF file content")
	}

	if ext == ".docx" && !isDOCX {
		return fmt.Errorf("invalid DOCX file content (signature mismatch)")
	}

	if ext == ".doc" && !isDOC {
		return fmt.Errorf("invalid DOC file content (signature mismatch)")
	}

	return nil
}

// SaveCaseDocument saves the uploaded file to the specified directory
func SaveCaseDocument(file *multipart.FileHeader, baseDir string, firmID string, caseID string) (*UploadResult, error) {
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// Create directory structure: uploads/firms/{firm_id}/cases/{case_id}/
	uploadDir := filepath.Join(baseDir, "firms", firmID, "cases", caseID)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Generate safe filename
	ext := filepath.Ext(file.Filename)
	uniqueID := uuid.New().String()
	safeFilename := fmt.Sprintf("%s_%d%s", uniqueID, time.Now().Unix(), ext)
	dstPath := filepath.Join(uploadDir, safeFilename)

	// Create destination file
	dst, err := os.Create(dstPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	// Copy content
	if _, err := io.Copy(dst, src); err != nil {
		return nil, fmt.Errorf("failed to save file content: %w", err)
	}

	return &UploadResult{
		FileName:         safeFilename,
		FileOriginalName: file.Filename,
		FilePath:         dstPath,
		FileSize:         file.Size,
		MimeType:         file.Header.Get("Content-Type"),
	}, nil
}

// DeleteUploadedFile deletes a file from the filesystem
func DeleteUploadedFile(path string) error {
	if path == "" {
		return nil
	}
	return os.Remove(path)
}
