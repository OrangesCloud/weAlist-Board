package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"project-board-api/internal/client"
	"project-board-api/internal/domain"
)

// Mock attachment repository for testing
type mockAttachmentRepository struct {
	createFunc         func(ctx context.Context, attachment *domain.Attachment) error
	findByIDFunc       func(ctx context.Context, id uuid.UUID) (*domain.Attachment, error)
	findByEntityIDFunc func(ctx context.Context, entityType domain.EntityType, entityID uuid.UUID) ([]*domain.Attachment, error)
	deleteFunc         func(ctx context.Context, id uuid.UUID) error
}

func (m *mockAttachmentRepository) Create(ctx context.Context, attachment *domain.Attachment) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, attachment)
	}
	// Default: set ID and timestamps
	attachment.ID = uuid.New()
	attachment.CreatedAt = time.Now()
	attachment.UpdatedAt = time.Now()
	return nil
}

func (m *mockAttachmentRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Attachment, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, fmt.Errorf("attachment not found")
}

func (m *mockAttachmentRepository) FindByEntityID(ctx context.Context, entityType domain.EntityType, entityID uuid.UUID) ([]*domain.Attachment, error) {
	if m.findByEntityIDFunc != nil {
		return m.findByEntityIDFunc(ctx, entityType, entityID)
	}
	return []*domain.Attachment{}, nil
}

func (m *mockAttachmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockAttachmentRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.Attachment, error) {
	return nil, nil
}

func (m *mockAttachmentRepository) FindExpiredTempAttachments(ctx context.Context) ([]*domain.Attachment, error) {
	return nil, nil
}

func (m *mockAttachmentRepository) ConfirmAttachments(ctx context.Context, attachmentIDs []uuid.UUID, entityID uuid.UUID) error {
	return nil
}

func (m *mockAttachmentRepository) DeleteBatch(ctx context.Context, attachmentIDs []uuid.UUID) error {
	return nil
}

// setupAttachmentHandler creates a test handler with a mock S3 client
func setupAttachmentHandler(t *testing.T) (*AttachmentHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)

	// Use MockS3Client instead of real S3 client
	mockS3Client := client.NewMockS3Client()

	// Create mock repository
	mockRepo := &mockAttachmentRepository{}

	// Create handler
	handler := NewAttachmentHandler(mockS3Client, mockRepo)

	// Setup router with auth middleware
	router := gin.New()
	// Add middleware to set user_id in context (simulating auth middleware)
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uuid.New())
		c.Next()
	})
	router.POST("/attachments/presigned-url", handler.GeneratePresignedURL)

	return handler, router
}

// TestGeneratePresignedURL_Success tests successful presigned URL generation
func TestGeneratePresignedURL_Success(t *testing.T) {
	_, router := setupAttachmentHandler(t)

	tests := []struct {
		name        string
		entityType  string
		fileName    string
		contentType string
	}{
		{
			name:        "Board image upload",
			entityType:  "BOARD",
			fileName:    "test-image.jpg",
			contentType: "image/jpeg",
		},
		{
			name:        "Comment PDF upload",
			entityType:  "COMMENT",
			fileName:    "document.pdf",
			contentType: "application/pdf",
		},
		{
			name:        "Project PNG upload",
			entityType:  "PROJECT",
			fileName:    "diagram.png",
			contentType: "image/png",
		},
		{
			name:        "Board DOCX upload",
			entityType:  "BOARD",
			fileName:    "report.docx",
			contentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := PresignedURLRequest{
				EntityType:  tt.entityType,
				WorkspaceID: "550e8400-e29b-41d4-a716-446655440000",
				FileName:    tt.fileName,
				FileSize:    1024000, // 1MB
				ContentType: tt.contentType,
			}

			body, err := json.Marshal(reqBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/attachments/presigned-url", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Check response structure
			data, ok := response["data"].(map[string]interface{})
			require.True(t, ok, "Response should contain data field")

			// Verify presigned URL fields
			attachmentID, ok := data["attachmentId"].(string)
			assert.True(t, ok, "Response should contain attachmentId")
			assert.NotEmpty(t, attachmentID, "Attachment ID should not be empty")

			uploadURL, ok := data["uploadUrl"].(string)
			assert.True(t, ok, "Response should contain uploadUrl")
			assert.NotEmpty(t, uploadURL, "Upload URL should not be empty")

			fileKey, ok := data["fileKey"].(string)
			assert.True(t, ok, "Response should contain fileKey")
			assert.NotEmpty(t, fileKey, "File key should not be empty")

			expiresIn, ok := data["expiresIn"].(float64)
			assert.True(t, ok, "Response should contain expiresIn")
			assert.Equal(t, float64(300), expiresIn, "ExpiresIn should be 300 seconds")
		})
	}
}

// TestGeneratePresignedURL_FileSizeExceeded tests file size validation
func TestGeneratePresignedURL_FileSizeExceeded(t *testing.T) {
	_, router := setupAttachmentHandler(t)

	reqBody := PresignedURLRequest{
		EntityType:  "BOARD",
		WorkspaceID: "550e8400-e29b-41d4-a716-446655440000",
		FileName:    "large-file.jpg",
		FileSize:    51 * 1024 * 1024, // 51MB (exceeds 50MB limit)
		ContentType: "image/jpeg",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/attachments/presigned-url", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	errorData, ok := response["error"].(map[string]interface{})
	require.True(t, ok, "Response should contain error field")

	code, ok := errorData["code"].(string)
	assert.True(t, ok)
	assert.Equal(t, "FILE_TOO_LARGE", code)
}

// TestGeneratePresignedURL_InvalidFileType tests file type validation
func TestGeneratePresignedURL_InvalidFileType(t *testing.T) {
	_, router := setupAttachmentHandler(t)

	tests := []struct {
		name        string
		fileName    string
		contentType string
	}{
		{
			name:        "Audio file (mp3)",
			fileName:    "audio.mp3",
			contentType: "audio/mpeg",
		},
		{
			name:        "Video file (mp4)",
			fileName:    "video.mp4",
			contentType: "video/mp4",
		},
		{
			name:        "Unsupported document type",
			fileName:    "diagram.exe",
			contentType: "application/x-msdownload",
		},
		{
			name:        "Mismatched extension and content type",
			fileName:    "image.jpg",
			contentType: "application/pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := PresignedURLRequest{
				EntityType:  "BOARD",
				WorkspaceID: "550e8400-e29b-41d4-a716-446655440000",
				FileName:    tt.fileName,
				FileSize:    1024000,
				ContentType: tt.contentType,
			}

			body, err := json.Marshal(reqBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/attachments/presigned-url", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			errorData, ok := response["error"].(map[string]interface{})
			require.True(t, ok, "Response should contain error field")

			code, ok := errorData["code"].(string)
			assert.True(t, ok)
			assert.Equal(t, "INVALID_FILE_TYPE", code)
		})
	}
}

// TestGeneratePresignedURL_InvalidEntityType tests entity type validation
func TestGeneratePresignedURL_InvalidEntityType(t *testing.T) {
	_, router := setupAttachmentHandler(t)

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
			name:       "Empty entity type",
			entityType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := PresignedURLRequest{
				EntityType:  tt.entityType,
				WorkspaceID: "550e8400-e29b-41d4-a716-446655440000",
				FileName:    "test.jpg",
				FileSize:    1024000,
				ContentType: "image/jpeg",
			}

			body, err := json.Marshal(reqBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/attachments/presigned-url", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			errorData, ok := response["error"].(map[string]interface{})
			require.True(t, ok, "Response should contain error field")

			code, ok := errorData["code"].(string)
			assert.True(t, ok)
			assert.Equal(t, "VALIDATION_ERROR", code)
		})
	}
}

// TestGeneratePresignedURL_InvalidWorkspaceID tests workspace ID validation
func TestGeneratePresignedURL_InvalidWorkspaceID(t *testing.T) {
	_, router := setupAttachmentHandler(t)

	reqBody := PresignedURLRequest{
		EntityType:  "BOARD",
		WorkspaceID: "invalid-uuid",
		FileName:    "test.jpg",
		FileSize:    1024000,
		ContentType: "image/jpeg",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/attachments/presigned-url", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	errorData, ok := response["error"].(map[string]interface{})
	require.True(t, ok, "Response should contain error field")

	code, ok := errorData["code"].(string)
	assert.True(t, ok)
	assert.Equal(t, "VALIDATION_ERROR", code)
}

// TestGeneratePresignedURL_ZeroFileSize tests zero file size validation
func TestGeneratePresignedURL_ZeroFileSize(t *testing.T) {
	_, router := setupAttachmentHandler(t)

	reqBody := PresignedURLRequest{
		EntityType:  "BOARD",
		WorkspaceID: "550e8400-e29b-41d4-a716-446655440000",
		FileName:    "test.jpg",
		FileSize:    0,
		ContentType: "image/jpeg",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/attachments/presigned-url", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	errorData, ok := response["error"].(map[string]interface{})
	require.True(t, ok, "Response should contain error field")

	code, ok := errorData["code"].(string)
	assert.True(t, ok)
	assert.Equal(t, "VALIDATION_ERROR", code)
}

// TestGeneratePresignedURL_MissingExtension tests file without extension
func TestGeneratePresignedURL_MissingExtension(t *testing.T) {
	_, router := setupAttachmentHandler(t)

	reqBody := PresignedURLRequest{
		EntityType:  "BOARD",
		WorkspaceID: "550e8400-e29b-41d4-a716-446655440000",
		FileName:    "testfile",
		FileSize:    1024000,
		ContentType: "image/jpeg",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/attachments/presigned-url", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	errorData, ok := response["error"].(map[string]interface{})
	require.True(t, ok, "Response should contain error field")

	code, ok := errorData["code"].(string)
	assert.True(t, ok)
	assert.Equal(t, "INVALID_FILE_TYPE", code)
}

// TestValidateEntityType tests the validateEntityType function
func TestValidateEntityType(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{"Valid BOARD", "BOARD", false},
		{"Valid board (lowercase)", "board", false},
		{"Valid COMMENT", "COMMENT", false},
		{"Valid PROJECT", "PROJECT", false},
		{"Invalid USER", "USER", true},
		{"Invalid empty", "", true},
		{"Invalid random", "RANDOM", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateEntityType(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateFileType tests the validateFileType function
func TestValidateFileType(t *testing.T) {
	tests := []struct {
		name        string
		fileName    string
		contentType string
		expectError bool
	}{
		{"Valid JPEG", "test.jpg", "image/jpeg", false},
		{"Valid PNG", "test.png", "image/png", false},
		{"Valid PDF", "doc.pdf", "application/pdf", false},
		{"Valid DOCX", "report.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", false},
		{"Invalid MP3", "audio.mp3", "audio/mpeg", true},
		{"Invalid MP4", "video.mp4", "video/mp4", true},
		{"Mismatched type", "image.jpg", "application/pdf", true},
		{"No extension", "file", "image/jpeg", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFileType(tt.fileName, tt.contentType)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestGeneratePresignedURL_AllSupportedImageTypes tests all supported image types
func TestGeneratePresignedURL_AllSupportedImageTypes(t *testing.T) {
	_, router := setupAttachmentHandler(t)

	imageTypes := []struct {
		ext         string
		contentType string
	}{
		{".jpg", "image/jpeg"},
		{".jpeg", "image/jpeg"},
		{".png", "image/png"},
		{".gif", "image/gif"},
		{".webp", "image/webp"},
	}

	for _, img := range imageTypes {
		t.Run("Image type "+img.ext, func(t *testing.T) {
			reqBody := PresignedURLRequest{
				EntityType:  "BOARD",
				WorkspaceID: "550e8400-e29b-41d4-a716-446655440000",
				FileName:    "test" + img.ext,
				FileSize:    1024000,
				ContentType: img.contentType,
			}

			body, err := json.Marshal(reqBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/attachments/presigned-url", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// TestGeneratePresignedURL_AllSupportedDocTypes tests all supported document types
func TestGeneratePresignedURL_AllSupportedDocTypes(t *testing.T) {
	_, router := setupAttachmentHandler(t)

	docTypes := []struct {
		ext         string
		contentType string
	}{
		{".pdf", "application/pdf"},
		{".txt", "text/plain"},
		{".doc", "application/msword"},
		{".docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{".xls", "application/vnd.ms-excel"},
		{".xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{".ppt", "application/vnd.ms-powerpoint"},
		{".pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation"},
	}

	for _, doc := range docTypes {
		t.Run("Document type "+doc.ext, func(t *testing.T) {
			reqBody := PresignedURLRequest{
				EntityType:  "COMMENT",
				WorkspaceID: "550e8400-e29b-41d4-a716-446655440000",
				FileName:    "document" + doc.ext,
				FileSize:    2048000,
				ContentType: doc.contentType,
			}

			body, err := json.Marshal(reqBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/attachments/presigned-url", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// Benchmark tests
func BenchmarkGeneratePresignedURL(b *testing.B) {
	_, router := setupAttachmentHandler(&testing.T{})

	reqBody := PresignedURLRequest{
		EntityType:  "BOARD",
		WorkspaceID: "550e8400-e29b-41d4-a716-446655440000",
		FileName:    "test.jpg",
		FileSize:    1024000,
		ContentType: "image/jpeg",
	}

	body, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/attachments/presigned-url", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
