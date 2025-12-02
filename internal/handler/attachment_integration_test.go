package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"project-board-api/internal/client"
	"project-board-api/internal/config"
	"project-board-api/internal/repository"
)

// setupAttachmentIntegrationTestDB creates an in-memory SQLite database for attachment integration testing
func setupAttachmentIntegrationTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "Failed to connect to test database")

	// Register callback to generate UUIDs for SQLite
	db.Callback().Create().Before("gorm:create").Register("generate_uuid", func(db *gorm.DB) {
		if db.Statement.Schema != nil {
			for _, field := range db.Statement.Schema.PrimaryFields {
				if field.DataType == "uuid" {
					fieldValue := field.ReflectValueOf(db.Statement.Context, db.Statement.ReflectValue)
					if fieldValue.IsZero() {
						field.Set(db.Statement.Context, db.Statement.ReflectValue, uuid.New())
					}
				}
			}
		}
	})

	// Create attachments table (current schema without Status and ExpiresAt)
	err = db.Exec(`
		CREATE TABLE attachments (
			id TEXT PRIMARY KEY,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			deleted_at DATETIME,
			entity_type TEXT NOT NULL,
			entity_id TEXT NOT NULL,
			file_name TEXT NOT NULL,
			file_url TEXT NOT NULL,
			file_size INTEGER NOT NULL,
			content_type TEXT NOT NULL,
			uploaded_by TEXT NOT NULL
		)
	`).Error
	require.NoError(t, err, "Failed to create attachments table")

	return db
}

// setupAttachmentIntegrationRouter creates a router with attachment handler
func setupAttachmentIntegrationRouter(db *gorm.DB, s3Client *client.S3Client) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add test middleware to set user_id from header
	router.Use(func(c *gin.Context) {
		if userIDStr := c.GetHeader("X-User-ID"); userIDStr != "" {
			if userID, err := uuid.Parse(userIDStr); err == nil {
				c.Set("user_id", userID)
			}
		}
		c.Next()
	})

	// Initialize repositories
	attachmentRepo := repository.NewAttachmentRepository(db)

	// Initialize handler
	attachmentHandler := NewAttachmentHandler(s3Client, attachmentRepo)

	// Setup routes
	api := router.Group("/api")
	{
		attachments := api.Group("/attachments")
		{
			attachments.POST("/presigned-url", attachmentHandler.GeneratePresignedURL)
		}
	}

	return router
}

// TestIntegration_PresignedURL_BoardFlow tests the complete presigned URL flow for board attachments
// **Validates: Requirements 1.3, 1.4, 1.5**
func TestIntegration_PresignedURL_BoardFlow(t *testing.T) {
	db := setupAttachmentIntegrationTestDB(t)

	// Create S3 client with test configuration
	cfg := &config.S3Config{
		Bucket:    "test-bucket",
		Region:    "us-east-1",
		AccessKey: "test-key",
		SecretKey: "test-secret",
	}
	s3Client, err := client.NewS3Client(cfg)
	require.NoError(t, err, "Failed to create S3 client")

	router := setupAttachmentIntegrationRouter(db, s3Client)

	workspaceID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name           string
		entityType     string
		fileName       string
		fileSize       int64
		contentType    string
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "Generate presigned URL for board image",
			entityType:     "BOARD",
			fileName:       "test-image.jpg",
			fileSize:       1024000,
			contentType:    "image/jpeg",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				data, ok := response["data"].(map[string]interface{})
				require.True(t, ok, "Response should contain data field")

				uploadURL, _ := data["uploadUrl"].(string)
				fileKey, _ := data["fileKey"].(string)
				expiresIn, _ := data["expiresIn"].(float64)

				assert.NotEmpty(t, uploadURL, "Upload URL should not be empty")
				assert.NotEmpty(t, fileKey, "File key should not be empty")
				assert.Equal(t, float64(300), expiresIn, "Expiration should be 300 seconds")
				assert.Contains(t, fileKey, "board/boards/", "File key should contain correct path")
			},
		},
		{
			name:           "Generate presigned URL for comment document",
			entityType:     "COMMENT",
			fileName:       "document.pdf",
			fileSize:       2048000,
			contentType:    "application/pdf",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				data, ok := response["data"].(map[string]interface{})
				require.True(t, ok, "Response should contain data field")

				uploadURL, _ := data["uploadUrl"].(string)
				fileKey, _ := data["fileKey"].(string)

				assert.NotEmpty(t, uploadURL)
				assert.NotEmpty(t, fileKey)
				assert.Contains(t, fileKey, "board/comments/", "File key should contain correct path")
			},
		},
		{
			name:           "Generate presigned URL for project file",
			entityType:     "PROJECT",
			fileName:       "spec.docx",
			fileSize:       512000,
			contentType:    "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				data, ok := response["data"].(map[string]interface{})
				require.True(t, ok, "Response should contain data field")

				uploadURL, _ := data["uploadUrl"].(string)
				fileKey, _ := data["fileKey"].(string)

				assert.NotEmpty(t, uploadURL)
				assert.NotEmpty(t, fileKey)
				assert.Contains(t, fileKey, "board/projects/", "File key should contain correct path")
			},
		},
		{
			name:           "Reject file exceeding size limit",
			entityType:     "BOARD",
			fileName:       "large-file.jpg",
			fileSize:       21 * 1024 * 1024, // 21MB
			contentType:    "image/jpeg",
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)

				assert.NotNil(t, resp["error"], "Should have error field")
			},
		},
		{
			name:           "Reject unsupported file type",
			entityType:     "BOARD",
			fileName:       "video.mp4",
			fileSize:       1024000,
			contentType:    "video/mp4",
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)

				assert.NotNil(t, resp["error"], "Should have error field")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Request presigned URL
			presignedReq := PresignedURLRequest{
				EntityType:  tt.entityType,
				WorkspaceID: workspaceID.String(),
				FileName:    tt.fileName,
				FileSize:    tt.fileSize,
				ContentType: tt.contentType,
			}

			body, err := json.Marshal(presignedReq)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/attachments/presigned-url", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-User-ID", userID.String())

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "Response body: %s", w.Body.String())

			if tt.validateFunc != nil {
				tt.validateFunc(t, w)
			}
		})
	}
}

// TestIntegration_PresignedURL_InvalidEntityType tests invalid entity type handling
// **Validates: Requirements 1.1, 1.2**
func TestIntegration_PresignedURL_InvalidEntityType(t *testing.T) {
	db := setupAttachmentIntegrationTestDB(t)

	cfg := &config.S3Config{
		Bucket:    "test-bucket",
		Region:    "us-east-1",
		AccessKey: "test-key",
		SecretKey: "test-secret",
	}
	s3Client, err := client.NewS3Client(cfg)
	require.NoError(t, err, "Failed to create S3 client")

	router := setupAttachmentIntegrationRouter(db, s3Client)

	workspaceID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name       string
		entityType string
	}{
		{
			name:       "Invalid entity type - USER",
			entityType: "USER",
		},
		{
			name:       "Invalid entity type - WORKSPACE",
			entityType: "WORKSPACE",
		},
		{
			name:       "Invalid entity type - empty string",
			entityType: "",
		},
		{
			name:       "Invalid entity type - random string",
			entityType: "INVALID_TYPE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			presignedReq := PresignedURLRequest{
				EntityType:  tt.entityType,
				WorkspaceID: workspaceID.String(),
				FileName:    "test.jpg",
				FileSize:    1024000,
				ContentType: "image/jpeg",
			}

			body, err := json.Marshal(presignedReq)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/attachments/presigned-url", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-User-ID", userID.String())

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code, "Should reject invalid entity type")

			var resp map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			assert.NotNil(t, resp["error"], "Should have error field")
		})
	}
}

// TestIntegration_PresignedURL_FileTypeValidation tests file type validation
// **Validates: Requirements 1.2, 1.5.2, 1.7.2**
func TestIntegration_PresignedURL_FileTypeValidation(t *testing.T) {
	db := setupAttachmentIntegrationTestDB(t)

	cfg := &config.S3Config{
		Bucket:    "test-bucket",
		Region:    "us-east-1",
		AccessKey: "test-key",
		SecretKey: "test-secret",
	}
	s3Client, err := client.NewS3Client(cfg)
	require.NoError(t, err, "Failed to create S3 client")

	router := setupAttachmentIntegrationRouter(db, s3Client)

	workspaceID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name           string
		fileName       string
		contentType    string
		expectedStatus int
		description    string
	}{
		{
			name:           "Accept JPEG image",
			fileName:       "photo.jpg",
			contentType:    "image/jpeg",
			expectedStatus: http.StatusOK,
			description:    "Should accept JPEG images",
		},
		{
			name:           "Accept PNG image",
			fileName:       "screenshot.png",
			contentType:    "image/png",
			expectedStatus: http.StatusOK,
			description:    "Should accept PNG images",
		},
		{
			name:           "Accept PDF document",
			fileName:       "report.pdf",
			contentType:    "application/pdf",
			expectedStatus: http.StatusOK,
			description:    "Should accept PDF documents",
		},
		{
			name:           "Accept Word document",
			fileName:       "document.docx",
			contentType:    "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			expectedStatus: http.StatusOK,
			description:    "Should accept Word documents",
		},
		{
			name:           "Reject MP3 audio",
			fileName:       "audio.mp3",
			contentType:    "audio/mpeg",
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject MP3 audio files",
		},
		{
			name:           "Reject MP4 video",
			fileName:       "video.mp4",
			contentType:    "video/mp4",
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject MP4 video files",
		},
		{
			name:           "Reject WAV audio",
			fileName:       "sound.wav",
			contentType:    "audio/wav",
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject WAV audio files",
		},
		{
			name:           "Reject AVI video",
			fileName:       "movie.avi",
			contentType:    "video/x-msvideo",
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject AVI video files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			presignedReq := PresignedURLRequest{
				EntityType:  "BOARD",
				WorkspaceID: workspaceID.String(),
				FileName:    tt.fileName,
				FileSize:    1024000,
				ContentType: tt.contentType,
			}

			body, err := json.Marshal(presignedReq)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/attachments/presigned-url", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-User-ID", userID.String())

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "%s - Response body: %s", tt.description, w.Body.String())
		})
	}
}

// TestIntegration_PresignedURL_FileSizeValidation tests file size validation
// **Validates: Requirements 1.1, 2.1**
func TestIntegration_PresignedURL_FileSizeValidation(t *testing.T) {
	db := setupAttachmentIntegrationTestDB(t)

	cfg := &config.S3Config{
		Bucket:    "test-bucket",
		Region:    "us-east-1",
		AccessKey: "test-key",
		SecretKey: "test-secret",
	}
	s3Client, err := client.NewS3Client(cfg)
	require.NoError(t, err, "Failed to create S3 client")

	router := setupAttachmentIntegrationRouter(db, s3Client)

	workspaceID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name           string
		fileSize       int64
		expectedStatus int
		description    string
	}{
		{
			name:           "Accept 1MB file",
			fileSize:       1 * 1024 * 1024,
			expectedStatus: http.StatusOK,
			description:    "Should accept 1MB file",
		},
		{
			name:           "Accept 10MB file",
			fileSize:       10 * 1024 * 1024,
			expectedStatus: http.StatusOK,
			description:    "Should accept 10MB file",
		},
		{
			name:           "Accept exactly 20MB file",
			fileSize:       20 * 1024 * 1024,
			expectedStatus: http.StatusOK,
			description:    "Should accept exactly 20MB file",
		},
		{
			name:           "Reject 21MB file",
			fileSize:       21 * 1024 * 1024,
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject 21MB file",
		},
		{
			name:           "Reject 50MB file",
			fileSize:       50 * 1024 * 1024,
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject 50MB file",
		},
		{
			name:           "Reject zero size file",
			fileSize:       0,
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject zero size file",
		},
		{
			name:           "Reject negative size file",
			fileSize:       -1,
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject negative size file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			presignedReq := PresignedURLRequest{
				EntityType:  "BOARD",
				WorkspaceID: workspaceID.String(),
				FileName:    "test.jpg",
				FileSize:    tt.fileSize,
				ContentType: "image/jpeg",
			}

			body, err := json.Marshal(presignedReq)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/attachments/presigned-url", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-User-ID", userID.String())

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "%s - Response body: %s", tt.description, w.Body.String())
		})
	}
}

// TestIntegration_PresignedURL_CompleteWorkflow tests the complete workflow from presigned URL generation
// **Validates: Requirements 1.3, 1.4, 1.5**
func TestIntegration_PresignedURL_CompleteWorkflow(t *testing.T) {
	db := setupAttachmentIntegrationTestDB(t)

	cfg := &config.S3Config{
		Bucket:    "test-bucket",
		Region:    "us-east-1",
		AccessKey: "test-key",
		SecretKey: "test-secret",
	}
	s3Client, err := client.NewS3Client(cfg)
	require.NoError(t, err, "Failed to create S3 client")

	router := setupAttachmentIntegrationRouter(db, s3Client)

	workspaceID := uuid.New()
	userID := uuid.New()

	// Step 1: Request presigned URL
	presignedReq := PresignedURLRequest{
		EntityType:  "BOARD",
		WorkspaceID: workspaceID.String(),
		FileName:    "complete-workflow-test.jpg",
		FileSize:    1024000,
		ContentType: "image/jpeg",
	}

	body, err := json.Marshal(presignedReq)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/attachments/presigned-url", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "Presigned URL generation should succeed")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data, ok := response["data"].(map[string]interface{})
	require.True(t, ok, "Response should contain data field")

	uploadURL, _ := data["uploadUrl"].(string)
	fileKey, _ := data["fileKey"].(string)
	expiresIn, _ := data["expiresIn"].(float64)

	// Validate presigned URL response
	assert.NotEmpty(t, uploadURL, "Upload URL should not be empty")
	assert.NotEmpty(t, fileKey, "File key should not be empty")
	assert.Equal(t, float64(300), expiresIn, "Expiration should be 300 seconds")
	assert.Contains(t, fileKey, "board/boards/", "File key should contain correct path")
	assert.Contains(t, fileKey, workspaceID.String(), "File key should contain workspace ID")

	// Step 2: Simulate client uploading to S3 (skipped in test)
	// In real scenario, client would PUT file to uploadURL

	// Step 3: Verify file key format
	// File key should be: board/boards/{workspaceId}/{year}/{month}/{uuid}_{timestamp}.ext
	assert.Regexp(t, `^board/boards/[a-f0-9-]+/\d{4}/\d{2}/[a-f0-9-]+_\d+\.jpg$`, fileKey, "File key should match expected format")
}
