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

// MockProjectMemberService is a mock implementation of ProjectMemberService
type MockProjectMemberService struct {
	GetMembersFunc       func(ctx context.Context, projectID, userID uuid.UUID, token string) ([]*dto.ProjectMemberResponse, error)
	RemoveMemberFunc     func(ctx context.Context, projectID, requesterID, memberID uuid.UUID) error
	UpdateMemberRoleFunc func(ctx context.Context, projectID, requesterID, memberID uuid.UUID, role string) (*dto.ProjectMemberResponse, error)
}

func (m *MockProjectMemberService) GetMembers(ctx context.Context, projectID, userID uuid.UUID, token string) ([]*dto.ProjectMemberResponse, error) {
	if m.GetMembersFunc != nil {
		return m.GetMembersFunc(ctx, projectID, userID, token)
	}
	return nil, nil
}

func (m *MockProjectMemberService) RemoveMember(ctx context.Context, projectID, requesterID, memberID uuid.UUID) error {
	if m.RemoveMemberFunc != nil {
		return m.RemoveMemberFunc(ctx, projectID, requesterID, memberID)
	}
	return nil
}

func (m *MockProjectMemberService) UpdateMemberRole(ctx context.Context, projectID, requesterID, memberID uuid.UUID, role string) (*dto.ProjectMemberResponse, error) {
	if m.UpdateMemberRoleFunc != nil {
		return m.UpdateMemberRoleFunc(ctx, projectID, requesterID, memberID, role)
	}
	return nil, nil
}

func TestProjectMemberHandler_GetMembers(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()
	token := "test-jwt-token"

	tests := []struct {
		name           string
		projectID      string
		setContext     bool
		mockService    func(*MockProjectMemberService)
		expectedStatus int
	}{
		{
			name:       "성공: 멤버 목록 조회",
			projectID:  projectID.String(),
			setContext: true,
			mockService: func(m *MockProjectMemberService) {
				m.GetMembersFunc = func(ctx context.Context, pID, uID uuid.UUID, t string) ([]*dto.ProjectMemberResponse, error) {
					return []*dto.ProjectMemberResponse{
						{
							MemberID:  uuid.New(),
							ProjectID: pID,
							UserID:    uID,
							UserEmail: "member@example.com",
							UserName:  "Member Name",
							RoleName:  "OWNER",
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
			mockService:    func(m *MockProjectMemberService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "실패: Context에 user_id 없음",
			projectID:      projectID.String(),
			setContext:     false,
			mockService:    func(m *MockProjectMemberService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:       "실패: 권한 없음",
			projectID:  projectID.String(),
			setContext: true,
			mockService: func(m *MockProjectMemberService) {
				m.GetMembersFunc = func(ctx context.Context, pID, uID uuid.UUID, t string) ([]*dto.ProjectMemberResponse, error) {
					return nil, response.NewForbiddenError("You are not a member of this project", "")
				}
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockService := &MockProjectMemberService{}
			tt.mockService(mockService)
			handler := NewProjectMemberHandler(mockService)

			router := setupTestRouter()

			if tt.setContext {
				router.Use(func(c *gin.Context) {
					c.Set("user_id", userID)
					c.Set("jwtToken", token)
					c.Set("requestId", uuid.New().String())
					c.Next()
				})
			}

			router.GET("/api/projects/:projectId/members", handler.GetMembers)

			req := httptest.NewRequest(http.MethodGet, "/api/projects/"+tt.projectID+"/members", nil)
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("GetMembers() status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestProjectMemberHandler_RemoveMember(t *testing.T) {
	projectID := uuid.New()
	memberID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name           string
		projectID      string
		memberID       string
		setContext     bool
		mockService    func(*MockProjectMemberService)
		expectedStatus int
	}{
		{
			name:       "성공: 멤버 제거",
			projectID:  projectID.String(),
			memberID:   memberID.String(),
			setContext: true,
			mockService: func(m *MockProjectMemberService) {
				m.RemoveMemberFunc = func(ctx context.Context, pID, rID, mID uuid.UUID) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "실패: 잘못된 Project UUID",
			projectID:      "invalid-uuid",
			memberID:       memberID.String(),
			setContext:     true,
			mockService:    func(m *MockProjectMemberService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "실패: OWNER 제거 시도",
			projectID:  projectID.String(),
			memberID:   memberID.String(),
			setContext: true,
			mockService: func(m *MockProjectMemberService) {
				m.RemoveMemberFunc = func(ctx context.Context, pID, rID, mID uuid.UUID) error {
					return response.NewValidationError("Cannot remove project owner", "")
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockService := &MockProjectMemberService{}
			tt.mockService(mockService)
			handler := NewProjectMemberHandler(mockService)

			router := setupTestRouter()

			if tt.setContext {
				router.Use(func(c *gin.Context) {
					c.Set("user_id", userID)
					c.Set("requestId", uuid.New().String())
					c.Next()
				})
			}

			router.DELETE("/api/projects/:projectId/members/:memberId", handler.RemoveMember)

			req := httptest.NewRequest(http.MethodDelete, "/api/projects/"+tt.projectID+"/members/"+tt.memberID, nil)
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("RemoveMember() status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestProjectMemberHandler_UpdateMemberRole(t *testing.T) {
	projectID := uuid.New()
	memberID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name           string
		projectID      string
		memberID       string
		requestBody    interface{}
		setContext     bool
		mockService    func(*MockProjectMemberService)
		expectedStatus int
	}{
		{
			name:      "성공: 역할 변경",
			projectID: projectID.String(),
			memberID:  memberID.String(),
			requestBody: dto.UpdateProjectMemberRoleRequest{
				RoleName: "ADMIN",
			},
			setContext: true,
			mockService: func(m *MockProjectMemberService) {
				m.UpdateMemberRoleFunc = func(ctx context.Context, pID, rID, mID uuid.UUID, role string) (*dto.ProjectMemberResponse, error) {
					return &dto.ProjectMemberResponse{
						MemberID:  mID,
						ProjectID: pID,
						UserID:    mID,
						RoleName:  role,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "실패: OWNER 역할 변경 시도",
			projectID: projectID.String(),
			memberID:  memberID.String(),
			requestBody: dto.UpdateProjectMemberRoleRequest{
				RoleName: "ADMIN",
			},
			setContext: true,
			mockService: func(m *MockProjectMemberService) {
				m.UpdateMemberRoleFunc = func(ctx context.Context, pID, rID, mID uuid.UUID, role string) (*dto.ProjectMemberResponse, error) {
					return nil, response.NewValidationError("Cannot change project owner role", "")
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockService := &MockProjectMemberService{}
			tt.mockService(mockService)
			handler := NewProjectMemberHandler(mockService)

			router := setupTestRouter()

			if tt.setContext {
				router.Use(func(c *gin.Context) {
					c.Set("user_id", userID)
					c.Set("requestId", uuid.New().String())
					c.Next()
				})
			}

			router.PUT("/api/projects/:projectId/members/:memberId/role", handler.UpdateMemberRole)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPut, "/api/projects/"+tt.projectID+"/members/"+tt.memberID+"/role", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("UpdateMemberRole() status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}
