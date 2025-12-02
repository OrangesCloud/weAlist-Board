package handler

import (
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
	"project-board-api/internal/config"
	"project-board-api/internal/domain"
)

// setupAttachmentRetrievalHandler creates a test handler for retrieval endpoints
func setupAttachmentRetrievalHandler(t *testing.T, mockRepo *mockAttachmentRepository) (*AttachmentHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)

	cfg := &config.S3Config{
		Bucket:    "test-bucket",
		Region:    "us-east-1",
		AccessKey: "test-access-key",
		SecretKey: "test-secret-key",
		Endpoint:  "",
	}

	s3Client, err := client.NewS3Client(cfg)
	require.NoError(t, err, "Failed to create S3 client")

	if mockRepo == nil {
		mockRepo = &mockAttachmentRepository{}
	}

	handler := NewAttachmentHandler(s3Client, mockRepo)

	router := gin.New()
	router.GET("/boards/:boardId/attachments", handler.GetBoardAttachments)
	router.GET("/comments/:commentId/attachments", handler.GetCommentAttachments)
	router.GET("/projects/:projectId/attachments", handler.GetProjectAttachments)

	return handler, router
}

// TestGetBoardAttachments_Success tests successful retrieval of board attachments
func TestGetBoardAttachments_Success(t *testing.T) {
	boardID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	mockAttachments := []*domain.Attachment{
		{
			BaseModel: domain.BaseModel{
				ID:        uuid.New(),
				CreatedAt: now,
				UpdatedAt: now,
			},
			EntityType:  domain.EntityTypeBoard,
			EntityID:    &boardID,
			Status:      domain.AttachmentStatusConfirmed,
			FileName:    "image1.jpg",
			FileURL:     "https://s3.amazonaws.com/bucket/image1.jpg",
			FileSize:    1024000,
			ContentType: "image/jpeg",
			UploadedBy:  userID,
			ExpiresAt:   nil,
		},
		{
			BaseModel: domain.BaseModel{
				ID:        uuid.New(),
				CreatedAt: now.Add(-1 * time.Hour),
				UpdatedAt: now.Add(-1 * time.Hour),
			},
			EntityType:  domain.EntityTypeBoard,
			EntityID:    &boardID,
			Status:      domain.AttachmentStatusConfirmed,
			FileName:    "document.pdf",
			FileURL:     "https://s3.amazonaws.com/bucket/document.pdf",
			FileSize:    2048000,
			ContentType: "application/pdf",
			UploadedBy:  userID,
			ExpiresAt:   nil,
		},
	}

	mockRepo := &mockAttachmentRepository{
		findByEntityIDFunc: func(ctx context.Context, entityType domain.EntityType, entityID uuid.UUID) ([]*domain.Attachment, error) {
			assert.Equal(t, domain.EntityTypeBoard, entityType)
			assert.Equal(t, boardID, entityID)
			return mockAttachments, nil
		},
	}

	_, router := setupAttachmentRetrievalHandler(t, mockRepo)

	req := httptest.NewRequest(http.MethodGet, "/boards/"+boardID.String()+"/attachments", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data, ok := response["data"].([]interface{})
	require.True(t, ok, "Response should contain data array")
	assert.Len(t, data, 2, "Should return 2 attachments")

	// Verify first attachment
	attachment1 := data[0].(map[string]interface{})
	assert.Equal(t, "BOARD", attachment1["entityType"])
	assert.Equal(t, "CONFIRMED", attachment1["status"])
	assert.Equal(t, "image1.jpg", attachment1["fileName"])
	assert.Equal(t, float64(1024000), attachment1["fileSize"])
}

// TestGetBoardAttachments_EmptyResult tests retrieval with no attachments
func TestGetBoardAttachments_EmptyResult(t *testing.T) {
	boardID := uuid.New()

	mockRepo := &mockAttachmentRepository{
		findByEntityIDFunc: func(ctx context.Context, entityType domain.EntityType, entityID uuid.UUID) ([]*domain.Attachment, error) {
			return []*domain.Attachment{}, nil
		},
	}

	_, router := setupAttachmentRetrievalHandler(t, mockRepo)

	req := httptest.NewRequest(http.MethodGet, "/boards/"+boardID.String()+"/attachments", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data, ok := response["data"].([]interface{})
	require.True(t, ok, "Response should contain data array")
	assert.Len(t, data, 0, "Should return empty array")
}

// TestGetBoardAttachments_InvalidBoardID tests invalid board ID
func TestGetBoardAttachments_InvalidBoardID(t *testing.T) {
	_, router := setupAttachmentRetrievalHandler(t, nil)

	req := httptest.NewRequest(http.MethodGet, "/boards/invalid-uuid/attachments", nil)
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

// TestGetBoardAttachments_RepositoryError tests database error
func TestGetBoardAttachments_RepositoryError(t *testing.T) {
	boardID := uuid.New()

	mockRepo := &mockAttachmentRepository{
		findByEntityIDFunc: func(ctx context.Context, entityType domain.EntityType, entityID uuid.UUID) ([]*domain.Attachment, error) {
			return nil, fmt.Errorf("database error")
		},
	}

	_, router := setupAttachmentRetrievalHandler(t, mockRepo)

	req := httptest.NewRequest(http.MethodGet, "/boards/"+boardID.String()+"/attachments", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	errorData, ok := response["error"].(map[string]interface{})
	require.True(t, ok, "Response should contain error field")

	code, ok := errorData["code"].(string)
	assert.True(t, ok)
	assert.Equal(t, "INTERNAL_ERROR", code)
}

// TestGetCommentAttachments_Success tests successful retrieval of comment attachments
func TestGetCommentAttachments_Success(t *testing.T) {
	commentID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	mockAttachments := []*domain.Attachment{
		{
			BaseModel: domain.BaseModel{
				ID:        uuid.New(),
				CreatedAt: now,
				UpdatedAt: now,
			},
			EntityType:  domain.EntityTypeComment,
			EntityID:    &commentID,
			Status:      domain.AttachmentStatusConfirmed,
			FileName:    "screenshot.png",
			FileURL:     "https://s3.amazonaws.com/bucket/screenshot.png",
			FileSize:    512000,
			ContentType: "image/png",
			UploadedBy:  userID,
			ExpiresAt:   nil,
		},
	}

	mockRepo := &mockAttachmentRepository{
		findByEntityIDFunc: func(ctx context.Context, entityType domain.EntityType, entityID uuid.UUID) ([]*domain.Attachment, error) {
			assert.Equal(t, domain.EntityTypeComment, entityType)
			assert.Equal(t, commentID, entityID)
			return mockAttachments, nil
		},
	}

	_, router := setupAttachmentRetrievalHandler(t, mockRepo)

	req := httptest.NewRequest(http.MethodGet, "/comments/"+commentID.String()+"/attachments", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data, ok := response["data"].([]interface{})
	require.True(t, ok, "Response should contain data array")
	assert.Len(t, data, 1, "Should return 1 attachment")

	attachment := data[0].(map[string]interface{})
	assert.Equal(t, "COMMENT", attachment["entityType"])
	assert.Equal(t, "screenshot.png", attachment["fileName"])
}

// TestGetCommentAttachments_EmptyResult tests retrieval with no attachments
func TestGetCommentAttachments_EmptyResult(t *testing.T) {
	commentID := uuid.New()

	mockRepo := &mockAttachmentRepository{
		findByEntityIDFunc: func(ctx context.Context, entityType domain.EntityType, entityID uuid.UUID) ([]*domain.Attachment, error) {
			return []*domain.Attachment{}, nil
		},
	}

	_, router := setupAttachmentRetrievalHandler(t, mockRepo)

	req := httptest.NewRequest(http.MethodGet, "/comments/"+commentID.String()+"/attachments", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data, ok := response["data"].([]interface{})
	require.True(t, ok, "Response should contain data array")
	assert.Len(t, data, 0, "Should return empty array")
}

// TestGetCommentAttachments_InvalidCommentID tests invalid comment ID
func TestGetCommentAttachments_InvalidCommentID(t *testing.T) {
	_, router := setupAttachmentRetrievalHandler(t, nil)

	req := httptest.NewRequest(http.MethodGet, "/comments/invalid-uuid/attachments", nil)
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

// TestGetProjectAttachments_Success tests successful retrieval of project attachments
func TestGetProjectAttachments_Success(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	mockAttachments := []*domain.Attachment{
		{
			BaseModel: domain.BaseModel{
				ID:        uuid.New(),
				CreatedAt: now,
				UpdatedAt: now,
			},
			EntityType:  domain.EntityTypeProject,
			EntityID:    &projectID,
			Status:      domain.AttachmentStatusConfirmed,
			FileName:    "requirements.docx",
			FileURL:     "https://s3.amazonaws.com/bucket/requirements.docx",
			FileSize:    3072000,
			ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			UploadedBy:  userID,
			ExpiresAt:   nil,
		},
		{
			BaseModel: domain.BaseModel{
				ID:        uuid.New(),
				CreatedAt: now.Add(-2 * time.Hour),
				UpdatedAt: now.Add(-2 * time.Hour),
			},
			EntityType:  domain.EntityTypeProject,
			EntityID:    &projectID,
			Status:      domain.AttachmentStatusConfirmed,
			FileName:    "architecture.png",
			FileURL:     "https://s3.amazonaws.com/bucket/architecture.png",
			FileSize:    1536000,
			ContentType: "image/png",
			UploadedBy:  userID,
			ExpiresAt:   nil,
		},
	}

	mockRepo := &mockAttachmentRepository{
		findByEntityIDFunc: func(ctx context.Context, entityType domain.EntityType, entityID uuid.UUID) ([]*domain.Attachment, error) {
			assert.Equal(t, domain.EntityTypeProject, entityType)
			assert.Equal(t, projectID, entityID)
			return mockAttachments, nil
		},
	}

	_, router := setupAttachmentRetrievalHandler(t, mockRepo)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+projectID.String()+"/attachments", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data, ok := response["data"].([]interface{})
	require.True(t, ok, "Response should contain data array")
	assert.Len(t, data, 2, "Should return 2 attachments")

	// Verify attachments
	attachment1 := data[0].(map[string]interface{})
	assert.Equal(t, "PROJECT", attachment1["entityType"])
	assert.Equal(t, "requirements.docx", attachment1["fileName"])

	attachment2 := data[1].(map[string]interface{})
	assert.Equal(t, "PROJECT", attachment2["entityType"])
	assert.Equal(t, "architecture.png", attachment2["fileName"])
}

// TestGetProjectAttachments_EmptyResult tests retrieval with no attachments
func TestGetProjectAttachments_EmptyResult(t *testing.T) {
	projectID := uuid.New()

	mockRepo := &mockAttachmentRepository{
		findByEntityIDFunc: func(ctx context.Context, entityType domain.EntityType, entityID uuid.UUID) ([]*domain.Attachment, error) {
			return []*domain.Attachment{}, nil
		},
	}

	_, router := setupAttachmentRetrievalHandler(t, mockRepo)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+projectID.String()+"/attachments", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data, ok := response["data"].([]interface{})
	require.True(t, ok, "Response should contain data array")
	assert.Len(t, data, 0, "Should return empty array")
}

// TestGetProjectAttachments_InvalidProjectID tests invalid project ID
func TestGetProjectAttachments_InvalidProjectID(t *testing.T) {
	_, router := setupAttachmentRetrievalHandler(t, nil)

	req := httptest.NewRequest(http.MethodGet, "/projects/invalid-uuid/attachments", nil)
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
