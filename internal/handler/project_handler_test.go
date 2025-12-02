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

func TestProjectHandler_UpdateProject(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name           string
		projectID      string
		requestBody    interface{}
		setContext     bool
		mockService    func(*MockProjectService)
		expectedStatus int
	}{
		{
			name:      "성공: Project 수정",
			projectID: projectID.String(),
			requestBody: dto.UpdateProjectRequest{
				Name:        stringPtr("Updated Project"),
				Description: stringPtr("Updated Description"),
			},
			setContext: true,
			mockService: func(m *MockProjectService) {
				m.UpdateProjectFunc = func(ctx context.Context, pID, uID uuid.UUID, req *dto.UpdateProjectRequest) (*dto.ProjectResponse, error) {
					return &dto.ProjectResponse{
						ID:          pID,
						WorkspaceID: uuid.New(),
						OwnerID:     uID,
						Name:        *req.Name,
						Description: *req.Description,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "실패: 잘못된 UUID",
			projectID:      "invalid-uuid",
			requestBody:    dto.UpdateProjectRequest{},
			setContext:     true,
			mockService:    func(m *MockProjectService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "실패: 잘못된 요청 본문",
			projectID:      projectID.String(),
			requestBody:    "invalid json",
			setContext:     true,
			mockService:    func(m *MockProjectService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "실패: OWNER가 아님",
			projectID: projectID.String(),
			requestBody: dto.UpdateProjectRequest{
				Name: stringPtr("Updated Project"),
			},
			setContext: true,
			mockService: func(m *MockProjectService) {
				m.UpdateProjectFunc = func(ctx context.Context, pID, uID uuid.UUID, req *dto.UpdateProjectRequest) (*dto.ProjectResponse, error) {
					return nil, response.NewForbiddenError("Only project owner can update project", "")
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:      "실패: Project가 존재하지 않음",
			projectID: projectID.String(),
			requestBody: dto.UpdateProjectRequest{
				Name: stringPtr("Updated Project"),
			},
			setContext: true,
			mockService: func(m *MockProjectService) {
				m.UpdateProjectFunc = func(ctx context.Context, pID, uID uuid.UUID, req *dto.UpdateProjectRequest) (*dto.ProjectResponse, error) {
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
					c.Set("requestId", uuid.New().String())
					c.Next()
				})
			}

			router.PUT("/api/projects/:projectId", handler.UpdateProject)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPut, "/api/projects/"+tt.projectID, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("UpdateProject() status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestProjectHandler_DeleteProject(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name           string
		projectID      string
		setContext     bool
		mockService    func(*MockProjectService)
		expectedStatus int
	}{
		{
			name:       "성공: Project 삭제",
			projectID:  projectID.String(),
			setContext: true,
			mockService: func(m *MockProjectService) {
				m.DeleteProjectFunc = func(ctx context.Context, pID, uID uuid.UUID) error {
					return nil
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
			name:       "실패: OWNER가 아님",
			projectID:  projectID.String(),
			setContext: true,
			mockService: func(m *MockProjectService) {
				m.DeleteProjectFunc = func(ctx context.Context, pID, uID uuid.UUID) error {
					return response.NewForbiddenError("Only project owner can delete project", "")
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:       "실패: Project가 존재하지 않음",
			projectID:  projectID.String(),
			setContext: true,
			mockService: func(m *MockProjectService) {
				m.DeleteProjectFunc = func(ctx context.Context, pID, uID uuid.UUID) error {
					return response.NewNotFoundError("Project not found", "")
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
					c.Set("requestId", uuid.New().String())
					c.Next()
				})
			}

			router.DELETE("/api/projects/:projectId", handler.DeleteProject)

			req := httptest.NewRequest(http.MethodDelete, "/api/projects/"+tt.projectID, nil)
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("DeleteProject() status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestProjectHandler_SearchProjects(t *testing.T) {
	workspaceID := uuid.New()
	userID := uuid.New()
	token := "test-jwt-token"

	tests := []struct {
		name           string
		queryParams    map[string]string
		setContext     bool
		mockService    func(*MockProjectService)
		expectedStatus int
	}{
		{
			name: "성공: Project 검색",
			queryParams: map[string]string{
				"workspaceId": workspaceID.String(),
				"query":       "test",
				"page":        "1",
				"limit":       "10",
			},
			setContext: true,
			mockService: func(m *MockProjectService) {
				m.SearchProjectsFunc = func(ctx context.Context, wID, uID uuid.UUID, query string, page, limit int, t string) (*dto.PaginatedProjectsResponse, error) {
					return &dto.PaginatedProjectsResponse{
						Projects: []dto.ProjectResponse{
							{
								ID:          uuid.New(),
								WorkspaceID: wID,
								Name:        "Test Project",
							},
						},
						Page:  page,
						Limit: limit,
						Total: 1,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "실패: workspaceId 누락",
			queryParams: map[string]string{
				"query": "test",
			},
			setContext:     true,
			mockService:    func(m *MockProjectService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "실패: query 누락",
			queryParams: map[string]string{
				"workspaceId": workspaceID.String(),
			},
			setContext:     true,
			mockService:    func(m *MockProjectService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "실패: 잘못된 workspaceId",
			queryParams: map[string]string{
				"workspaceId": "invalid-uuid",
				"query":       "test",
			},
			setContext:     true,
			mockService:    func(m *MockProjectService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "실패: Workspace 멤버가 아님",
			queryParams: map[string]string{
				"workspaceId": workspaceID.String(),
				"query":       "test",
			},
			setContext: true,
			mockService: func(m *MockProjectService) {
				m.SearchProjectsFunc = func(ctx context.Context, wID, uID uuid.UUID, query string, page, limit int, t string) (*dto.PaginatedProjectsResponse, error) {
					return nil, response.NewForbiddenError("You are not a member of this workspace", "")
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

			if tt.setContext {
				router.Use(func(c *gin.Context) {
					c.Set("user_id", userID)
					c.Set("jwtToken", token)
					c.Set("requestId", uuid.New().String())
					c.Next()
				})
			}

			router.GET("/api/projects/search", handler.SearchProjects)

			url := "/api/projects/search?"
			for k, v := range tt.queryParams {
				url += k + "=" + v + "&"
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("SearchProjects() status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestProjectHandler_GetProjectInitSettings(t *testing.T) {
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
			name:       "성공: 초기 설정 조회",
			projectID:  projectID.String(),
			setContext: true,
			mockService: func(m *MockProjectService) {
				m.GetProjectInitSettingsFunc = func(ctx context.Context, pID, uID uuid.UUID, t string) (*dto.ProjectInitSettingsResponse, error) {
					return &dto.ProjectInitSettingsResponse{
						Project: dto.ProjectBasicInfo{
							ProjectID:   pID,
							WorkspaceID: uuid.New(),
							Name:        "Test Project",
						},
						Fields:     []dto.FieldWithOptionsResponse{},
						FieldTypes: []dto.FieldTypeInfo{},
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
				m.GetProjectInitSettingsFunc = func(ctx context.Context, pID, uID uuid.UUID, t string) (*dto.ProjectInitSettingsResponse, error) {
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
				m.GetProjectInitSettingsFunc = func(ctx context.Context, pID, uID uuid.UUID, t string) (*dto.ProjectInitSettingsResponse, error) {
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

			router.GET("/api/projects/:projectId/init-settings", handler.GetProjectInitSettings)

			req := httptest.NewRequest(http.MethodGet, "/api/projects/"+tt.projectID+"/init-settings", nil)
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("GetProjectInitSettings() status = %v, want %v", w.Code, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				var resp map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &resp)
				if _, ok := resp["requestId"]; !ok {
					t.Error("GetProjectInitSettings() response missing requestId field")
				}
			}
		})
	}
}

func TestProjectHandler_GetProjectInitSettings_WithFieldOptions(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()
	token := "test-jwt-token"

	t.Run("성공: 새 프로젝트의 초기 설정에 12개의 field options 반환", func(t *testing.T) {
		// Given
		mockService := &MockProjectService{}
		mockService.GetProjectInitSettingsFunc = func(ctx context.Context, pID, uID uuid.UUID, t string) (*dto.ProjectInitSettingsResponse, error) {
			// Stage options (4개)
			stageOptions := []dto.FieldOption{
				{OptionID: uuid.New().String(), OptionLabel: "대기", OptionValue: "pending", Color: "#F59E0B", DisplayOrder: 1, FieldID: "stage"},
				{OptionID: uuid.New().String(), OptionLabel: "진행중", OptionValue: "in_progress", Color: "#3B82F6", DisplayOrder: 2, FieldID: "stage"},
				{OptionID: uuid.New().String(), OptionLabel: "검토", OptionValue: "review", Color: "#8B5CF6", DisplayOrder: 3, FieldID: "stage"},
				{OptionID: uuid.New().String(), OptionLabel: "완료", OptionValue: "approved", Color: "#10B981", DisplayOrder: 4, FieldID: "stage"},
			}

			// Importance options (4개)
			importanceOptions := []dto.FieldOption{
				{OptionID: uuid.New().String(), OptionLabel: "긴급", OptionValue: "urgent", Color: "#EF4444", DisplayOrder: 1, FieldID: "importance"},
				{OptionID: uuid.New().String(), OptionLabel: "높음", OptionValue: "high", Color: "#F97316", DisplayOrder: 2, FieldID: "importance"},
				{OptionID: uuid.New().String(), OptionLabel: "보통", OptionValue: "normal", Color: "#10B981", DisplayOrder: 3, FieldID: "importance"},
				{OptionID: uuid.New().String(), OptionLabel: "낮음", OptionValue: "low", Color: "#6B7280", DisplayOrder: 4, FieldID: "importance"},
			}

			// Role options (4개)
			roleOptions := []dto.FieldOption{
				{OptionID: uuid.New().String(), OptionLabel: "개발자", OptionValue: "developer", Color: "#8B5CF6", DisplayOrder: 1, FieldID: "role"},
				{OptionID: uuid.New().String(), OptionLabel: "기획자", OptionValue: "planner", Color: "#EC4899", DisplayOrder: 2, FieldID: "role"},
				{OptionID: uuid.New().String(), OptionLabel: "디자이너", OptionValue: "designer", Color: "#F59E0B", DisplayOrder: 3, FieldID: "role"},
				{OptionID: uuid.New().String(), OptionLabel: "QA", OptionValue: "qa", Color: "#06B6D4", DisplayOrder: 4, FieldID: "role"},
			}

			return &dto.ProjectInitSettingsResponse{
				Project: dto.ProjectBasicInfo{
					ProjectID:   pID,
					WorkspaceID: uuid.New(),
					Name:        "Test Project",
				},
				Fields: []dto.FieldWithOptionsResponse{
					{
						FieldID:     "stage",
						FieldName:   "Stage",
						FieldType:   "select",
						IsRequired:  true,
						Description: "Current stage of the board",
						Options:     stageOptions,
					},
					{
						FieldID:     "importance",
						FieldName:   "Importance",
						FieldType:   "select",
						IsRequired:  true,
						Description: "Priority level of the board",
						Options:     importanceOptions,
					},
					{
						FieldID:     "role",
						FieldName:   "Role",
						FieldType:   "select",
						IsRequired:  true,
						Description: "Role responsible for the board",
						Options:     roleOptions,
					},
				},
				FieldTypes: []dto.FieldTypeInfo{},
			}, nil
		}

		handler := NewProjectHandler(mockService)
		router := setupTestRouter()
		router.Use(func(c *gin.Context) {
			c.Set("user_id", userID)
			c.Set("jwtToken", token)
			c.Set("requestId", uuid.New().String())
			c.Next()
		})
		router.GET("/api/projects/:projectId/init-settings", handler.GetProjectInitSettings)

		req := httptest.NewRequest(http.MethodGet, "/api/projects/"+projectID.String()+"/init-settings", nil)
		w := httptest.NewRecorder()

		// When
		router.ServeHTTP(w, req)

		// Then
		if w.Code != http.StatusOK {
			t.Errorf("GetProjectInitSettings() status = %v, want %v", w.Code, http.StatusOK)
		}

		var response struct {
			Success bool `json:"success"`
			Data    struct {
				Project dto.ProjectBasicInfo             `json:"project"`
				Fields  []dto.FieldWithOptionsResponse `json:"fields"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Verify 3 fields are returned
		if len(response.Data.Fields) != 3 {
			t.Errorf("Expected 3 fields, got %d", len(response.Data.Fields))
		}

		// Count total options across all fields
		totalOptions := 0
		for _, field := range response.Data.Fields {
			totalOptions += len(field.Options)
		}

		// Verify 12 total field options are returned
		if totalOptions != 12 {
			t.Errorf("Expected 12 total field options, got %d", totalOptions)
		}

		// Verify stage field has 4 options
		var stageField *dto.FieldWithOptionsResponse
		for i := range response.Data.Fields {
			if response.Data.Fields[i].FieldID == "stage" {
				stageField = &response.Data.Fields[i]
				break
			}
		}
		if stageField == nil {
			t.Error("Stage field not found")
		} else if len(stageField.Options) != 4 {
			t.Errorf("Expected stage field to have 4 options, got %d", len(stageField.Options))
		}

		// Verify importance field has 4 options
		var importanceField *dto.FieldWithOptionsResponse
		for i := range response.Data.Fields {
			if response.Data.Fields[i].FieldID == "importance" {
				importanceField = &response.Data.Fields[i]
				break
			}
		}
		if importanceField == nil {
			t.Error("Importance field not found")
		} else if len(importanceField.Options) != 4 {
			t.Errorf("Expected importance field to have 4 options, got %d", len(importanceField.Options))
		}

		// Verify role field has 4 options
		var roleField *dto.FieldWithOptionsResponse
		for i := range response.Data.Fields {
			if response.Data.Fields[i].FieldID == "role" {
				roleField = &response.Data.Fields[i]
				break
			}
		}
		if roleField == nil {
			t.Error("Role field not found")
		} else if len(roleField.Options) != 4 {
			t.Errorf("Expected role field to have 4 options, got %d", len(roleField.Options))
		}
	})
}

// Helper function for string pointers
func stringPtr(s string) *string {
	return &s
}
