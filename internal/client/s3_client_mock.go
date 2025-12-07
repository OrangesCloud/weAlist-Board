package client

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MockS3Client implements S3ClientInterface for testing without AWS credentials
type MockS3Client struct {
	Bucket   string
	Region   string
	Endpoint string

	// Optional function overrides for custom test behavior
	GenerateFileKeyFunc      func(entityType, workspaceID, fileExt string) (string, error)
	GeneratePresignedURLFunc func(ctx context.Context, entityType, workspaceID, fileName, contentType string) (string, string, error)
	UploadFileFunc           func(ctx context.Context, key string, file io.Reader, contentType string) (string, error)
	DeleteFileFunc           func(ctx context.Context, key string) error
	GetFileURLFunc           func(key string) string
}

// NewMockS3Client creates a new mock S3 client for testing
func NewMockS3Client() *MockS3Client {
	return &MockS3Client{
		Bucket:   "test-bucket",
		Region:   "ap-northeast-2",
		Endpoint: "",
	}
}

// GenerateFileKey generates a unique file key for S3 storage
func (m *MockS3Client) GenerateFileKey(entityType, workspaceID, fileExt string) (string, error) {
	if m.GenerateFileKeyFunc != nil {
		return m.GenerateFileKeyFunc(entityType, workspaceID, fileExt)
	}

	// Default implementation
	validTypes := map[string]bool{
		"boards":   true,
		"comments": true,
		"projects": true,
	}

	if entityType == "" {
		return "", fmt.Errorf("entity type cannot be empty")
	}

	if !validTypes[entityType] {
		return "", fmt.Errorf("invalid entity type: %s", entityType)
	}

	now := time.Now()
	uniqueID := uuid.New().String()
	timestamp := now.UnixNano()

	key := fmt.Sprintf("board/%s/%s/%d/%02d/%s_%d%s",
		entityType,
		workspaceID,
		now.Year(),
		now.Month(),
		uniqueID,
		timestamp,
		fileExt,
	)

	return key, nil
}

// GeneratePresignedURL generates a mock presigned URL for testing
func (m *MockS3Client) GeneratePresignedURL(ctx context.Context, entityType, workspaceID, fileName, contentType string) (string, string, error) {
	if m.GeneratePresignedURLFunc != nil {
		return m.GeneratePresignedURLFunc(ctx, entityType, workspaceID, fileName, contentType)
	}

	// Default implementation
	fileExt := filepath.Ext(fileName)
	if fileExt == "" {
		fileExt = ".bin"
	}

	fileKey, err := m.GenerateFileKey(entityType, workspaceID, fileExt)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate file key: %w", err)
	}

	// Generate mock presigned URL with all AWS signature parameters
	now := time.Now().UTC().Format("20060102T150405Z")
	presignedURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=test-access-key%%2F%s%%2F%s%%2Fs3%%2Faws4_request&X-Amz-Date=%s&X-Amz-Expires=900&X-Amz-SignedHeaders=host&X-Amz-Signature=mocksignature123",
		m.Bucket,
		m.Region,
		fileKey,
		time.Now().UTC().Format("20060102"),
		m.Region,
		now,
	)

	return presignedURL, fileKey, nil
}

// UploadFile simulates file upload
func (m *MockS3Client) UploadFile(ctx context.Context, key string, file io.Reader, contentType string) (string, error) {
	if m.UploadFileFunc != nil {
		return m.UploadFileFunc(ctx, key, file, contentType)
	}

	// Default implementation - just return the URL
	return m.GetFileURL(key), nil
}

// DeleteFile simulates file deletion
func (m *MockS3Client) DeleteFile(ctx context.Context, key string) error {
	if m.DeleteFileFunc != nil {
		return m.DeleteFileFunc(ctx, key)
	}

	// Default implementation - always succeed
	return nil
}

// GetFileURL returns the public URL for a file
func (m *MockS3Client) GetFileURL(key string) string {
	if m.GetFileURLFunc != nil {
		return m.GetFileURLFunc(key)
	}

	// Default implementation
	if m.Endpoint != "" && !strings.Contains(m.Endpoint, "amazonaws.com") {
		return fmt.Sprintf("%s/%s/%s", m.Endpoint, m.Bucket, key)
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", m.Bucket, m.Region, key)
}

// Ensure MockS3Client implements S3ClientInterface
var _ S3ClientInterface = (*MockS3Client)(nil)
