package services

import (
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"law_flow_app_go/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

// StorageProvider defines the interface for file storage operations
type StorageProvider interface {
	Upload(ctx context.Context, file *multipart.FileHeader, key string) (*StorageResult, error)
	UploadReader(ctx context.Context, reader io.Reader, key string, contentType string, size int64) (*StorageResult, error)
	Delete(ctx context.Context, key string) error
	Get(ctx context.Context, key string) (io.ReadCloser, string, error) // Returns reader, content-type, error
	GetSignedURL(ctx context.Context, key string, expiration time.Duration) (string, error)
	GetPublicURL(key string) string
	IsConfigured() bool
}

// StorageResult contains information about the stored file
type StorageResult struct {
	Key              string // Storage key/path
	FileName         string // Generated safe filename
	FileOriginalName string // Original filename
	FileSize         int64
	MimeType         string
	URL              string // Public or signed URL
}

// Storage is the global storage instance
var Storage StorageProvider

// InitializeStorage sets up the storage provider based on configuration
func InitializeStorage(cfg *config.Config) {
	if cfg.R2AccountID != "" && cfg.R2AccessKeyID != "" && cfg.R2SecretAccessKey != "" && cfg.R2BucketName != "" {
		r2, err := NewR2Storage(cfg)
		if err != nil {
			log.Printf("[WARNING] Failed to initialize R2 storage: %v. Falling back to local storage.", err)
			Storage = NewLocalStorage(cfg.UploadDir)
			log.Println("Storage connection established (Local filesystem - fallback)")
			return
		}

		// Test R2 connection by listing bucket (HeadBucket)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err = r2.client.HeadBucket(ctx, &s3.HeadBucketInput{
			Bucket: &cfg.R2BucketName,
		})
		if err != nil {
			log.Printf("[WARNING] R2 bucket connection test failed: %v. Falling back to local storage.", err)
			Storage = NewLocalStorage(cfg.UploadDir)
			log.Println("Storage connection established (Local filesystem - fallback)")
			return
		}

		Storage = r2
		log.Printf("Storage connection established (Cloudflare R2 - bucket: %s)", cfg.R2BucketName)
	} else {
		Storage = NewLocalStorage(cfg.UploadDir)
		log.Printf("Storage connection established (Local filesystem - path: %s)", cfg.UploadDir)
	}
}

// R2Storage implements StorageProvider for Cloudflare R2
type R2Storage struct {
	client    *s3.Client
	presigner *s3.PresignClient
	bucket    string
	publicURL string
	accountID string
}

// NewR2Storage creates a new R2 storage provider
func NewR2Storage(cfg *config.Config) (*R2Storage, error) {
	// R2 endpoint format: https://<account_id>.r2.cloudflarestorage.com
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.R2AccountID)

	// Create custom credentials provider
	creds := credentials.NewStaticCredentialsProvider(
		cfg.R2AccessKeyID,
		cfg.R2SecretAccessKey,
		"",
	)

	// Load AWS config with custom endpoint
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithCredentialsProvider(creds),
		awsconfig.WithRegion("auto"), // R2 uses "auto" region
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with custom endpoint
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	// Create presigner for signed URLs
	presigner := s3.NewPresignClient(client)

	return &R2Storage{
		client:    client,
		presigner: presigner,
		bucket:    cfg.R2BucketName,
		publicURL: cfg.R2PublicURL,
		accountID: cfg.R2AccountID,
	}, nil
}

// IsConfigured returns true if R2 is properly configured
func (r *R2Storage) IsConfigured() bool {
	return r.client != nil && r.bucket != ""
}

// Upload uploads a file to R2
func (r *R2Storage) Upload(ctx context.Context, file *multipart.FileHeader, key string) (*StorageResult, error) {
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return r.UploadReader(ctx, src, key, contentType, file.Size)
}

// UploadReader uploads content from a reader to R2
func (r *R2Storage) UploadReader(ctx context.Context, reader io.Reader, key string, contentType string, size int64) (*StorageResult, error) {
	input := &s3.PutObjectInput{
		Bucket:        aws.String(r.bucket),
		Key:           aws.String(key),
		Body:          reader,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(size),
	}

	_, err := r.client.PutObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to R2: %w", err)
	}

	return &StorageResult{
		Key:      key,
		FileName: filepath.Base(key),
		FileSize: size,
		MimeType: contentType,
		URL:      r.GetPublicURL(key),
	}, nil
}

// Delete removes a file from R2
func (r *R2Storage) Delete(ctx context.Context, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	}

	_, err := r.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete from R2: %w", err)
	}

	return nil
}

// Get retrieves a file from R2 and returns a reader
func (r *R2Storage) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	}

	result, err := r.client.GetObject(ctx, input)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get object from R2: %w", err)
	}

	contentType := "application/octet-stream"
	if result.ContentType != nil {
		contentType = *result.ContentType
	}

	return result.Body, contentType, nil
}

// GetSignedURL generates a presigned URL for temporary access
func (r *R2Storage) GetSignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	}

	presignedReq, err := r.presigner.PresignGetObject(ctx, input, s3.WithPresignExpires(expiration))
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}

	return presignedReq.URL, nil
}

// GetPublicURL returns the public URL for a file (if public URL is configured)
func (r *R2Storage) GetPublicURL(key string) string {
	if r.publicURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimSuffix(r.publicURL, "/"), key)
	}
	// If no public URL, return empty - caller should use GetSignedURL
	return ""
}

// LocalStorage implements StorageProvider for local filesystem
type LocalStorage struct {
	baseDir string
}

// NewLocalStorage creates a new local storage provider
func NewLocalStorage(baseDir string) *LocalStorage {
	return &LocalStorage{baseDir: baseDir}
}

// IsConfigured returns true (local storage is always available)
func (l *LocalStorage) IsConfigured() bool {
	return true
}

// Upload saves a file to local filesystem
func (l *LocalStorage) Upload(ctx context.Context, file *multipart.FileHeader, key string) (*StorageResult, error) {
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return l.UploadReader(ctx, src, key, contentType, file.Size)
}

// UploadReader saves content from a reader to local filesystem
func (l *LocalStorage) UploadReader(ctx context.Context, reader io.Reader, key string, contentType string, size int64) (*StorageResult, error) {
	fullPath := filepath.Join(l.baseDir, key)

	// Create directory structure
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Create destination file
	dst, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	// Copy content
	written, err := io.Copy(dst, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	return &StorageResult{
		Key:      key,
		FileName: filepath.Base(key),
		FileSize: written,
		MimeType: contentType,
		URL:      "/" + filepath.Join(l.baseDir, key),
	}, nil
}

// Delete removes a file from local filesystem
func (l *LocalStorage) Delete(ctx context.Context, key string) error {
	fullPath := filepath.Join(l.baseDir, key)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// Get retrieves a file from local filesystem and returns a reader
func (l *LocalStorage) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	fullPath := filepath.Join(l.baseDir, key)

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open file: %w", err)
	}

	// Detect content type from extension
	contentType := "application/octet-stream"
	ext := strings.ToLower(filepath.Ext(key))
	switch ext {
	case ".pdf":
		contentType = "application/pdf"
	case ".png":
		contentType = "image/png"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".doc":
		contentType = "application/msword"
	case ".docx":
		contentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	}

	return file, contentType, nil
}

// GetSignedURL for local storage just returns the file path (no signing needed)
func (l *LocalStorage) GetSignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	// Local files don't need signed URLs - return the path
	return "/" + filepath.Join(l.baseDir, key), nil
}

// GetPublicURL returns the local file path
func (l *LocalStorage) GetPublicURL(key string) string {
	return "/" + filepath.Join(l.baseDir, key)
}

// Helper functions for generating storage keys

// GenerateStorageKey creates a unique storage key for files
func GenerateStorageKey(prefix string, originalFilename string) string {
	ext := filepath.Ext(originalFilename)
	uniqueID := uuid.New().String()
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("%s_%d%s", uniqueID, timestamp, ext)
	return filepath.Join(prefix, filename)
}

// GenerateCaseDocumentKey creates a storage key for case documents
func GenerateCaseDocumentKey(firmID, caseID, originalFilename string) string {
	prefix := fmt.Sprintf("firms/%s/cases/%s", firmID, caseID)
	return GenerateStorageKey(prefix, originalFilename)
}

// GenerateFirmLogoKey creates a storage key for firm logos
func GenerateFirmLogoKey(firmID, originalFilename string) string {
	ext := filepath.Ext(originalFilename)
	return fmt.Sprintf("logos/%s%s", firmID, ext)
}

// GenerateCaseRequestFileKey creates a storage key for case request files
func GenerateCaseRequestFileKey(firmID, requestID, originalFilename string) string {
	prefix := fmt.Sprintf("firms/%s/requests/%s", firmID, requestID)
	return GenerateStorageKey(prefix, originalFilename)
}

// GenerateGeneratedDocumentKey creates a storage key for generated documents
func GenerateGeneratedDocumentKey(firmID, caseID, originalFilename string) string {
	prefix := fmt.Sprintf("firms/%s/cases/%s/generated", firmID, caseID)
	return GenerateStorageKey(prefix, originalFilename)
}
