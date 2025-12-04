package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"project-board-api/internal/client"
	"project-board-api/internal/config"
	"project-board-api/internal/domain"
)

// setupAttachmentHandlerWithRepo는 mock repository와 함께 테스트 핸들러를 생성합니다
func setupAttachmentHandlerWithRepo(t *testing.T, mockRepo *mockAttachmentRepository) (*AttachmentHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)

	// 테스트용 S3 설정 생성
	cfg := &config.S3Config{
		Bucket:    "test-bucket",
		Region:    "us-east-1",
		AccessKey: "test-access-key",
		SecretKey: "test-secret-key",
		Endpoint:  "",
	}

	// S3 클라이언트 생성
	s3Client, err := client.NewS3Client(cfg)
	require.NoError(t, err, "Failed to create S3 client")

	// 제공된 mock repo를 사용하거나 기본값 생성
	if mockRepo == nil {
		mockRepo = &mockAttachmentRepository{}
	}

	// 핸들러 생성
	handler := NewAttachmentHandler(s3Client, mockRepo)

	// 인증 미들웨어가 포함된 라우터 설정
	router := gin.New()
	// user_id를 컨텍스트에 설정하는 미들웨어 추가 (인증 미들웨어 시뮬레이션)
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uuid.New())
		c.Next()
	})
	router.POST("/attachments", handler.SaveAttachmentMetadata)

	return handler, router
}

// TestSaveAttachmentMetadata_Success는 첨부파일 메타데이터 저장 성공 테스트
func TestSaveAttachmentMetadata_Success(t *testing.T) {
	tests := []struct {
		name        string
		entityType  string
		fileKey     string
		fileName    string
		fileSize    int64
		contentType string
	}{
		{
			name:        "Board image attachment",
			entityType:  "BOARD",
			fileKey:     "board/boards/550e8400-e29b-41d4-a716-446655440000/2024/01/test-uuid_1704067200.jpg",
			fileName:    "test-image.jpg",
			fileSize:    1024000,
			contentType: "image/jpeg",
		},
		{
			name:        "Comment PDF attachment",
			entityType:  "COMMENT",
			fileKey:     "board/comments/550e8400-e29b-41d4-a716-446655440000/2024/01/test-uuid_1704067200.pdf",
			fileName:    "document.pdf",
			fileSize:    2048000,
			contentType: "application/pdf",
		},
		{
			name:        "Project PNG attachment",
			entityType:  "PROJECT",
			fileKey:     "board/projects/550e8400-e29b-41d4-a716-446655440000/2024/01/test-uuid_1704067200.png",
			fileName:    "diagram.png",
			fileSize:    512000,
			contentType: "image/png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, router := setupAttachmentHandlerWithRepo(t, nil)

			reqBody := SaveAttachmentMetadataRequest{
				EntityType:  tt.entityType,
				FileKey:     tt.fileKey,
				FileName:    tt.fileName,
				FileSize:    tt.fileSize,
				ContentType: tt.contentType,
			}

			body, err := json.Marshal(reqBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/attachments", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusCreated, w.Code)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// 응답 구조 확인
			data, ok := response["data"].(map[string]interface{})
			require.True(t, ok, "Response should contain data field")

			// 첨부파일 필드 검증
			assert.NotEmpty(t, data["id"], "ID should not be empty")
			assert.Equal(t, tt.entityType, data["entityType"], "Entity type should match")
			assert.Nil(t, data["entityId"], "EntityID should be nil for temporary attachment")
			assert.Equal(t, "TEMP", data["status"], "Status should be TEMP")
			assert.Equal(t, tt.fileName, data["fileName"], "File name should match")
			assert.NotEmpty(t, data["fileUrl"], "File URL should not be empty")
			assert.Equal(t, float64(tt.fileSize), data["fileSize"], "File size should match")
			assert.Equal(t, tt.contentType, data["contentType"], "Content type should match")
			assert.NotEmpty(t, data["uploadedBy"], "UploadedBy should not be empty")
			assert.NotEmpty(t, data["uploadedAt"], "UploadedAt should not be empty")
			assert.NotNil(t, data["expiresAt"], "ExpiresAt should not be nil")
		})
	}
}

// TestSaveAttachmentMetadata_InvalidFileKey는 잘못된 파일 키 검증 테스트
func TestSaveAttachmentMetadata_InvalidFileKey(t *testing.T) {
	tests := []struct {
		name    string
		fileKey string
	}{
		{
			name:    "Empty file key",
			fileKey: "",
		},
		{
			name:    "Invalid prefix",
			fileKey: "user/abc123/2024/01/test.jpg",
		},
		{
			name:    "Missing prefix",
			fileKey: "test.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, router := setupAttachmentHandlerWithRepo(t, nil)

			reqBody := SaveAttachmentMetadataRequest{
				EntityType:  "BOARD",
				FileKey:     tt.fileKey,
				FileName:    "test.jpg",
				FileSize:    1024000,
				ContentType: "image/jpeg",
			}

			body, err := json.Marshal(reqBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/attachments", bytes.NewReader(body))
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

// TestSaveAttachmentMetadata_DBError는 데이터베이스 저장 실패 테스트
func TestSaveAttachmentMetadata_DBError(t *testing.T) {
	mockRepo := &mockAttachmentRepository{
		createFunc: func(ctx context.Context, attachment *domain.Attachment) error {
			return fmt.Errorf("database error")
		},
	}

	_, router := setupAttachmentHandlerWithRepo(t, mockRepo)

	reqBody := SaveAttachmentMetadataRequest{
		EntityType:  "BOARD",
		FileKey:     "board/boards/550e8400-e29b-41d4-a716-446655440000/2024/01/test-uuid_1704067200.jpg",
		FileName:    "test.jpg",
		FileSize:    1024000,
		ContentType: "image/jpeg",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/attachments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	errorData, ok := response["error"].(map[string]interface{})
	require.True(t, ok, "Response should contain error field")

	code, ok := errorData["code"].(string)
	assert.True(t, ok)
	assert.Equal(t, "INTERNAL_ERROR", code)
}

// TestSaveAttachmentMetadata_Unauthorized는 사용자 인증 누락 테스트
func TestSaveAttachmentMetadata_Unauthorized(t *testing.T) {
	// 인증 미들웨어 없이 라우터 생성
	gin.SetMode(gin.TestMode)
	cfg := &config.S3Config{
		Bucket:    "test-bucket",
		Region:    "us-east-1",
		AccessKey: "test-access-key",
		SecretKey: "test-secret-key",
		Endpoint:  "",
	}
	s3Client, err := client.NewS3Client(cfg)
	require.NoError(t, err)
	mockRepo := &mockAttachmentRepository{}
	handler := NewAttachmentHandler(s3Client, mockRepo)
	router := gin.New()
	// 인증 미들웨어 없음 - user_id가 설정되지 않음
	router.POST("/attachments", handler.SaveAttachmentMetadata)

	reqBody := SaveAttachmentMetadataRequest{
		EntityType:  "BOARD",
		FileKey:     "board/boards/550e8400-e29b-41d4-a716-446655440000/2024/01/test-uuid_1704067200.jpg",
		FileName:    "test.jpg",
		FileSize:    1024000,
		ContentType: "image/jpeg",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/attachments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	errorData, ok := response["error"].(map[string]interface{})
	require.True(t, ok, "Response should contain error field")

	code, ok := errorData["code"].(string)
	assert.True(t, ok)
	assert.Equal(t, "UNAUTHORIZED", code)
}

// TestSaveAttachmentMetadata_FileSizeValidation는 파일 크기 검증 테스트
func TestSaveAttachmentMetadata_FileSizeValidation(t *testing.T) {
	tests := []struct {
		name     string
		fileSize int64
	}{
		{
			name:     "Zero file size",
			fileSize: 0,
		},
		{
			name:     "Negative file size",
			fileSize: -1,
		},
		{
			name:     "File size exceeds limit",
			fileSize: 51 * 1024 * 1024, // 51MB
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, router := setupAttachmentHandlerWithRepo(t, nil)

			reqBody := SaveAttachmentMetadataRequest{
				EntityType:  "BOARD",
				FileKey:     "board/boards/550e8400-e29b-41d4-a716-446655440000/2024/01/test-uuid_1704067200.jpg",
				FileName:    "test.jpg",
				FileSize:    tt.fileSize,
				ContentType: "image/jpeg",
			}

			body, err := json.Marshal(reqBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/attachments", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

// TestSaveAttachmentMetadata_InvalidFileType는 파일 타입 검증 테스트
func TestSaveAttachmentMetadata_InvalidFileType(t *testing.T) {
	tests := []struct {
		name        string
		fileName    string
		contentType string
	}{
		{
			name:        "Audio file",
			fileName:    "audio.mp3",
			contentType: "audio/mpeg",
		},
		{
			name:        "Video file",
			fileName:    "video.mp4",
			contentType: "video/mp4",
		},
		{
			name:        "Unsupported type",
			fileName:    "diagram.exe",
			contentType: "application/x-msdownload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, router := setupAttachmentHandlerWithRepo(t, nil)

			reqBody := SaveAttachmentMetadataRequest{
				EntityType:  "BOARD",
				FileKey:     "board/boards/550e8400-e29b-41d4-a716-446655440000/2024/01/test-uuid_1704067200" + tt.fileName[strings.LastIndex(tt.fileName, "."):],
				FileName:    tt.fileName,
				FileSize:    1024000,
				ContentType: tt.contentType,
			}

			body, err := json.Marshal(reqBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/attachments", bytes.NewReader(body))
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

// TestSaveAttachmentMetadata_InvalidEntityType는 엔티티 타입 검증 테스트
func TestSaveAttachmentMetadata_InvalidEntityType(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, router := setupAttachmentHandlerWithRepo(t, nil)

			reqBody := SaveAttachmentMetadataRequest{
				EntityType:  tt.entityType,
				FileKey:     "board/boards/550e8400-e29b-41d4-a716-446655440000/2024/01/test-uuid_1704067200.jpg",
				FileName:    "test.jpg",
				FileSize:    1024000,
				ContentType: "image/jpeg",
			}

			body, err := json.Marshal(reqBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/attachments", bytes.NewReader(body))
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
