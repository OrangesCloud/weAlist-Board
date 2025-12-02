package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
	"project-board-api/internal/response"
)

// TestErrorResponseConsistency verifies that all API error responses follow the standard format
// Error responses should be: {"error": {"code": "...", "message": "..."}, "requestId": "..."}
func TestErrorResponseConsistency(t *testing.T) {
	tests := []struct {
		name           string
		setupHandler   func() (*gin.Engine, string, string, []byte)
		expectedStatus int
		expectedCode   string
	}{
		{
			name: "FieldOption: 잘못된 fieldType 파라미터",
			setupHandler: func() (*gin.Engine, string, string, []byte) {
				mockService := &MockFieldOptionService{}
				handler := NewFieldOptionHandler(mockService)
				router := setupTestRouter()
				router.GET("/api/field-options", handler.GetFieldOptions)
				return router, http.MethodGet, "/api/field-options?fieldType=invalid", nil
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   response.ErrCodeValidation,
		},
		{
			name: "FieldOption: fieldType 파라미터 누락",
			setupHandler: func() (*gin.Engine, string, string, []byte) {
				mockService := &MockFieldOptionService{}
				handler := NewFieldOptionHandler(mockService)
				router := setupTestRouter()
				router.GET("/api/field-options", handler.GetFieldOptions)
				return router, http.MethodGet, "/api/field-options", nil
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   response.ErrCodeValidation,
		},
		{
			name: "FieldOption: 잘못된 UUID",
			setupHandler: func() (*gin.Engine, string, string, []byte) {
				mockService := &MockFieldOptionService{}
				handler := NewFieldOptionHandler(mockService)
				router := setupTestRouter()
				router.DELETE("/api/field-options/:optionId", handler.DeleteFieldOption)
				return router, http.MethodDelete, "/api/field-options/invalid-uuid", nil
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   response.ErrCodeValidation,
		},
		{
			name: "FieldOption: 존재하지 않는 옵션",
			setupHandler: func() (*gin.Engine, string, string, []byte) {
				mockService := &MockFieldOptionService{}
				mockService.DeleteFieldOptionFunc = func(ctx context.Context, optionID uuid.UUID) error {
					return response.NewNotFoundError("Field option not found", "")
				}
				handler := NewFieldOptionHandler(mockService)
				router := setupTestRouter()
				router.DELETE("/api/field-options/:optionId", handler.DeleteFieldOption)
				return router, http.MethodDelete, "/api/field-options/" + uuid.New().String(), nil
			},
			expectedStatus: http.StatusNotFound,
			expectedCode:   response.ErrCodeNotFound,
		},
		{
			name: "FieldOption: 시스템 기본 옵션 삭제 시도",
			setupHandler: func() (*gin.Engine, string, string, []byte) {
				mockService := &MockFieldOptionService{}
				mockService.DeleteFieldOptionFunc = func(ctx context.Context, optionID uuid.UUID) error {
					return response.NewValidationError("Cannot delete system default field option", "")
				}
				handler := NewFieldOptionHandler(mockService)
				router := setupTestRouter()
				router.DELETE("/api/field-options/:optionId", handler.DeleteFieldOption)
				return router, http.MethodDelete, "/api/field-options/" + uuid.New().String(), nil
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   response.ErrCodeValidation,
		},
		{
			name: "Board: 잘못된 UUID",
			setupHandler: func() (*gin.Engine, string, string, []byte) {
				mockService := &MockBoardService{}
				handler := NewBoardHandler(mockService)
				router := setupTestRouter()
				router.GET("/api/boards/:boardId", handler.GetBoard)
				return router, http.MethodGet, "/api/boards/invalid-uuid", nil
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   response.ErrCodeValidation,
		},
		{
			name: "Board: 존재하지 않는 보드",
			setupHandler: func() (*gin.Engine, string, string, []byte) {
				mockService := &MockBoardService{}
				mockService.GetBoardFunc = func(ctx context.Context, boardID uuid.UUID) (*dto.BoardDetailResponse, error) {
					return nil, response.NewAppError(response.ErrCodeNotFound, "Board not found", "")
				}
				handler := NewBoardHandler(mockService)
				router := setupTestRouter()
				router.GET("/api/boards/:boardId", handler.GetBoard)
				return router, http.MethodGet, "/api/boards/" + uuid.New().String(), nil
			},
			expectedStatus: http.StatusNotFound,
			expectedCode:   response.ErrCodeNotFound,
		},
		{
			name: "Board: 잘못된 CustomFields JSON",
			setupHandler: func() (*gin.Engine, string, string, []byte) {
				mockService := &MockBoardService{}
				handler := NewBoardHandler(mockService)
				router := setupTestRouter()
				router.GET("/api/boards/project/:projectId", handler.GetBoardsByProject)
				return router, http.MethodGet, "/api/boards/project/" + uuid.New().String() + "?customFields=invalid-json", nil
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   response.ErrCodeValidation,
		},
		{
			name: "Board: 존재하지 않는 프로젝트",
			setupHandler: func() (*gin.Engine, string, string, []byte) {
				mockService := &MockBoardService{}
				mockService.GetBoardsByProjectFunc = func(ctx context.Context, projectID uuid.UUID, filters *dto.BoardFilters) ([]*dto.BoardResponse, error) {
					return nil, response.NewAppError(response.ErrCodeNotFound, "Project not found", "")
				}
				handler := NewBoardHandler(mockService)
				router := setupTestRouter()
				router.GET("/api/boards/project/:projectId", handler.GetBoardsByProject)
				return router, http.MethodGet, "/api/boards/project/" + uuid.New().String(), nil
			},
			expectedStatus: http.StatusNotFound,
			expectedCode:   response.ErrCodeNotFound,
		},
		{
			name: "Board: 잘못된 요청 본문",
			setupHandler: func() (*gin.Engine, string, string, []byte) {
				mockService := &MockBoardService{}
				handler := NewBoardHandler(mockService)
				router := setupTestRouter()
				router.POST("/api/boards", handler.CreateBoard)
				return router, http.MethodPost, "/api/boards", []byte("invalid json")
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   response.ErrCodeValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			router, method, url, body := tt.setupHandler()
			
			var req *http.Request
			if body != nil {
				req = httptest.NewRequest(method, url, bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(method, url, nil)
			}
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Verify error response structure
			var errorResp response.ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &errorResp); err != nil {
				t.Fatalf("Failed to unmarshal error response: %v\nBody: %s", err, w.Body.String())
			}

			// Verify error field exists
			if errorResp.Error == nil {
				t.Fatal("Error field is missing in response")
			}

			// Verify error is a map with code and message
			errorData, ok := errorResp.Error.(map[string]interface{})
			if !ok {
				t.Fatalf("Error field is not a map, got type: %T", errorResp.Error)
			}

			// Verify code field exists and matches expected
			code, ok := errorData["code"].(string)
			if !ok {
				t.Fatal("Error code field is missing or not a string")
			}
			if code != tt.expectedCode {
				t.Errorf("Expected error code '%s', got '%s'", tt.expectedCode, code)
			}

			// Verify message field exists and is not empty
			message, ok := errorData["message"].(string)
			if !ok {
				t.Fatal("Error message field is missing or not a string")
			}
			if message == "" {
				t.Error("Error message is empty")
			}

			// Verify requestId exists
			if errorResp.RequestID == "" {
				t.Error("RequestID is missing in error response")
			}

			t.Logf("✓ Error response format is consistent: {\"error\": {\"code\": \"%s\", \"message\": \"%s\"}, \"requestId\": \"%s\"}", 
				code, message, errorResp.RequestID)
		})
	}
}

// TestSuccessResponseConsistency verifies that all API success responses follow the standard format
// Success responses should be: {"data": {...}, "requestId": "..."}
func TestSuccessResponseConsistency(t *testing.T) {
	tests := []struct {
		name         string
		setupHandler func() (*gin.Engine, string, string, []byte)
	}{
		{
			name: "FieldOption: 필드 옵션 조회 성공",
			setupHandler: func() (*gin.Engine, string, string, []byte) {
				mockService := &MockFieldOptionService{}
				mockService.GetFieldOptionsFunc = func(ctx context.Context, fieldType domain.FieldType) ([]*dto.FieldOptionResponse, error) {
					return []*dto.FieldOptionResponse{
						{
							OptionID:        uuid.New(),
							FieldType:       "stage",
							Value:           "pending",
							Label:           "대기",
							Color:           "#F59E0B",
							DisplayOrder:    1,
							IsSystemDefault: true,
						},
					}, nil
				}
				handler := NewFieldOptionHandler(mockService)
				router := setupTestRouter()
				router.GET("/api/field-options", handler.GetFieldOptions)
				return router, http.MethodGet, "/api/field-options?fieldType=stage", nil
			},
		},
		{
			name: "FieldOption: 필드 옵션 생성 성공",
			setupHandler: func() (*gin.Engine, string, string, []byte) {
				mockService := &MockFieldOptionService{}
				mockService.CreateFieldOptionFunc = func(ctx context.Context, req *dto.CreateFieldOptionRequest) (*dto.FieldOptionResponse, error) {
					return &dto.FieldOptionResponse{
						OptionID:        uuid.New(),
						FieldType:       req.FieldType,
						Value:           req.Value,
						Label:           req.Label,
						Color:           req.Color,
						DisplayOrder:    req.DisplayOrder,
						IsSystemDefault: false,
					}, nil
				}
				handler := NewFieldOptionHandler(mockService)
				router := setupTestRouter()
				router.POST("/api/field-options", handler.CreateFieldOption)
				
				reqBody := dto.CreateFieldOptionRequest{
					FieldType:    "stage",
					Value:        "custom",
					Label:        "커스텀",
					Color:        "#FF5733",
					DisplayOrder: 10,
				}
				body, _ := json.Marshal(reqBody)
				return router, http.MethodPost, "/api/field-options", body
			},
		},
		{
			name: "Board: 보드 생성 성공",
			setupHandler: func() (*gin.Engine, string, string, []byte) {
				mockService := &MockBoardService{}
				mockService.CreateBoardFunc = func(ctx context.Context, req *dto.CreateBoardRequest) (*dto.BoardResponse, error) {
					return &dto.BoardResponse{
						ID:           uuid.New(),
						ProjectID:    req.ProjectID,
						Title:        req.Title,
						Content:      req.Content,
						CustomFields: req.CustomFields,
					}, nil
				}
				handler := NewBoardHandler(mockService)
				router := setupTestRouter()
				router.POST("/api/boards", handler.CreateBoard)
				
				reqBody := dto.CreateBoardRequest{
					ProjectID: uuid.New(),
					Title:     "Test Board",
					Content:   "Test Content",
					CustomFields: map[string]interface{}{
						"stage": "in_progress",
					},
				}
				body, _ := json.Marshal(reqBody)
				return router, http.MethodPost, "/api/boards", body
			},
		},
		{
			name: "Board: 보드 목록 조회 성공",
			setupHandler: func() (*gin.Engine, string, string, []byte) {
				mockService := &MockBoardService{}
				mockService.GetBoardsByProjectFunc = func(ctx context.Context, projectID uuid.UUID, filters *dto.BoardFilters) ([]*dto.BoardResponse, error) {
					return []*dto.BoardResponse{
						{
							ID:        uuid.New(),
							ProjectID: projectID,
							Title:     "Board 1",
							CustomFields: map[string]interface{}{
								"stage": "in_progress",
							},
						},
					}, nil
				}
				handler := NewBoardHandler(mockService)
				router := setupTestRouter()
				router.GET("/api/boards/project/:projectId", handler.GetBoardsByProject)
				return router, http.MethodGet, "/api/boards/project/" + uuid.New().String(), nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			router, method, url, body := tt.setupHandler()
			
			var req *http.Request
			if body != nil {
				req = httptest.NewRequest(method, url, bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(method, url, nil)
			}
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code < 200 || w.Code >= 300 {
				t.Fatalf("Expected success status, got %d", w.Code)
			}

			// Verify success response structure
			var successResp response.SuccessResponse
			if err := json.Unmarshal(w.Body.Bytes(), &successResp); err != nil {
				t.Fatalf("Failed to unmarshal success response: %v\nBody: %s", err, w.Body.String())
			}

			// Verify data field exists
			if successResp.Data == nil {
				t.Fatal("Data field is missing in response")
			}

			// Verify requestId exists
			if successResp.RequestID == "" {
				t.Error("RequestID is missing in success response")
			}

			t.Logf("✓ Success response format is consistent: {\"data\": {...}, \"requestId\": \"%s\"}", successResp.RequestID)
		})
	}
}
