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

// MockProjectService is a mock implementation of ProjectService
type MockProjectService struct {
	CreateProjectFunc          func(ctx context.Context, req *dto.CreateProjectRequest, userID uuid.UUID, token string) (*dto.ProjectResponse, error)
	GetProjectsByWorkspaceFunc func(ctx context.Context, workspaceID, userID uuid.UUID, token string) ([]*dto.ProjectResponse, error)
	GetDefaultProjectFunc      func(ctx context.Context, workspaceID, userID uuid.UUID, token string) (*dto.ProjectResponse, error)
	GetProjectFunc             func(ctx context.Context, projectID, userID uuid.UUID, token string) (*dto.ProjectResponse, error)
	UpdateProjectFunc          func(ctx context.Context, projectID, userID uuid.UUID, req *dto.UpdateProjectRequest) (*dto.ProjectResponse, error)
	DeleteProjectFunc          func(ctx context.Context, projectID, userID uuid.UUID) error
	SearchProjectsFunc         func(ctx context.Context, workspaceID, userID uuid.UUID, query string, page, limit int, token string) (*dto.PaginatedProjectsResponse, error)
	GetProjectInitSettingsFunc func(ctx context.Context, projectID, userID uuid.UUID, token string) (*dto.ProjectInitSettingsResponse, error)
}

func (m *MockProjectService) CreateProject(ctx context.Context, req *dto.CreateProjectRequest, userID uuid.UUID, token string) (*dto.ProjectResponse, error) {
	if m.CreateProjectFunc != nil {
		return m.CreateProjectFunc(ctx, req, userID, token)
	}
	return nil, nil
}

func (m *MockProjectService) GetProjectsByWorkspace(ctx context.Context, workspaceID, userID uuid.UUID, token string) ([]*dto.ProjectResponse, error) {
	if m.GetProjectsByWorkspaceFunc != nil {
		return m.GetProjectsByWorkspaceFunc(ctx, workspaceID, userID, token)
	}
	return nil, nil
}

func (m *MockProjectService) GetDefaultProject(ctx context.Context, workspaceID, userID uuid.UUID, token string) (*dto.ProjectResponse, error) {
	if m.GetDefaultProjectFunc != nil {
		return m.GetDefaultProjectFunc(ctx, workspaceID, userID, token)
	}
	return nil, nil
}

func (m *MockProjectService) GetProject(ctx context.Context, projectID, userID uuid.UUID, token string) (*dto.ProjectResponse, error) {
	if m.GetProjectFunc != nil {
		return m.GetProjectFunc(ctx, projectID, userID, token)
	}
	return nil, nil
}

func (m *MockProjectService) UpdateProject(ctx context.Context, projectID, userID uuid.UUID, req *dto.UpdateProjectRequest) (*dto.ProjectResponse, error) {
	if m.UpdateProjectFunc != nil {
		return m.UpdateProjectFunc(ctx, projectID, userID, req)
	}
	return nil, nil
}

func (m *MockProjectService) DeleteProject(ctx context.Context, projectID, userID uuid.UUID) error {
	if m.DeleteProjectFunc != nil {
		return m.DeleteProjectFunc(ctx, projectID, userID)
	}
	return nil
}

func (m *MockProjectService) SearchProjects(ctx context.Context, workspaceID, userID uuid.UUID, query string, page, limit int, token string) (*dto.PaginatedProjectsResponse, error) {
	if m.SearchProjectsFunc != nil {
		return m.SearchProjectsFunc(ctx, workspaceID, userID, query, page, limit, token)
	}
	return nil, nil
}

func (m *MockProjectService) GetProjectInitSettings(ctx context.Context, projectID, userID uuid.UUID, token string) (*dto.ProjectInitSettingsResponse, error) {
	if m.GetProjectInitSettingsFunc != nil {
		return m.GetProjectInitSettingsFunc(ctx, projectID, userID, token)
	}
	return nil, nil
}

func TestProjectHandler_CreateProject(t *testing.T) {
	workspaceID := uuid.New()
	projectID := uuid.New()
	userID := uuid.New()
	token := "test-jwt-token"

	tests := []struct {
		name           string
		requestBody    interface{}
		setContext     bool
		mockService    func(*MockProjectService)
		expectedStatus int
	}{
		{
			name: "성공: Project 생성",
			requestBody: dto.CreateProjectRequest{
				WorkspaceID: workspaceID,
				Name:        "Test Project",
				Description: "Test Description",
			},
			setContext: true,
			mockService: func(m *MockProjectService) {
				m.CreateProjectFunc = func(ctx context.Context, req *dto.CreateProjectRequest, uID uuid.UUID, t string) (*dto.ProjectResponse, error) {
					return &dto.ProjectResponse{
						ID:          projectID,
						WorkspaceID: req.WorkspaceID,
						OwnerID:     uID,
						Name:        req.Name,
						Description: req.Description,
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "실패: 잘못된 요청 본문",
			requestBody:    "invalid json",
			setContext:     true,
			mockService:    func(m *MockProjectService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "실패: Context에 user_id 없음",
			requestBody: dto.CreateProjectRequest{
				WorkspaceID: workspaceID,
				Name:        "Test Project",
			},
			setContext:     false,
			mockService:    func(m *MockProjectService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "실패: Workspace 멤버십 검증 실패",
			requestBody: dto.CreateProjectRequest{
				WorkspaceID: workspaceID,
				Name:        "Test Project",
			},
			setContext: true,
			mockService: func(m *MockProjectService) {
				m.CreateProjectFunc = func(ctx context.Context, req *dto.CreateProjectRequest, uID uuid.UUID, t string) (*dto.ProjectResponse, error) {
					return nil, response.NewAppError(response.ErrCodeForbidden, "You are not a member of this workspace", "")
				}
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockService := &MockProjectService{}
			tt.mockService(mockService)
			handler := NewProjectHandler(mockService)

			router := setupTestRouter()

			// Add middleware to set context values
			if tt.setContext {
				router.Use(func(c *gin.Context) {
					c.Set("user_id", userID)
					c.Set("jwtToken", token)
					c.Set("requestId", uuid.New().String())
					c.Next()
				})
			}

			router.POST("/api/projects", handler.CreateProject)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("CreateProject() status = %v, want %v", w.Code, tt.expectedStatus)
			}

			// Verify response structure includes requestId
			if tt.expectedStatus == http.StatusCreated {
				var resp map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &resp)
				if _, ok := resp["requestId"]; !ok {
					t.Error("CreateProject() response missing requestId field")
				}
			}
		})
	}
}

func TestProjectHandler_GetProjectsByWorkspace(t *testing.T) {
	workspaceID := uuid.New()
	userID := uuid.New()
	token := "test-jwt-token"

	tests := []struct {
		name           string
		workspaceID    string
		setContext     bool
		mockService    func(*MockProjectService)
		expectedStatus int
	}{
		{
			name:        "성공: Workspace의 Project 목록 조회",
			workspaceID: workspaceID.String(),
			setContext:  true,
			mockService: func(m *MockProjectService) {
				m.GetProjectsByWorkspaceFunc = func(ctx context.Context, wID, uID uuid.UUID, t string) ([]*dto.ProjectResponse, error) {
					return []*dto.ProjectResponse{
						{
							ID:          uuid.New(),
							WorkspaceID: wID,
							OwnerID:     uID,
							Name:        "Project 1",
							OwnerEmail:  "owner@example.com",
							OwnerName:   "Owner Name",
						},
						{
							ID:          uuid.New(),
							WorkspaceID: wID,
							OwnerID:     uID,
							Name:        "Project 2",
						},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "실패: 잘못된 UUID",
			workspaceID:    "invalid-uuid",
			setContext:     true,
			mockService:    func(m *MockProjectService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "실패: Context에 user_id 없음",
			workspaceID:    workspaceID.String(),
			setContext:     false,
			mockService:    func(m *MockProjectService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:        "실패: Workspace 멤버십 검증 실패",
			workspaceID: workspaceID.String(),
			setContext:  true,
			mockService: func(m *MockProjectService) {
				m.GetProjectsByWorkspaceFunc = func(ctx context.Context, wID, uID uuid.UUID, t string) ([]*dto.ProjectResponse, error) {
					return nil, response.NewAppError(response.ErrCodeForbidden, "You are not a member of this workspace", "")
				}
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockService := &MockProjectService{}
			tt.mockService(mockService)
			handler := NewProjectHandler(mockService)

			router := setupTestRouter()

			// Add middleware to set context values
			if tt.setContext {
				router.Use(func(c *gin.Context) {
					c.Set("user_id", userID)
					c.Set("jwtToken", token)
					c.Set("requestId", uuid.New().String())
					c.Next()
				})
			}

			router.GET("/api/projects/workspace/:workspaceId", handler.GetProjectsByWorkspace)

			req := httptest.NewRequest(http.MethodGet, "/api/projects/workspace/"+tt.workspaceID, nil)
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("GetProjectsByWorkspace() status = %v, want %v", w.Code, tt.expectedStatus)
			}

			// Verify response structure includes requestId
			if tt.expectedStatus == http.StatusOK {
				var resp map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &resp)
				if _, ok := resp["requestId"]; !ok {
					t.Error("GetProjectsByWorkspace() response missing requestId field")
				}
			}
		})
	}
}

func TestProjectHandler_GetDefaultProject(t *testing.T) {
	workspaceID := uuid.New()
	projectID := uuid.New()
	userID := uuid.New()
	token := "test-jwt-token"

	tests := []struct {
		name           string
		workspaceID    string
		setContext     bool
		mockService    func(*MockProjectService)
		expectedStatus int
	}{
		{
			name:        "성공: Default Project 조회",
			workspaceID: workspaceID.String(),
			setContext:  true,
			mockService: func(m *MockProjectService) {
				m.GetDefaultProjectFunc = func(ctx context.Context, wID, uID uuid.UUID, t string) (*dto.ProjectResponse, error) {
					return &dto.ProjectResponse{
						ID:          projectID,
						WorkspaceID: wID,
						OwnerID:     uID,
						Name:        "Default Project",
						OwnerEmail:  "default@example.com",
						OwnerName:   "Default Owner",
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "실패: 잘못된 UUID",
			workspaceID:    "invalid-uuid",
			setContext:     true,
			mockService:    func(m *MockProjectService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "실패: Context에 user_id 없음",
			workspaceID:    workspaceID.String(),
			setContext:     false,
			mockService:    func(m *MockProjectService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:        "실패: Default Project가 존재하지 않음",
			workspaceID: workspaceID.String(),
			setContext:  true,
			mockService: func(m *MockProjectService) {
				m.GetDefaultProjectFunc = func(ctx context.Context, wID, uID uuid.UUID, t string) (*dto.ProjectResponse, error) {
					return nil, response.NewAppError(response.ErrCodeNotFound, "Default project not found", "")
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockService := &MockProjectService{}
			tt.mockService(mockService)
			handler := NewProjectHandler(mockService)

			router := setupTestRouter()

			// Add middleware to set context values
			if tt.setContext {
				router.Use(func(c *gin.Context) {
					c.Set("user_id", userID)
					c.Set("jwtToken", token)
					c.Set("requestId", uuid.New().String())
					c.Next()
				})
			}

			router.GET("/api/projects/workspace/:workspaceId/default", handler.GetDefaultProject)

			req := httptest.NewRequest(http.MethodGet, "/api/projects/workspace/"+tt.workspaceID+"/default", nil)
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("GetDefaultProject() status = %v, want %v", w.Code, tt.expectedStatus)
			}

			// Verify response structure includes requestId
			if tt.expectedStatus == http.StatusOK {
				var resp map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &resp)
				if _, ok := resp["requestId"]; !ok {
					t.Error("GetDefaultProject() response missing requestId field")
				}
			}
		})
	}
}

func TestProjectHandler_GetProject(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()
	token := "test-jwt-token"

	tests := []struct {
		name           string
		projectID      string
		setContext     bool
		mockService    func(*MockProjectService)
		expectedStatus int
	}{
		{
			name:       "성공: Project 상세 조회",
			projectID:  projectID.String(),
			setContext: true,
			mockService: func(m *MockProjectService) {
				m.GetProjectFunc = func(ctx context.Context, pID, uID uuid.UUID, t string) (*dto.ProjectResponse, error) {
					return &dto.ProjectResponse{
						ID:          pID,
						WorkspaceID: uuid.New(),
						OwnerID:     uID,
						Name:        "Test Project",
						Description: "Test Description",
						OwnerEmail:  "owner@example.com",
						OwnerName:   "Owner Name",
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "실패: 잘못된 UUID",
			projectID:      "invalid-uuid",
			setContext:     true,
			mockService:    func(m *MockProjectService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "실패: Context에 user_id 없음",
			projectID:      projectID.String(),
			setContext:     false,
			mockService:    func(m *MockProjectService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:       "실패: 프로젝트 멤버가 아님",
			projectID:  projectID.String(),
			setContext: true,
			mockService: func(m *MockProjectService) {
				m.GetProjectFunc = func(ctx context.Context, pID, uID uuid.UUID, t string) (*dto.ProjectResponse, error) {
					return nil, response.NewForbiddenError("You are not a member of this project", "")
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:       "실패: Project가 존재하지 않음",
			projectID:  projectID.String(),
			setContext: true,
			mockService: func(m *MockProjectService) {
				m.GetProjectFunc = func(ctx context.Context, pID, uID uuid.UUID, t string) (*dto.ProjectResponse, error) {
					return nil, response.NewNotFoundError("Project not found", "")
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockService := &MockProjectService{}
			tt.mockService(mockService)
			handler := NewProjectHandler(mockService)

			router := setupTestRouter()

			if tt.setContext {
				router.Use(func(c *gin.Context) {
					c.Set("user_id", userID)
					c.Set("jwtToken", token)
					c.Set("requestId", uuid.New().String())
					c.Next()
				})
			}

			router.GET("/api/projects/:projectId", handler.GetProject)

			req := httptest.NewRequest(http.MethodGet, "/api/projects/"+tt.projectID, nil)
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("GetProject() status = %v, want %v", w.Code, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				var resp map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &resp)
				if _, ok := resp["requestId"]; !ok {
					t.Error("GetProject() response missing requestId field")
				}
			}
		})
	}
}
