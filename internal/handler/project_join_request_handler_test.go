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

	"project-board-api/internal/dto"
	"project-board-api/internal/response"
)

// MockProjectJoinRequestService is a mock implementation of ProjectJoinRequestService
type MockProjectJoinRequestService struct {
	CreateJoinRequestFunc func(ctx context.Context, projectID, userID uuid.UUID, token string) (*dto.ProjectJoinRequestResponse, error)
	GetJoinRequestsFunc   func(ctx context.Context, projectID, userID uuid.UUID, status *string, token string) ([]*dto.ProjectJoinRequestResponse, error)
	UpdateJoinRequestFunc func(ctx context.Context, requestID, userID uuid.UUID, status string, token string) (*dto.ProjectJoinRequestResponse, error)
}

func (m *MockProjectJoinRequestService) CreateJoinRequest(ctx context.Context, projectID, userID uuid.UUID, token string) (*dto.ProjectJoinRequestResponse, error) {
	if m.CreateJoinRequestFunc != nil {
		return m.CreateJoinRequestFunc(ctx, projectID, userID, token)
	}
	return nil, nil
}

func (m *MockProjectJoinRequestService) GetJoinRequests(ctx context.Context, projectID, userID uuid.UUID, status *string, token string) ([]*dto.ProjectJoinRequestResponse, error) {
	if m.GetJoinRequestsFunc != nil {
		return m.GetJoinRequestsFunc(ctx, projectID, userID, status, token)
	}
	return nil, nil
}

func (m *MockProjectJoinRequestService) UpdateJoinRequest(ctx context.Context, requestID, userID uuid.UUID, status string, token string) (*dto.ProjectJoinRequestResponse, error) {
	if m.UpdateJoinRequestFunc != nil {
		return m.UpdateJoinRequestFunc(ctx, requestID, userID, status, token)
	}
	return nil, nil
}

func TestProjectJoinRequestHandler_CreateJoinRequest(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()
	token := "test-jwt-token"

	tests := []struct {
		name           string
		requestBody    interface{}
		setContext     bool
		mockService    func(*MockProjectJoinRequestService)
		expectedStatus int
	}{
		{
			name: "성공: 가입 요청 생성",
			requestBody: dto.CreateProjectJoinRequestRequest{
				ProjectID: projectID,
			},
			setContext: true,
			mockService: func(m *MockProjectJoinRequestService) {
				m.CreateJoinRequestFunc = func(ctx context.Context, pID, uID uuid.UUID, t string) (*dto.ProjectJoinRequestResponse, error) {
					return &dto.ProjectJoinRequestResponse{
						RequestID: uuid.New(),
						ProjectID: pID,
						UserID:    uID,
						Status:    "PENDING",
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "실패: 잘못된 요청 본문",
			requestBody:    "invalid json",
			setContext:     true,
			mockService:    func(m *MockProjectJoinRequestService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "실패: 이미 멤버임",
			requestBody: dto.CreateProjectJoinRequestRequest{
				ProjectID: projectID,
			},
			setContext: true,
			mockService: func(m *MockProjectJoinRequestService) {
				m.CreateJoinRequestFunc = func(ctx context.Context, pID, uID uuid.UUID, t string) (*dto.ProjectJoinRequestResponse, error) {
					return nil, response.NewAppError("ALREADY_MEMBER", "User is already a member of this project", "")
				}
			},
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockService := &MockProjectJoinRequestService{}
			tt.mockService(mockService)
			handler := NewProjectJoinRequestHandler(mockService)

			router := setupTestRouter()

			if tt.setContext {
				router.Use(func(c *gin.Context) {
					c.Set("user_id", userID)
					c.Set("jwtToken", token)
					c.Set("requestId", uuid.New().String())
					c.Next()
				})
			}

			router.POST("/api/projects/join-requests", handler.CreateJoinRequest)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/projects/join-requests", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("CreateJoinRequest() status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestProjectJoinRequestHandler_GetJoinRequests(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()
	token := "test-jwt-token"

	tests := []struct {
		name           string
		projectID      string
		setContext     bool
		mockService    func(*MockProjectJoinRequestService)
		expectedStatus int
	}{
		{
			name:       "성공: 가입 요청 목록 조회",
			projectID:  projectID.String(),
			setContext: true,
			mockService: func(m *MockProjectJoinRequestService) {
				m.GetJoinRequestsFunc = func(ctx context.Context, pID, uID uuid.UUID, status *string, t string) ([]*dto.ProjectJoinRequestResponse, error) {
					return []*dto.ProjectJoinRequestResponse{
						{
							RequestID: uuid.New(),
							ProjectID: pID,
							UserID:    uuid.New(),
							Status:    "PENDING",
						},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "실패: 잘못된 UUID",
			projectID:      "invalid-uuid",
			setContext:     true,
			mockService:    func(m *MockProjectJoinRequestService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "실패: 권한 없음",
			projectID:  projectID.String(),
			setContext: true,
			mockService: func(m *MockProjectJoinRequestService) {
				m.GetJoinRequestsFunc = func(ctx context.Context, pID, uID uuid.UUID, status *string, t string) ([]*dto.ProjectJoinRequestResponse, error) {
					return nil, response.NewForbiddenError("Only project owner or admin can view join requests", "")
				}
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockService := &MockProjectJoinRequestService{}
			tt.mockService(mockService)
			handler := NewProjectJoinRequestHandler(mockService)

			router := setupTestRouter()

			if tt.setContext {
				router.Use(func(c *gin.Context) {
					c.Set("user_id", userID)
					c.Set("jwtToken", token)
					c.Set("requestId", uuid.New().String())
					c.Next()
				})
			}

			router.GET("/api/projects/:projectId/join-requests", handler.GetJoinRequests)

			req := httptest.NewRequest(http.MethodGet, "/api/projects/"+tt.projectID+"/join-requests", nil)
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("GetJoinRequests() status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestProjectJoinRequestHandler_UpdateJoinRequest(t *testing.T) {
	requestID := uuid.New()
	userID := uuid.New()
	token := "test-jwt-token"

	tests := []struct {
		name           string
		requestID      string
		requestBody    interface{}
		setContext     bool
		mockService    func(*MockProjectJoinRequestService)
		expectedStatus int
	}{
		{
			name:      "성공: 가입 요청 승인",
			requestID: requestID.String(),
			requestBody: dto.UpdateProjectJoinRequestRequest{
				Status: "APPROVED",
			},
			setContext: true,
			mockService: func(m *MockProjectJoinRequestService) {
				m.UpdateJoinRequestFunc = func(ctx context.Context, rID, uID uuid.UUID, status string, t string) (*dto.ProjectJoinRequestResponse, error) {
					return &dto.ProjectJoinRequestResponse{
						RequestID: rID,
						ProjectID: uuid.New(),
						UserID:    uuid.New(),
						Status:    status,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "실패: 이미 처리된 요청",
			requestID: requestID.String(),
			requestBody: dto.UpdateProjectJoinRequestRequest{
				Status: "APPROVED",
			},
			setContext: true,
			mockService: func(m *MockProjectJoinRequestService) {
				m.UpdateJoinRequestFunc = func(ctx context.Context, rID, uID uuid.UUID, status string, t string) (*dto.ProjectJoinRequestResponse, error) {
					return nil, response.NewValidationError("Join request has already been processed", "")
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockService := &MockProjectJoinRequestService{}
			tt.mockService(mockService)
			handler := NewProjectJoinRequestHandler(mockService)

			router := setupTestRouter()

			if tt.setContext {
				router.Use(func(c *gin.Context) {
					c.Set("user_id", userID)
					c.Set("jwtToken", token)
					c.Set("requestId", uuid.New().String())
					c.Next()
				})
			}

			router.PUT("/api/projects/join-requests/:joinRequestId", handler.UpdateJoinRequest)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPut, "/api/projects/join-requests/"+tt.requestID, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("UpdateJoinRequest() status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}
