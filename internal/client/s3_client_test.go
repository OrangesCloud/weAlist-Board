package client

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"project-board-api/internal/config"
)

func TestGenerateFileKey(t *testing.T) {
	cfg := &config.S3Config{
		Bucket:    "test-bucket",
		Region:    "ap-northeast-2",
		AccessKey: "test-access-key",
		SecretKey: "test-secret-key",
	}

	client, err := NewS3Client(cfg)
	require.NoError(t, err)
	require.NotNil(t, client)

	tests := []struct {
		name        string
		entityType  string
		workspaceID string
		fileExt     string
		wantErr     bool
		errContains string
	}{
		{
			name:        "Valid boards entity type",
			entityType:  "boards",
			workspaceID: "workspace-123",
			fileExt:     ".jpg",
			wantErr:     false,
		},
		{
			name:        "Valid comments entity type",
			entityType:  "comments",
			workspaceID: "workspace-456",
			fileExt:     ".pdf",
			wantErr:     false,
		},
		{
			name:        "Valid projects entity type",
			entityType:  "projects",
			workspaceID: "workspace-789",
			fileExt:     ".png",
			wantErr:     false,
		},
		{
			name:        "Invalid entity type",
			entityType:  "invalid",
			workspaceID: "workspace-123",
			fileExt:     ".jpg",
			wantErr:     true,
			errContains: "invalid entity type",
		},
		{
			name:        "Empty entity type",
			entityType:  "",
			workspaceID: "workspace-123",
			fileExt:     ".jpg",
			wantErr:     true,
			errContains: "invalid entity type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := client.GenerateFileKey(tt.entityType, tt.workspaceID, tt.fileExt)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, key)

			// Verify key format: board/{entityType}/{workspaceId}/{year}/{month}/{uuid}_{timestamp}.ext
			parts := strings.Split(key, "/")
			assert.Equal(t, 6, len(parts), "Key should have 6 parts separated by /")
			assert.Equal(t, "board", parts[0])
			assert.Equal(t, tt.entityType, parts[1])
			assert.Equal(t, tt.workspaceID, parts[2])

			// Verify year format (4 digits)
			assert.Len(t, parts[3], 4, "Year should be 4 digits")

			// Verify month format (2 digits)
			assert.Len(t, parts[4], 2, "Month should be 2 digits")

			// Verify filename format: {uuid}_{timestamp}.ext
			filename := parts[5]
			assert.True(t, strings.HasSuffix(filename, tt.fileExt), "Filename should end with extension")
			assert.Contains(t, filename, "_", "Filename should contain underscore separator")
		})
	}
}

func TestGenerateFileKey_Uniqueness(t *testing.T) {
	cfg := &config.S3Config{
		Bucket:    "test-bucket",
		Region:    "ap-northeast-2",
		AccessKey: "test-access-key",
		SecretKey: "test-secret-key",
	}

	client, err := NewS3Client(cfg)
	require.NoError(t, err)

	// Generate multiple keys and verify they are unique
	keys := make(map[string]bool)
	for i := 0; i < 100; i++ {
		key, err := client.GenerateFileKey("boards", "workspace-123", ".jpg")
		require.NoError(t, err)
		assert.False(t, keys[key], "Generated key should be unique")
		keys[key] = true
	}
}

func TestGeneratePresignedURL(t *testing.T) {
	cfg := &config.S3Config{
		Bucket:    "test-bucket",
		Region:    "ap-northeast-2",
		AccessKey: "test-access-key",
		SecretKey: "test-secret-key",
	}

	client, err := NewS3Client(cfg)
	require.NoError(t, err)
	require.NotNil(t, client)

	tests := []struct {
		name        string
		entityType  string
		workspaceID string
		fileName    string
		contentType string
		wantErr     bool
		errContains string
	}{
		{
			name:        "Valid board image upload",
			entityType:  "boards",
			workspaceID: "workspace-123",
			fileName:    "image.jpg",
			contentType: "image/jpeg",
			wantErr:     false,
		},
		{
			name:        "Valid comment PDF upload",
			entityType:  "comments",
			workspaceID: "workspace-456",
			fileName:    "document.pdf",
			contentType: "application/pdf",
			wantErr:     false,
		},
		{
			name:        "Valid project PNG upload",
			entityType:  "projects",
			workspaceID: "workspace-789",
			fileName:    "diagram.png",
			contentType: "image/png",
			wantErr:     false,
		},
		{
			name:        "Invalid entity type",
			entityType:  "invalid",
			workspaceID: "workspace-123",
			fileName:    "image.jpg",
			contentType: "image/jpeg",
			wantErr:     true,
			errContains: "invalid entity type",
		},
		{
			name:        "Empty workspace ID",
			entityType:  "boards",
			workspaceID: "",
			fileName:    "image.jpg",
			contentType: "image/jpeg",
			wantErr:     false, // Should still work, just creates key with empty workspace
		},
		{
			name:        "File without extension",
			entityType:  "boards",
			workspaceID: "workspace-123",
			fileName:    "noextension",
			contentType: "image/jpeg",
			wantErr:     false, // Should work, just no extension in key
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			url, fileKey, err := client.GeneratePresignedURL(ctx, tt.entityType, tt.workspaceID, tt.fileName, tt.contentType)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, url, "Presigned URL should not be empty")
			assert.NotEmpty(t, fileKey, "File key should not be empty")

			// Verify URL contains bucket name
			assert.Contains(t, url, cfg.Bucket, "URL should contain bucket name")

			// Verify URL contains file key (URL encoded)
			assert.Contains(t, url, "board", "URL should contain 'board' prefix")

			// Verify URL has AWS signature parameters
			assert.Contains(t, url, "X-Amz-Algorithm", "URL should contain AWS signature algorithm")
			assert.Contains(t, url, "X-Amz-Credential", "URL should contain AWS credentials")
			assert.Contains(t, url, "X-Amz-Date", "URL should contain date")
			assert.Contains(t, url, "X-Amz-Expires", "URL should contain expiration")
			assert.Contains(t, url, "X-Amz-SignedHeaders", "URL should contain signed headers")
			assert.Contains(t, url, "X-Amz-Signature", "URL should contain signature")

			// Verify file key format
			parts := strings.Split(fileKey, "/")
			assert.GreaterOrEqual(t, len(parts), 6, "File key should have at least 6 parts")
			assert.Equal(t, "board", parts[0])
			assert.Equal(t, tt.entityType, parts[1])
		})
	}
}

func TestGeneratePresignedURL_ExpirationTime(t *testing.T) {
	cfg := &config.S3Config{
		Bucket:    "test-bucket",
		Region:    "ap-northeast-2",
		AccessKey: "test-access-key",
		SecretKey: "test-secret-key",
	}

	client, err := NewS3Client(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	url, _, err := client.GeneratePresignedURL(ctx, "boards", "workspace-123", "test.jpg", "image/jpeg")
	require.NoError(t, err)

	// Verify URL contains expiration parameter
	assert.Contains(t, url, "X-Amz-Expires=300", "URL should expire in 300 seconds (5 minutes)")
}

func TestNewS3Client_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.S3Config
		wantErr     bool
		errContains string
	}{
		{
			name: "Valid configuration",
			cfg: &config.S3Config{
				Bucket:    "test-bucket",
				Region:    "ap-northeast-2",
				AccessKey: "test-access-key",
				SecretKey: "test-secret-key",
			},
			wantErr: false,
		},
		{
			name: "Missing bucket",
			cfg: &config.S3Config{
				Region:    "ap-northeast-2",
				AccessKey: "test-access-key",
				SecretKey: "test-secret-key",
			},
			wantErr:     true,
			errContains: "bucket is required",
		},
		{
			name: "Missing region",
			cfg: &config.S3Config{
				Bucket:    "test-bucket",
				AccessKey: "test-access-key",
				SecretKey: "test-secret-key",
			},
			wantErr:     true,
			errContains: "region is required",
		},
		{
			name: "With custom endpoint (MinIO)",
			cfg: &config.S3Config{
				Bucket:    "test-bucket",
				Region:    "us-east-1",
				AccessKey: "minioadmin",
				SecretKey: "minioadmin",
				Endpoint:  "http://localhost:9000",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewS3Client(tt.cfg)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestGetFileURL(t *testing.T) {
	cfg := &config.S3Config{
		Bucket:    "test-bucket",
		Region:    "ap-northeast-2",
		AccessKey: "test-access-key",
		SecretKey: "test-secret-key",
	}

	client, err := NewS3Client(cfg)
	require.NoError(t, err)

	fileKey := "board/boards/workspace-123/2024/01/uuid_1234567890.jpg"
	url := client.GetFileURL(fileKey)

	expectedURL := "https://test-bucket.s3.ap-northeast-2.amazonaws.com/board/boards/workspace-123/2024/01/uuid_1234567890.jpg"
	assert.Equal(t, expectedURL, url)
}

func TestGeneratePresignedURL_ContextCancellation(t *testing.T) {
	cfg := &config.S3Config{
		Bucket:    "test-bucket",
		Region:    "ap-northeast-2",
		AccessKey: "test-access-key",
		SecretKey: "test-secret-key",
	}

	client, err := NewS3Client(cfg)
	require.NoError(t, err)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// This should fail because presigning requires credential resolution
	// which may involve network calls
	_, _, err = client.GeneratePresignedURL(ctx, "boards", "workspace-123", "test.jpg", "image/jpeg")
	
	// Should get an error due to cancelled context
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestGeneratePresignedURL_ConcurrentCalls(t *testing.T) {
	cfg := &config.S3Config{
		Bucket:    "test-bucket",
		Region:    "ap-northeast-2",
		AccessKey: "test-access-key",
		SecretKey: "test-secret-key",
	}

	client, err := NewS3Client(cfg)
	require.NoError(t, err)

	// Test concurrent calls to ensure thread safety
	const numGoroutines = 10
	results := make(chan struct {
		url     string
		fileKey string
		err     error
	}, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			ctx := context.Background()
			url, fileKey, err := client.GeneratePresignedURL(ctx, "boards", "workspace-123", "test.jpg", "image/jpeg")
			results <- struct {
				url     string
				fileKey string
				err     error
			}{url, fileKey, err}
		}()
	}

	// Collect results
	urls := make(map[string]bool)
	fileKeys := make(map[string]bool)
	for i := 0; i < numGoroutines; i++ {
		result := <-results
		require.NoError(t, result.err)
		assert.NotEmpty(t, result.url)
		assert.NotEmpty(t, result.fileKey)
		
		// Each call should generate unique file keys
		assert.False(t, fileKeys[result.fileKey], "File keys should be unique")
		fileKeys[result.fileKey] = true
		urls[result.url] = true
	}

	// All URLs and file keys should be unique
	assert.Equal(t, numGoroutines, len(urls), "All URLs should be unique")
	assert.Equal(t, numGoroutines, len(fileKeys), "All file keys should be unique")
}

func TestGenerateFileKey_DateFormatting(t *testing.T) {
	cfg := &config.S3Config{
		Bucket:    "test-bucket",
		Region:    "ap-northeast-2",
		AccessKey: "test-access-key",
		SecretKey: "test-secret-key",
	}

	client, err := NewS3Client(cfg)
	require.NoError(t, err)

	key, err := client.GenerateFileKey("boards", "workspace-123", ".jpg")
	require.NoError(t, err)

	parts := strings.Split(key, "/")
	require.Equal(t, 6, len(parts))

	// Verify year is current year
	year := parts[3]
	currentYear := time.Now().Format("2006")
	assert.Equal(t, currentYear, year)

	// Verify month is current month
	month := parts[4]
	currentMonth := time.Now().Format("01")
	assert.Equal(t, currentMonth, month)
}
