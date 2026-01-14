package services

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	MaxUploadSize   = 10 * 1024 * 1024 // 10MB
	AllowedMimeType = "application/pdf"
)

type UploadResult struct {
	FileName         string
	FileOriginalName string
	FilePath         string
	FileSize         int64
	MimeType         string
}

// ValidatePDFUpload checks if the uploaded file is a valid PDF within size limits
func ValidatePDFUpload(fileHeader *multipart.FileHeader) error {
	// Check file size
	if fileHeader.Size > MaxUploadSize {
		return fmt.Errorf("file size exceeds maximum allowed size of 10MB")
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".pdf" {
		return fmt.Errorf("only PDF files are allowed")
	}

	// Open file to check MIME type
	file, err := fileHeader.Open()
	if err != nil {
		return fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer file.Close()

	// Read first 512 bytes to detect content type
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read file content: %w", err)
	}

	// Check if it's a PDF (PDF files start with %PDF)
	if len(buffer) < 4 || string(buffer[0:4]) != "%PDF" {
		return fmt.Errorf("file is not a valid PDF")
	}

	return nil
}

// SaveUploadedFile saves the uploaded file to disk with a secure filename
func SaveUploadedFile(fileHeader *multipart.FileHeader, uploadDir string, firmID string) (*UploadResult, error) {
	// Create firm-specific directory with better organization
	firmDir := filepath.Join(uploadDir, "firms", firmID, "case_requests")
	if err := os.MkdirAll(firmDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Generate secure filename using SHA256 hash + timestamp
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer file.Close()

	// Calculate hash
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, fmt.Errorf("failed to calculate file hash: %w", err)
	}
	hashStr := hex.EncodeToString(hash.Sum(nil))[:16] // Use first 16 chars

	// Generate filename: hash_timestamp.pdf
	timestamp := time.Now().Unix()
	fileName := fmt.Sprintf("%s_%d.pdf", hashStr, timestamp)
	filePath := filepath.Join(firmDir, fileName)

	// Verify path is within upload directory (prevent path traversal)
	absUploadDir, err := filepath.Abs(uploadDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve upload directory: %w", err)
	}
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve file path: %w", err)
	}
	if !strings.HasPrefix(absFilePath, absUploadDir) {
		return nil, fmt.Errorf("invalid file path: path traversal detected")
	}

	// Reset file pointer to beginning
	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to reset file pointer: %w", err)
	}

	// Create destination file
	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	// Copy file content
	written, err := io.Copy(dst, file)
	if err != nil {
		// Clean up on error
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	return &UploadResult{
		FileName:         fileName,
		FileOriginalName: fileHeader.Filename,
		FilePath:         filePath,
		FileSize:         written,
	}, nil
}

// DeleteUploadedFile removes a file from disk
func DeleteUploadedFile(filePath string) error {
	if filePath == "" {
		return nil
	}

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// GetFileURL generates a URL for file access (for authenticated downloads)
func GetFileURL(requestID string) string {
	return fmt.Sprintf("/api/case-requests/%s/file", requestID)
}

// ValidateDocumentUpload checks if the uploaded file is valid within size limits
func ValidateDocumentUpload(fileHeader *multipart.FileHeader) error {
	// Check file size
	if fileHeader.Size > MaxUploadSize {
		return fmt.Errorf("file size exceeds maximum allowed size of 10MB")
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	allowedExtensions := []string{".pdf", ".doc", ".docx", ".txt", ".jpg", ".jpeg", ".png"}

	isAllowed := false
	for _, allowed := range allowedExtensions {
		if ext == allowed {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		return fmt.Errorf("file type not allowed. Accepted formats: PDF, DOC, DOCX, TXT, JPG, PNG")
	}

	return nil
}

// SaveCaseDocument saves a document file for a specific case
func SaveCaseDocument(fileHeader *multipart.FileHeader, uploadDir string, firmID string, caseID string) (*UploadResult, error) {
	// Create case-specific directory
	caseDir := filepath.Join(uploadDir, "firms", firmID, "cases", caseID)
	if err := os.MkdirAll(caseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Generate secure filename using SHA256 hash + timestamp
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer file.Close()

	// Calculate hash
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, fmt.Errorf("failed to calculate file hash: %w", err)
	}
	hashStr := hex.EncodeToString(hash.Sum(nil))[:16] // Use first 16 chars

	// Get file extension
	ext := filepath.Ext(fileHeader.Filename)

	// Generate filename: hash_timestamp.ext
	timestamp := time.Now().Unix()
	fileName := fmt.Sprintf("%s_%d%s", hashStr, timestamp, ext)
	filePath := filepath.Join(caseDir, fileName)

	// Verify path is within upload directory (prevent path traversal)
	absUploadDir, err := filepath.Abs(uploadDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve upload directory: %w", err)
	}
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve file path: %w", err)
	}
	if !strings.HasPrefix(absFilePath, absUploadDir) {
		return nil, fmt.Errorf("invalid file path: path traversal detected")
	}

	// Reset file pointer to beginning
	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to reset file pointer: %w", err)
	}

	// Create destination file
	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	// Copy file content
	written, err := io.Copy(dst, file)
	if err != nil {
		// Clean up on error
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// Get MIME type
	mimeType := fileHeader.Header.Get("Content-Type")

	return &UploadResult{
		FileName:         fileName,
		FileOriginalName: fileHeader.Filename,
		FilePath:         filePath,
		FileSize:         written,
		MimeType:         mimeType,
	}, nil
}
