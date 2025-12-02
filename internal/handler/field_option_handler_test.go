package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
	"project-board-api/internal/response"
)

// MockFieldOptionService is a mock implementation of FieldOptionService
type MockFieldOptionService struct {
	GetFieldOptionsFunc    func(ctx context.Context, fieldType domain.FieldType) ([]*dto.FieldOptionResponse, error)
	CreateFieldOptionFunc  func(ctx context.Context, req *dto.CreateFieldOptionRequest) (*dto.FieldOptionResponse, error)
	UpdateFieldOptionFunc  func(ctx context.Context, optionID uuid.UUID, req *dto.UpdateFieldOptionRequest) (*dto.FieldOptionResponse, error)
	DeleteFieldOptionFunc  func(ctx context.Context, optionID uuid.UUID) error
}

func (m *MockFieldOptionService) GetFieldOptions(ctx context.Context, fieldType domain.FieldType) ([]*dto.FieldOptionResponse, error) {
	if m.GetFieldOptionsFunc != nil {
		return m.GetFieldOptionsFunc(ctx, fieldType)
	}
	return nil, nil
}

func (m *MockFieldOptionService) CreateFieldOption(ctx context.Context, req *dto.CreateFieldOptionRequest) (*dto.FieldOptionResponse, error) {
	if m.CreateFieldOptionFunc != nil {
		return m.CreateFieldOptionFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockFieldOptionService) UpdateFieldOption(ctx context.Context, optionID uuid.UUID, req *dto.UpdateFieldOptionRequest) (*dto.FieldOptionResponse, error) {
	if m.UpdateFieldOptionFunc != nil {
		return m.UpdateFieldOptionFunc(ctx, optionID, req)
	}
	return nil, nil
}

func (m *MockFieldOptionService) DeleteFieldOption(ctx context.Context, optionID uuid.UUID) error {
	if m.DeleteFieldOptionFunc != nil {
		return m.DeleteFieldOptionFunc(ctx, optionID)
	}
	return nil
}

func TestFieldOptionHandler_GetFieldOptions(t *testing.T) {
	optionID1 := uuid.New()
	optionID2 := uuid.New()

	tests := []struct {
		name           string
		fieldType      string
		mockService    func(*MockFieldOptionService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:      "성공: stage 필드 옵션 조회",
			fieldType: "stage",
			mockService: func(m *MockFieldOptionService) {
				m.GetFieldOptionsFunc = func(ctx context.Context, fieldType domain.FieldType) ([]*dto.FieldOptionResponse, error) {
					return []*dto.FieldOptionResponse{
						{
							OptionID:        optionID1,
							FieldType:       "stage",
							Value:           "pending",
							Label:           "대기",
							Color:           "#F59E0B",
							DisplayOrder:    1,
							IsSystemDefault: true,
						},
						{
							OptionID:        optionID2,
							FieldType:       "stage",
							Value:           "in_progress",
							Label:           "진행중",
							Color:           "#3B82F6",
							DisplayOrder:    2,
							IsSystemDefault: true,
						},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp response.SuccessResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				
				dataBytes, _ := json.Marshal(resp.Data)
				var options []*dto.FieldOptionResponse
				if err := json.Unmarshal(dataBytes, &options); err != nil {
					t.Fatalf("Failed to unmarshal data: %v", err)
				}
				
				if len(options) != 2 {
					t.Errorf("Expected 2 options, got %d", len(options))
				}
				if options[0].FieldType != "stage" {
					t.Errorf("Expected fieldType 'stage', got '%s'", options[0].FieldType)
				}
			},
		},
		{
			name:      "성공: role 필드 옵션 조회",
			fieldType: "role",
			mockService: func(m *MockFieldOptionService) {
				m.GetFieldOptionsFunc = func(ctx context.Context, fieldType domain.FieldType) ([]*dto.FieldOptionResponse, error) {
					return []*dto.FieldOptionResponse{
						{
							OptionID:        optionID1,
							FieldType:       "role",
							Value:           "developer",
							Label:           "개발자",
							Color:           "#8B5CF6",
							DisplayOrder:    1,
							IsSystemDefault: true,
						},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "실패: fieldType 파라미터 누락",
			fieldType:      "",
			mockService:    func(m *MockFieldOptionService) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp response.ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				
				errorData, ok := resp.Error.(map[string]interface{})
				if !ok {
					t.Fatal("Error field is not a map")
				}
				
				if errorData["code"] != response.ErrCodeValidation {
					t.Errorf("Expected error code '%s', got '%s'", response.ErrCodeValidation, errorData["code"])
				}
			},
		},
		{
			name:           "실패: 잘못된 fieldType",
			fieldType:      "invalid_type",
			mockService:    func(m *MockFieldOptionService) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp response.ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				
				errorData, ok := resp.Error.(map[string]interface{})
				if !ok {
					t.Fatal("Error field is not a map")
				}
				
				if errorData["code"] != response.ErrCodeValidation {
					t.Errorf("Expected error code '%s', got '%s'", response.ErrCodeValidation, errorData["code"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockService := &MockFieldOptionService{}
			tt.mockService(mockService)
			handler := NewFieldOptionHandler(mockService)
			
			router := setupTestRouter()
			router.GET("/api/field-options", handler.GetFieldOptions)

			url := "/api/field-options"
			if tt.fieldType != "" {
				url += "?fieldType=" + tt.fieldType
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("GetFieldOptions() status = %v, want %v", w.Code, tt.expectedStatus)
			}
			
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestFieldOptionHandler_CreateFieldOption(t *testing.T) {
	optionID := uuid.New()

	tests := []struct {
		name           string
		requestBody    interface{}
		mockService    func(*MockFieldOptionService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "성공: 필드 옵션 생성",
			requestBody: dto.CreateFieldOptionRequest{
				FieldType:    "stage",
				Value:        "custom_stage",
				Label:        "커스텀 단계",
				Color:        "#FF5733",
				DisplayOrder: 10,
			},
			mockService: func(m *MockFieldOptionService) {
				m.CreateFieldOptionFunc = func(ctx context.Context, req *dto.CreateFieldOptionRequest) (*dto.FieldOptionResponse, error) {
					return &dto.FieldOptionResponse{
						OptionID:        optionID,
						FieldType:       req.FieldType,
						Value:           req.Value,
						Label:           req.Label,
						Color:           req.Color,
						DisplayOrder:    req.DisplayOrder,
						IsSystemDefault: false,
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp response.SuccessResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				
				dataBytes, _ := json.Marshal(resp.Data)
				var option dto.FieldOptionResponse
				if err := json.Unmarshal(dataBytes, &option); err != nil {
					t.Fatalf("Failed to unmarshal data: %v", err)
				}
				
				if option.Value != "custom_stage" {
					t.Errorf("Expected value 'custom_stage', got '%s'", option.Value)
				}
				if option.IsSystemDefault {
					t.Error("Expected IsSystemDefault to be false")
				}
			},
		},
		{
			name:           "실패: 잘못된 요청 본문",
			requestBody:    "invalid json",
			mockService:    func(m *MockFieldOptionService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "실패: 중복된 필드 옵션",
			requestBody: dto.CreateFieldOptionRequest{
				FieldType:    "stage",
				Value:        "pending",
				Label:        "대기",
				Color:        "#F59E0B",
				DisplayOrder: 1,
			},
			mockService: func(m *MockFieldOptionService) {
				m.CreateFieldOptionFunc = func(ctx context.Context, req *dto.CreateFieldOptionRequest) (*dto.FieldOptionResponse, error) {
					return nil, response.NewValidationError("Field option with value 'pending' already exists for field type 'stage'", "")
				}
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp response.ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				
				errorData, ok := resp.Error.(map[string]interface{})
				if !ok {
					t.Fatal("Error field is not a map")
				}
				
				if errorData["code"] != response.ErrCodeValidation {
					t.Errorf("Expected error code '%s', got '%s'", response.ErrCodeValidation, errorData["code"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockService := &MockFieldOptionService{}
			tt.mockService(mockService)
			handler := NewFieldOptionHandler(mockService)
			
			router := setupTestRouter()
			router.POST("/api/field-options", handler.CreateFieldOption)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/field-options", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("CreateFieldOption() status = %v, want %v", w.Code, tt.expectedStatus)
			}
			
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestFieldOptionHandler_UpdateFieldOption(t *testing.T) {
	optionID := uuid.New()
	newLabel := "업데이트된 라벨"
	newColor := "#00FF00"
	newDisplayOrder := 5

	tests := []struct {
		name           string
		optionID       string
		requestBody    interface{}
		mockService    func(*MockFieldOptionService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:     "성공: 필드 옵션 수정",
			optionID: optionID.String(),
			requestBody: dto.UpdateFieldOptionRequest{
				Label:        &newLabel,
				Color:        &newColor,
				DisplayOrder: &newDisplayOrder,
			},
			mockService: func(m *MockFieldOptionService) {
				m.UpdateFieldOptionFunc = func(ctx context.Context, id uuid.UUID, req *dto.UpdateFieldOptionRequest) (*dto.FieldOptionResponse, error) {
					return &dto.FieldOptionResponse{
						OptionID:        id,
						FieldType:       "stage",
						Value:           "pending",
						Label:           *req.Label,
						Color:           *req.Color,
						DisplayOrder:    *req.DisplayOrder,
						IsSystemDefault: true,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp response.SuccessResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				
				dataBytes, _ := json.Marshal(resp.Data)
				var option dto.FieldOptionResponse
				if err := json.Unmarshal(dataBytes, &option); err != nil {
					t.Fatalf("Failed to unmarshal data: %v", err)
				}
				
				if option.Label != newLabel {
					t.Errorf("Expected label '%s', got '%s'", newLabel, option.Label)
				}
			},
		},
		{
			name:           "실패: 잘못된 UUID",
			optionID:       "invalid-uuid",
			requestBody:    dto.UpdateFieldOptionRequest{},
			mockService:    func(m *MockFieldOptionService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "실패: 필드 옵션을 찾을 수 없음",
			optionID: optionID.String(),
			requestBody: dto.UpdateFieldOptionRequest{
				Label: &newLabel,
			},
			mockService: func(m *MockFieldOptionService) {
				m.UpdateFieldOptionFunc = func(ctx context.Context, id uuid.UUID, req *dto.UpdateFieldOptionRequest) (*dto.FieldOptionResponse, error) {
					return nil, response.NewNotFoundError("Field option not found", "")
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockService := &MockFieldOptionService{}
			tt.mockService(mockService)
			handler := NewFieldOptionHandler(mockService)
			
			router := setupTestRouter()
			router.PATCH("/api/field-options/:optionId", handler.UpdateFieldOption)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPatch, "/api/field-options/"+tt.optionID, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("UpdateFieldOption() status = %v, want %v", w.Code, tt.expectedStatus)
			}
			
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestFieldOptionHandler_DeleteFieldOption(t *testing.T) {
	optionID := uuid.New()

	tests := []struct {
		name           string
		optionID       string
		mockService    func(*MockFieldOptionService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:     "성공: 필드 옵션 삭제",
			optionID: optionID.String(),
			mockService: func(m *MockFieldOptionService) {
				m.DeleteFieldOptionFunc = func(ctx context.Context, id uuid.UUID) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "실패: 잘못된 UUID",
			optionID:       "invalid-uuid",
			mockService:    func(m *MockFieldOptionService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "실패: 필드 옵션을 찾을 수 없음",
			optionID: optionID.String(),
			mockService: func(m *MockFieldOptionService) {
				m.DeleteFieldOptionFunc = func(ctx context.Context, id uuid.UUID) error {
					return response.NewNotFoundError("Field option not found", "")
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:     "실패: 시스템 기본 옵션 삭제 시도",
			optionID: optionID.String(),
			mockService: func(m *MockFieldOptionService) {
				m.DeleteFieldOptionFunc = func(ctx context.Context, id uuid.UUID) error {
					return response.NewValidationError("Cannot delete system default field option", "")
				}
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp response.ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				
				errorData, ok := resp.Error.(map[string]interface{})
				if !ok {
					t.Fatal("Error field is not a map")
				}
				
				if errorData["code"] != response.ErrCodeValidation {
					t.Errorf("Expected error code '%s', got '%s'", response.ErrCodeValidation, errorData["code"])
				}
				
				message, ok := errorData["message"].(string)
				if !ok || message != "Cannot delete system default field option" {
					t.Errorf("Expected message about system default, got '%v'", errorData["message"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockService := &MockFieldOptionService{}
			tt.mockService(mockService)
			handler := NewFieldOptionHandler(mockService)
			
			router := setupTestRouter()
			router.DELETE("/api/field-options/:optionId", handler.DeleteFieldOption)

			req := httptest.NewRequest(http.MethodDelete, "/api/field-options/"+tt.optionID, nil)
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("DeleteFieldOption() status = %v, want %v", w.Code, tt.expectedStatus)
			}
			
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}
