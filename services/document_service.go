package services

import (
	"context"
	"fmt"
	"io"
	"law_flow_app_go/models"
	"log"
	"os"
	"path/filepath"

	"gorm.io/gorm"
)

// TransferRequestDocumentToCase transfers the document from a CaseRequest to a Case
// This moves the physical file from case_requests/ to cases/{case_id}/ folder
func TransferRequestDocumentToCase(tx *gorm.DB, request *models.CaseRequest, caseID string, uploadedByID string) error {
	// Check if request has a document
	if request.FileName == "" || request.FilePath == "" {
		// No document to transfer
		return nil
	}

	// Build new file path in cases folder
	newFilePath := filepath.Join("uploads", "firms", request.FirmID, "cases", caseID, request.FileName)

	// Create destination directory
	destDir := filepath.Dir(newFilePath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create case documents directory: %w", err)
	}

	// Move file from case_requests to cases folder
	oldFilePath := request.FilePath
	if err := os.Rename(oldFilePath, newFilePath); err != nil {
		// If rename fails (e.g., cross-device), try copy + delete
		if err := copyFile(oldFilePath, newFilePath); err != nil {
			return fmt.Errorf("failed to copy file to case folder: %w", err)
		}
		// Delete original file after successful copy
		if err := os.Remove(oldFilePath); err != nil {
			log.Printf("Warning: failed to delete original file %s: %v", oldFilePath, err)
		}
	}

	// Create CaseDocument record with new path
	caseDoc := &models.CaseDocument{
		FirmID:           request.FirmID,
		CaseRequestID:    &request.ID,
		CaseID:           &caseID,
		FileName:         request.FileName,
		FileOriginalName: request.FileOriginalName,
		FilePath:         newFilePath, // Use new path
		FileSize:         request.FileSize,
		DocumentType:     "initial_request", // Mark as initial request document
		UploadedByID:     &uploadedByID,
	}

	if err := tx.Create(caseDoc).Error; err != nil {
		// Rollback file move on database error
		os.Rename(newFilePath, oldFilePath)
		return fmt.Errorf("failed to create case document: %w", err)
	}

	log.Printf("Moved document from request %s to case %s: %s -> %s", request.ID, caseID, oldFilePath, newFilePath)
	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// GetCaseDocuments retrieves all documents for a case
func GetCaseDocuments(db *gorm.DB, caseID string) ([]models.CaseDocument, error) {
	var documents []models.CaseDocument
	if err := db.Where("case_id = ?", caseID).
		Preload("UploadedBy").
		Order("created_at DESC").
		Find(&documents).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch case documents: %w", err)
	}
	return documents, nil
}

// GetDocumentPath returns the full path to a document file
func GetDocumentPath(document *models.CaseDocument) string {
	// Documents are organized as: uploads/firms/{firm_id}/cases/{case_id}/{filename}
	if document.CaseID != nil {
		return filepath.Join("uploads", "firms", document.FirmID, "cases", *document.CaseID, document.FileName)
	}
	// Fallback for request-only documents
	if document.CaseRequestID != nil {
		return filepath.Join("uploads", "firms", document.FirmID, "case_requests", document.FileName)
	}
	return document.FilePath // Use stored path as fallback
}

// DeleteCaseDocument soft deletes a case document and removes the physical file
func DeleteCaseDocument(db *gorm.DB, documentID string, userID string, firmID string) error {
	// First find the document to get the file path
	var document models.CaseDocument
	if err := db.Where("id = ? AND firm_id = ?", documentID, firmID).First(&document).Error; err != nil {
		return fmt.Errorf("document not found: %w", err)
	}

	// Delete physical file from storage
	if document.FilePath != "" {
		// Use background context for deletion as this is a cleanup task
		if err := Storage.Delete(context.Background(), document.FilePath); err != nil {
			log.Printf("Warning: failed to delete file document %s: %v", document.FilePath, err)
			// We continue with DB deletion even if file deletion fails,
			// though ideally we might want to schedule a retry
		}
	}

	// Delete from database
	result := db.Delete(&document)
	if result.Error != nil {
		return fmt.Errorf("failed to delete document: %w", result.Error)
	}

	// Audit log handled by handler or caller
	log.Printf("Document %s deleted by user %s", documentID, userID)
	return nil
}
