package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"project-board-api/internal/client"
	"project-board-api/internal/config"
	"project-board-api/internal/domain"
)

// setupDeleteAttachmentHandler creates a test handler for delete operations
func setupDeleteAttachmentHandler(t *testing.T, mockRepo *mockAttachmentRepository) (*AttachmentHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)

	// Create S3 config for testing
	cfg := &config.S3Config{
		Bucket:    "test-bucket",
		Region:    "us-east-1",
		AccessKey: "test-access-key",
		SecretKey: "test-secret-key",
		Endpoint:  "",
	}

	// Create S3 client
	s3Client, err := client.NewS3Client(cfg)
	require.NoError(t, err, "Failed to create S3 client")

	// Create handler
	handler := NewAttachmentHandler(s3Client, mockRepo)

	// Setup router
	router := gin.New()

	// Add middleware to set user_id in context
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"))
		c.Next()
	})

	router.DELETE("/attachments/:attachmentId", handler.DeleteAttachment)

	return handler, router
}

// TestDeleteAttachment_Success tests successful attachment deletion
func TestDeleteAttachment_Success(t *testing.T) {
	attachmentID := uuid.New()
	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	mockRepo := &mockAttachmentRepository{
		findByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Attachment, error) {
			if id == attachmentID {
				return &domain.Attachment{
					BaseModel: domain.BaseModel{
						ID: attachmentID,
					},
					EntityType:  domain.EntityTypeBoard,
					FileName:    "test-file.jpg",
					FileURL:     "https://test-bucket.s3.us-east-1.amazonaws.com/board/boards/workspace123/2024/01/test_1234567890.jpg",
					FileSize:    1024000,
					ContentType: "image/jpeg",
					UploadedBy:  userID,
					Status:      domain.AttachmentStatusConfirmed,
				}, nil
			}
			return nil, fmt.Errorf("attachment not found")
		},
		deleteFunc: func(ctx context.Context, id uuid.UUID) error {
			if id == attachmentID {
				return nil
			}
			return fmt.Errorf("attachment not found")
		},
	}

	_, router := setupDeleteAttachmentHandler(t, mockRepo)

	req := httptest.NewRequest(http.MethodDelete, "/attachments/"+attachmentID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data, ok := response["data"].(map[string]interface{})
	require.True(t, ok, "Response should contain data field")

	message, ok := data["message"].(string)
	assert.True(t, ok)
	assert.Equal(t, "Attachment deleted successfully", message)
}

// TestDeleteAttachment_NotFound tests deletion of non-existent attachment
func TestDeleteAttachment_NotFound(t *testing.T) {
	attachmentID := uuid.New()

	mockRepo := &mockAttachmentRepository{
		findByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Attachment, error) {
			return nil, fmt.Errorf("attachment not found")
		},
	}

	_, router := setupDeleteAttachmentHandler(t, mockRepo)

	req := httptest.NewRequest(http.MethodDelete, "/attachments/"+attachmentID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	errorData, ok := response["error"].(map[string]interface{})
	require.True(t, ok, "Response should contain error field")

	code, ok := errorData["code"].(string)
	assert.True(t, ok)
	assert.Equal(t, "NOT_FOUND", code)
}

// TestDeleteAttachment_Forbidden tests deletion by non-owner
func TestDeleteAttachment_Forbidden(t *testing.T) {
	attachmentID := uuid.New()
	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	differentUserID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")

	mockRepo := &mockAttachmentRepository{
		findByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Attachment, error) {
			if id == attachmentID {
				return &domain.Attachment{
					BaseModel: domain.BaseModel{
						ID: attachmentID,
					},
					EntityType:  domain.EntityTypeBoard,
					FileName:    "test-file.jpg",
					FileURL:     "https://test-bucket.s3.us-east-1.amazonaws.com/board/boards/workspace123/2024/01/test_1234567890.jpg",
					FileSize:    1024000,
					ContentType: "image/jpeg",
					UploadedBy:  differentUserID, // Different user uploaded this
					Status:      domain.AttachmentStatusConfirmed,
				}, nil
			}
			return nil, fmt.Errorf("attachment not found")
		},
	}

	gin.SetMode(gin.TestMode)

	// Create S3 config for testing
	cfg := &config.S3Config{
		Bucket:    "test-bucket",
		Region:    "us-east-1",
		AccessKey: "test-access-key",
		SecretKey: "test-secret-key",
		Endpoint:  "",
	}

	// Create S3 client
	s3Client, err := client.NewS3Client(cfg)
	require.NoError(t, err, "Failed to create S3 client")

	// Create handler
	handler := NewAttachmentHandler(s3Client, mockRepo)

	// Setup router with current user
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID) // Current user
		c.Next()
	})
	router.DELETE("/attachments/:attachmentId", handler.DeleteAttachment)

	req := httptest.NewRequest(http.MethodDelete, "/attachments/"+attachmentID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	errorData, ok := response["error"].(map[string]interface{})
	require.True(t, ok, "Response should contain error field")

	code, ok := errorData["code"].(string)
	assert.True(t, ok)
	assert.Equal(t, "FORBIDDEN", code)
}

// TestDeleteAttachment_InvalidAttachmentID tests deletion with invalid attachment ID
func TestDeleteAttachment_InvalidAttachmentID(t *testing.T) {
	mockRepo := &mockAttachmentRepository{}

	_, router := setupDeleteAttachmentHandler(t, mockRepo)

	req := httptest.NewRequest(http.MethodDelete, "/attachments/invalid-uuid", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	errorData, ok := response["error"].(map[string]interface{})
	require.True(t, ok, "Response should contain error field")

	code, ok := errorData["code"].(string)
	assert.True(t, ok)
	assert.Equal(t, "VALIDATION_ERROR", code)
}
