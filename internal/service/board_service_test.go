package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
	"project-board-api/internal/response"
)

// MockBoardRepository is a mock implementation of BoardRepository
type MockBoardRepository struct {
	CreateFunc         func(ctx context.Context, board *domain.Board) error
	FindByIDFunc       func(ctx context.Context, id uuid.UUID) (*domain.Board, error)
	FindByProjectIDFunc func(ctx context.Context, projectID uuid.UUID, filters interface{}) ([]*domain.Board, error)
	UpdateFunc         func(ctx context.Context, board *domain.Board) error
	DeleteFunc         func(ctx context.Context, id uuid.UUID) error
}

func (m *MockBoardRepository) Create(ctx context.Context, board *domain.Board) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, board)
	}
	return nil
}

func (m *MockBoardRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockBoardRepository) FindByProjectID(ctx context.Context, projectID uuid.UUID, filters interface{}) ([]*domain.Board, error) {
	if m.FindByProjectIDFunc != nil {
		return m.FindByProjectIDFunc(ctx, projectID, filters)
	}
	return nil, nil
}

func (m *MockBoardRepository) Update(ctx context.Context, board *domain.Board) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, board)
	}
	return nil
}

func (m *MockBoardRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

// MockProjectRepository is a mock implementation of ProjectRepository
type MockProjectRepository struct {
	CreateFunc                       func(ctx context.Context, project *domain.Project) error
	FindByIDFunc                     func(ctx context.Context, id uuid.UUID) (*domain.Project, error)
	FindByWorkspaceIDFunc            func(ctx context.Context, workspaceID uuid.UUID) ([]*domain.Project, error)
	FindDefaultByWorkspaceIDFunc     func(ctx context.Context, workspaceID uuid.UUID) (*domain.Project, error)
	UpdateFunc                       func(ctx context.Context, project *domain.Project) error
	DeleteFunc                       func(ctx context.Context, id uuid.UUID) error
	SearchFunc                       func(ctx context.Context, workspaceID uuid.UUID, query string, page, limit int) ([]*domain.Project, int64, error)
	AddMemberFunc                    func(ctx context.Context, member *domain.ProjectMember) error
	FindMemberByProjectAndUserFunc   func(ctx context.Context, projectID, userID uuid.UUID) (*domain.ProjectMember, error)
	RemoveMemberFunc                 func(ctx context.Context, memberID uuid.UUID) error
	UpdateMemberRoleFunc             func(ctx context.Context, memberID uuid.UUID, role domain.ProjectRole) error
	IsProjectMemberFunc              func(ctx context.Context, projectID, userID uuid.UUID) (bool, error)
	FindMembersByProjectIDFunc       func(ctx context.Context, projectID uuid.UUID) ([]*domain.ProjectMember, error)
	CreateJoinRequestFunc            func(ctx context.Context, request *domain.ProjectJoinRequest) error
	FindJoinRequestByIDFunc          func(ctx context.Context, id uuid.UUID) (*domain.ProjectJoinRequest, error)
	FindJoinRequestsByProjectIDFunc  func(ctx context.Context, projectID uuid.UUID, status *domain.ProjectJoinRequestStatus) ([]*domain.ProjectJoinRequest, error)
	FindPendingByProjectAndUserFunc  func(ctx context.Context, projectID, userID uuid.UUID) (*domain.ProjectJoinRequest, error)
	UpdateJoinRequestStatusFunc      func(ctx context.Context, id uuid.UUID, status domain.ProjectJoinRequestStatus) error
}

func (m *MockProjectRepository) Create(ctx context.Context, project *domain.Project) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, project)
	}
	return nil
}

func (m *MockProjectRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockProjectRepository) FindByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) ([]*domain.Project, error) {
	if m.FindByWorkspaceIDFunc != nil {
		return m.FindByWorkspaceIDFunc(ctx, workspaceID)
	}
	return nil, nil
}

func (m *MockProjectRepository) FindDefaultByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) (*domain.Project, error) {
	if m.FindDefaultByWorkspaceIDFunc != nil {
		return m.FindDefaultByWorkspaceIDFunc(ctx, workspaceID)
	}
	return nil, nil
}

func (m *MockProjectRepository) Update(ctx context.Context, project *domain.Project) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, project)
	}
	return nil
}

func (m *MockProjectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *MockProjectRepository) Search(ctx context.Context, workspaceID uuid.UUID, query string, page, limit int) ([]*domain.Project, int64, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, workspaceID, query, page, limit)
	}
	return nil, 0, nil
}

func (m *MockProjectRepository) AddMember(ctx context.Context, member *domain.ProjectMember) error {
	if m.AddMemberFunc != nil {
		return m.AddMemberFunc(ctx, member)
	}
	return nil
}

func (m *MockProjectRepository) FindMembersByProjectID(ctx context.Context, projectID uuid.UUID) ([]*domain.ProjectMember, error) {
	if m.FindMembersByProjectIDFunc != nil {
		return m.FindMembersByProjectIDFunc(ctx, projectID)
	}
	return nil, nil
}

func (m *MockProjectRepository) FindMemberByProjectAndUser(ctx context.Context, projectID, userID uuid.UUID) (*domain.ProjectMember, error) {
	if m.FindMemberByProjectAndUserFunc != nil {
		return m.FindMemberByProjectAndUserFunc(ctx, projectID, userID)
	}
	return nil, nil
}

func (m *MockProjectRepository) RemoveMember(ctx context.Context, memberID uuid.UUID) error {
	if m.RemoveMemberFunc != nil {
		return m.RemoveMemberFunc(ctx, memberID)
	}
	return nil
}

func (m *MockProjectRepository) UpdateMemberRole(ctx context.Context, memberID uuid.UUID, role domain.ProjectRole) error {
	if m.UpdateMemberRoleFunc != nil {
		return m.UpdateMemberRoleFunc(ctx, memberID, role)
	}
	return nil
}

func (m *MockProjectRepository) IsProjectMember(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	if m.IsProjectMemberFunc != nil {
		return m.IsProjectMemberFunc(ctx, projectID, userID)
	}
	return false, nil
}

func (m *MockProjectRepository) CreateJoinRequest(ctx context.Context, request *domain.ProjectJoinRequest) error {
	if m.CreateJoinRequestFunc != nil {
		return m.CreateJoinRequestFunc(ctx, request)
	}
	return nil
}

func (m *MockProjectRepository) FindJoinRequestByID(ctx context.Context, id uuid.UUID) (*domain.ProjectJoinRequest, error) {
	if m.FindJoinRequestByIDFunc != nil {
		return m.FindJoinRequestByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockProjectRepository) FindJoinRequestsByProjectID(ctx context.Context, projectID uuid.UUID, status *domain.ProjectJoinRequestStatus) ([]*domain.ProjectJoinRequest, error) {
	if m.FindJoinRequestsByProjectIDFunc != nil {
		return m.FindJoinRequestsByProjectIDFunc(ctx, projectID, status)
	}
	return nil, nil
}

func (m *MockProjectRepository) FindPendingByProjectAndUser(ctx context.Context, projectID, userID uuid.UUID) (*domain.ProjectJoinRequest, error) {
	if m.FindPendingByProjectAndUserFunc != nil {
		return m.FindPendingByProjectAndUserFunc(ctx, projectID, userID)
	}
	return nil, nil
}

func (m *MockProjectRepository) UpdateJoinRequestStatus(ctx context.Context, id uuid.UUID, status domain.ProjectJoinRequestStatus) error {
	if m.UpdateJoinRequestStatusFunc != nil {
		return m.UpdateJoinRequestStatusFunc(ctx, id, status)
	}
	return nil
}

func TestBoardService_CreateBoard(t *testing.T) {
	projectID := uuid.New()
	validUserID := uuid.New()
	
	tests := []struct {
		name          string
		req           *dto.CreateBoardRequest
		ctx           context.Context
		mockProject   func(*MockProjectRepository)
		mockBoard     func(*MockBoardRepository)
		wantErr       bool
		wantErrCode   string
	}{
		{
			name: "성공: 정상적인 Board 생성",
			ctx:  context.WithValue(context.Background(), "user_id", validUserID),
			req: &dto.CreateBoardRequest{
				ProjectID:  projectID,
				Title:      "Test Board",
				Content:    "Test Content",
				CustomFields: map[string]interface{}{
					"stage":      "in_progress",
					"importance": "urgent",
					"role":       "developer",
				},
			},
			mockProject: func(m *MockProjectRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return &domain.Project{}, nil
				}
			},
			mockBoard: func(m *MockBoardRepository) {
				m.CreateFunc = func(ctx context.Context, board *domain.Board) error {
					board.ID = uuid.New()
					board.CreatedAt = time.Now()
					board.UpdatedAt = time.Now()
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "성공: CustomFields 없이 Board 생성",
			ctx:  context.WithValue(context.Background(), "user_id", validUserID),
			req: &dto.CreateBoardRequest{
				ProjectID: projectID,
				Title:     "Test Board",
				Content:   "Test Content",
			},
			mockProject: func(m *MockProjectRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return &domain.Project{}, nil
				}
			},
			mockBoard: func(m *MockBoardRepository) {
				m.CreateFunc = func(ctx context.Context, board *domain.Board) error {
					board.ID = uuid.New()
					board.CreatedAt = time.Now()
					board.UpdatedAt = time.Now()
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "실패: Context에 user_id가 없음",
			ctx:  context.Background(),
			req: &dto.CreateBoardRequest{
				ProjectID: projectID,
				Title:     "Test Board",
				Content:   "Test Content",
			},
			mockProject: func(m *MockProjectRepository) {},
			mockBoard:   func(m *MockBoardRepository) {},
			wantErr:     true,
			wantErrCode: response.ErrCodeUnauthorized,
		},
		{
			name: "실패: Project가 존재하지 않음",
			ctx:  context.WithValue(context.Background(), "user_id", validUserID),
			req: &dto.CreateBoardRequest{
				ProjectID:  projectID,
				Title:      "Test Board",
				Content:    "Test Content",
				CustomFields: map[string]interface{}{
					"stage": "in_progress",
				},
			},
			mockProject: func(m *MockProjectRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return nil, gorm.ErrRecordNotFound
				}
			},
			mockBoard: func(m *MockBoardRepository) {},
			wantErr:     true,
			wantErrCode: response.ErrCodeNotFound,
		},
		{
			name: "실패: Board 생성 중 DB 에러",
			ctx:  context.WithValue(context.Background(), "user_id", validUserID),
			req: &dto.CreateBoardRequest{
				ProjectID:  projectID,
				Title:      "Test Board",
				Content:    "Test Content",
				CustomFields: map[string]interface{}{
					"stage": "in_progress",
				},
			},
			mockProject: func(m *MockProjectRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return &domain.Project{}, nil
				}
			},
			mockBoard: func(m *MockBoardRepository) {
				m.CreateFunc = func(ctx context.Context, board *domain.Board) error {
					return errors.New("database error")
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockProjectRepo := &MockProjectRepository{}
			mockBoardRepo := &MockBoardRepository{}
			mockFieldOptionRepo := &MockFieldOptionRepository{}
			mockConverter := &MockFieldOptionConverter{}
			tt.mockProject(mockProjectRepo)
			tt.mockBoard(mockBoardRepo)
			
			mockParticipantRepo := &MockParticipantRepository{}
			logger, _ := zap.NewDevelopment()
			service := NewBoardService(mockBoardRepo, mockProjectRepo, mockFieldOptionRepo, mockParticipantRepo, &MockAttachmentRepository{}, nil, mockConverter, nil, logger)

			// When
			got, err := service.CreateBoard(tt.ctx, tt.req)

			// Then
			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateBoard() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if appErr, ok := err.(*response.AppError); ok {
					if appErr.Code != tt.wantErrCode {
						t.Errorf("CreateBoard() error code = %v, want %v", appErr.Code, tt.wantErrCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("CreateBoard() unexpected error = %v", err)
					return
				}
				if got == nil {
					t.Error("CreateBoard() returned nil response")
					return
				}
				if got.Title != tt.req.Title {
					t.Errorf("CreateBoard() Title = %v, want %v", got.Title, tt.req.Title)
				}
				// Verify CustomFields are preserved
				if tt.req.CustomFields != nil {
					if got.CustomFields == nil {
						t.Error("CreateBoard() CustomFields = nil, want non-nil")
					}
				}
			}
		})
	}
}

func TestBoardService_CreateBoard_CustomFields(t *testing.T) {
	projectID := uuid.New()
	
	tests := []struct {
		name         string
		customFields map[string]interface{}
		wantFields   map[string]interface{}
	}{
		{
			name: "CustomFields 저장: stage, role, importance",
			customFields: map[string]interface{}{
				"stage":      "in_progress",
				"role":       "developer",
				"importance": "urgent",
			},
			wantFields: map[string]interface{}{
				"stage":      "in_progress",
				"role":       "developer",
				"importance": "urgent",
			},
		},
		{
			name: "CustomFields 저장: stage만",
			customFields: map[string]interface{}{
				"stage": "pending",
			},
			wantFields: map[string]interface{}{
				"stage": "pending",
			},
		},
		{
			name:         "CustomFields 저장: 빈 맵",
			customFields: map[string]interface{}{},
			wantFields:   map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockProjectRepo := &MockProjectRepository{
				FindByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return &domain.Project{}, nil
				},
			}
			
			var savedBoard *domain.Board
			mockBoardRepo := &MockBoardRepository{
				CreateFunc: func(ctx context.Context, board *domain.Board) error {
					savedBoard = board
					board.ID = uuid.New()
					board.CreatedAt = time.Now()
					board.UpdatedAt = time.Now()
					return nil
				},
			}
			
			mockFieldOptionRepo := &MockFieldOptionRepository{}
			mockConverter := &MockFieldOptionConverter{}
			mockParticipantRepo := &MockParticipantRepository{}
			logger, _ := zap.NewDevelopment()
			service := NewBoardService(mockBoardRepo, mockProjectRepo, mockFieldOptionRepo, mockParticipantRepo, &MockAttachmentRepository{}, nil, mockConverter, nil, logger)
			
			req := &dto.CreateBoardRequest{
				ProjectID:    projectID,
				Title:        "Test Board",
				Content:      "Test Content",
				CustomFields: tt.customFields,
			}

			// Create context with user_id (as uuid.UUID type)
			ctx := context.WithValue(context.Background(), "user_id", uuid.New())

			// When
			got, err := service.CreateBoard(ctx, req)

			// Then
			if err != nil {
				t.Errorf("CreateBoard() unexpected error = %v", err)
				return
			}
			
			// Verify CustomFields were saved to domain model
			if savedBoard == nil {
				t.Fatal("Board was not saved")
			}
			
			if len(tt.wantFields) > 0 {
				if savedBoard.CustomFields == nil {
					t.Error("Board.CustomFields = nil, want non-nil")
					return
				}
				
				var customFields map[string]interface{}
				if err := json.Unmarshal(savedBoard.CustomFields, &customFields); err != nil {
					t.Errorf("Failed to unmarshal CustomFields: %v", err)
					return
				}
				
				for key, expectedValue := range tt.wantFields {
					if actualValue, ok := customFields[key]; !ok {
						t.Errorf("Board.CustomFields[%s] not found", key)
					} else if actualValue != expectedValue {
						t.Errorf("Board.CustomFields[%s] = %v, want %v", key, actualValue, expectedValue)
					}
				}
			}
			
			// Verify CustomFields are in response
			if got.CustomFields == nil && len(tt.wantFields) > 0 {
				t.Error("Response.CustomFields = nil, want non-nil")
			}
		})
	}
}

func TestBoardService_GetBoard(t *testing.T) {
	boardID := uuid.New()
	
	tests := []struct {
		name        string
		boardID     uuid.UUID
		mockBoard   func(*MockBoardRepository)
		wantErr     bool
		wantErrCode string
	}{
		{
			name:    "성공: Board 조회",
			boardID: boardID,
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					customFieldsJSON, _ := json.Marshal(map[string]interface{}{
						"stage":      "in_progress",
						"importance": "urgent",
						"role":       "developer",
					})
					return &domain.Board{
						BaseModel: domain.BaseModel{
							ID:        boardID,
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						},
						Title:        "Test Board",
						Content:      "Test Content",
						CustomFields: customFieldsJSON,
						Participants: []domain.Participant{},
						Comments:     []domain.Comment{},
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name:    "실패: Board가 존재하지 않음",
			boardID: boardID,
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return nil, gorm.ErrRecordNotFound
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockBoardRepo := &MockBoardRepository{}
			mockProjectRepo := &MockProjectRepository{}
			mockFieldOptionRepo := &MockFieldOptionRepository{}
			mockConverter := &MockFieldOptionConverter{}
			tt.mockBoard(mockBoardRepo)
			
			mockParticipantRepo := &MockParticipantRepository{}
			logger, _ := zap.NewDevelopment()
			service := NewBoardService(mockBoardRepo, mockProjectRepo, mockFieldOptionRepo, mockParticipantRepo, &MockAttachmentRepository{}, nil, mockConverter, nil, logger)

			// When
			got, err := service.GetBoard(context.Background(), tt.boardID)

			// Then
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetBoard() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if appErr, ok := err.(*response.AppError); ok {
					if appErr.Code != tt.wantErrCode {
						t.Errorf("GetBoard() error code = %v, want %v", appErr.Code, tt.wantErrCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("GetBoard() unexpected error = %v", err)
					return
				}
				if got == nil {
					t.Error("GetBoard() returned nil response")
				}
			}
		})
	}
}

func TestBoardService_UpdateBoard(t *testing.T) {
	boardID := uuid.New()
	newTitle := "Updated Title"
	newCustomFields := map[string]interface{}{
		"stage":      "approved",
		"importance": "normal",
	}
	
	tests := []struct {
		name        string
		boardID     uuid.UUID
		req         *dto.UpdateBoardRequest
		mockBoard   func(*MockBoardRepository)
		wantErr     bool
		wantErrCode string
	}{
		{
			name:    "성공: Board 업데이트",
			boardID: boardID,
			req: &dto.UpdateBoardRequest{
				Title: &newTitle,
			},
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					customFieldsJSON, _ := json.Marshal(map[string]interface{}{
						"stage": "in_progress",
					})
					return &domain.Board{
						BaseModel: domain.BaseModel{
							ID:        boardID,
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						},
						Title:        "Old Title",
						CustomFields: customFieldsJSON,
					}, nil
				}
				m.UpdateFunc = func(ctx context.Context, board *domain.Board) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:    "성공: CustomFields 업데이트",
			boardID: boardID,
			req: &dto.UpdateBoardRequest{
				CustomFields: &newCustomFields,
			},
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					customFieldsJSON, _ := json.Marshal(map[string]interface{}{
						"stage": "in_progress",
					})
					return &domain.Board{
						BaseModel: domain.BaseModel{
							ID:        boardID,
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						},
						Title:        "Test Board",
						CustomFields: customFieldsJSON,
					}, nil
				}
				m.UpdateFunc = func(ctx context.Context, board *domain.Board) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:    "실패: Board가 존재하지 않음",
			boardID: boardID,
			req: &dto.UpdateBoardRequest{
				Title: &newTitle,
			},
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return nil, gorm.ErrRecordNotFound
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockBoardRepo := &MockBoardRepository{}
			mockProjectRepo := &MockProjectRepository{}
			mockFieldOptionRepo := &MockFieldOptionRepository{}
			mockConverter := &MockFieldOptionConverter{}
			tt.mockBoard(mockBoardRepo)
			
			mockParticipantRepo := &MockParticipantRepository{}
			logger, _ := zap.NewDevelopment()
			service := NewBoardService(mockBoardRepo, mockProjectRepo, mockFieldOptionRepo, mockParticipantRepo, &MockAttachmentRepository{}, nil, mockConverter, nil, logger)

			// When
			got, err := service.UpdateBoard(context.Background(), tt.boardID, tt.req)

			// Then
			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateBoard() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if appErr, ok := err.(*response.AppError); ok {
					if appErr.Code != tt.wantErrCode {
						t.Errorf("UpdateBoard() error code = %v, want %v", appErr.Code, tt.wantErrCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("UpdateBoard() unexpected error = %v", err)
					return
				}
				if got == nil {
					t.Error("UpdateBoard() returned nil response")
					return
				}
				if tt.req.Title != nil && got.Title != *tt.req.Title {
					t.Errorf("UpdateBoard() Title = %v, want %v", got.Title, *tt.req.Title)
				}
			}
		})
	}
}

func TestBoardService_UpdateBoard_CustomFields(t *testing.T) {
	boardID := uuid.New()
	
	tests := []struct {
		name             string
		existingFields   map[string]interface{}
		updateFields     map[string]interface{}
		wantFields       map[string]interface{}
	}{
		{
			name: "CustomFields 수정: 전체 교체",
			existingFields: map[string]interface{}{
				"stage":      "in_progress",
				"importance": "urgent",
			},
			updateFields: map[string]interface{}{
				"stage":      "approved",
				"importance": "normal",
				"role":       "developer",
			},
			wantFields: map[string]interface{}{
				"stage":      "approved",
				"importance": "normal",
				"role":       "developer",
			},
		},
		{
			name: "CustomFields 수정: 일부 필드만 변경",
			existingFields: map[string]interface{}{
				"stage":      "in_progress",
				"importance": "urgent",
				"role":       "planner",
			},
			updateFields: map[string]interface{}{
				"stage": "approved",
			},
			wantFields: map[string]interface{}{
				"stage": "approved",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			var updatedBoard *domain.Board
			mockBoardRepo := &MockBoardRepository{
				FindByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					customFieldsJSON, _ := json.Marshal(tt.existingFields)
					return &domain.Board{
						BaseModel: domain.BaseModel{
							ID:        boardID,
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						},
						Title:        "Test Board",
						CustomFields: customFieldsJSON,
					}, nil
				},
				UpdateFunc: func(ctx context.Context, board *domain.Board) error {
					updatedBoard = board
					return nil
				},
			}
			
			mockProjectRepo := &MockProjectRepository{}
			mockFieldOptionRepo := &MockFieldOptionRepository{}
			mockConverter := &MockFieldOptionConverter{}
			mockParticipantRepo := &MockParticipantRepository{}
			logger, _ := zap.NewDevelopment()
			service := NewBoardService(mockBoardRepo, mockProjectRepo, mockFieldOptionRepo, mockParticipantRepo, &MockAttachmentRepository{}, nil, mockConverter, nil, logger)
			
			req := &dto.UpdateBoardRequest{
				CustomFields: &tt.updateFields,
			}

			// When
			got, err := service.UpdateBoard(context.Background(), boardID, req)

			// Then
			if err != nil {
				t.Errorf("UpdateBoard() unexpected error = %v", err)
				return
			}
			
			// Verify CustomFields were updated in domain model
			if updatedBoard == nil {
				t.Fatal("Board was not updated")
			}
			
			if updatedBoard.CustomFields == nil {
				t.Error("Board.CustomFields = nil, want non-nil")
				return
			}
			
			var customFields map[string]interface{}
			if err := json.Unmarshal(updatedBoard.CustomFields, &customFields); err != nil {
				t.Errorf("Failed to unmarshal CustomFields: %v", err)
				return
			}
			
			for key, expectedValue := range tt.wantFields {
				if actualValue, ok := customFields[key]; !ok {
					t.Errorf("Board.CustomFields[%s] not found", key)
				} else if actualValue != expectedValue {
					t.Errorf("Board.CustomFields[%s] = %v, want %v", key, actualValue, expectedValue)
				}
			}
			
			// Verify CustomFields are in response
			if got.CustomFields == nil {
				t.Error("Response.CustomFields = nil, want non-nil")
			}
		})
	}
}

func TestBoardService_GetBoardsByProject_CustomFieldsFilter(t *testing.T) {
	projectID := uuid.New()
	
	tests := []struct {
		name        string
		filters     *dto.BoardFilters
		mockProject func(*MockProjectRepository)
		mockBoard   func(*MockBoardRepository)
		wantCount   int
		wantErr     bool
		wantErrCode string
	}{
		{
			name: "성공: CustomFields 필터링 없이 조회",
			filters: nil,
			mockProject: func(m *MockProjectRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return &domain.Project{}, nil
				}
			},
			mockBoard: func(m *MockBoardRepository) {
				m.FindByProjectIDFunc = func(ctx context.Context, pid uuid.UUID, filters interface{}) ([]*domain.Board, error) {
					customFields1JSON, _ := json.Marshal(map[string]interface{}{"stage": "in_progress"})
					customFields2JSON, _ := json.Marshal(map[string]interface{}{"stage": "approved"})
					return []*domain.Board{
						{
							BaseModel:    domain.BaseModel{ID: uuid.New()},
							Title:        "Board 1",
							CustomFields: customFields1JSON,
						},
						{
							BaseModel:    domain.BaseModel{ID: uuid.New()},
							Title:        "Board 2",
							CustomFields: customFields2JSON,
						},
					}, nil
				}
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "성공: stage 필터링",
			filters: &dto.BoardFilters{
				CustomFields: map[string]interface{}{
					"stage": "in_progress",
				},
			},
			mockProject: func(m *MockProjectRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return &domain.Project{}, nil
				}
			},
			mockBoard: func(m *MockBoardRepository) {
				m.FindByProjectIDFunc = func(ctx context.Context, pid uuid.UUID, filters interface{}) ([]*domain.Board, error) {
					// Simulate filtering
					if customFields, ok := filters.(map[string]interface{}); ok {
						if stage, ok := customFields["stage"]; ok && stage == "in_progress" {
							customFieldsJSON, _ := json.Marshal(map[string]interface{}{"stage": "in_progress"})
							return []*domain.Board{
								{
									BaseModel:    domain.BaseModel{ID: uuid.New()},
									Title:        "Board 1",
									CustomFields: customFieldsJSON,
								},
							}, nil
						}
					}
					return []*domain.Board{}, nil
				}
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "성공: 여러 필드로 필터링",
			filters: &dto.BoardFilters{
				CustomFields: map[string]interface{}{
					"stage":      "in_progress",
					"importance": "urgent",
				},
			},
			mockProject: func(m *MockProjectRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return &domain.Project{}, nil
				}
			},
			mockBoard: func(m *MockBoardRepository) {
				m.FindByProjectIDFunc = func(ctx context.Context, pid uuid.UUID, filters interface{}) ([]*domain.Board, error) {
					// Simulate AND filtering
					if customFields, ok := filters.(map[string]interface{}); ok {
						stage, hasStage := customFields["stage"]
						importance, hasImportance := customFields["importance"]
						if hasStage && hasImportance && stage == "in_progress" && importance == "urgent" {
							customFieldsJSON, _ := json.Marshal(map[string]interface{}{
								"stage":      "in_progress",
								"importance": "urgent",
							})
							return []*domain.Board{
								{
									BaseModel:    domain.BaseModel{ID: uuid.New()},
									Title:        "Urgent Board",
									CustomFields: customFieldsJSON,
								},
							}, nil
						}
					}
					return []*domain.Board{}, nil
				}
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "성공: 필터 조건에 맞는 보드 없음",
			filters: &dto.BoardFilters{
				CustomFields: map[string]interface{}{
					"stage": "nonexistent",
				},
			},
			mockProject: func(m *MockProjectRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return &domain.Project{}, nil
				}
			},
			mockBoard: func(m *MockBoardRepository) {
				m.FindByProjectIDFunc = func(ctx context.Context, pid uuid.UUID, filters interface{}) ([]*domain.Board, error) {
					return []*domain.Board{}, nil
				}
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:    "실패: Project가 존재하지 않음",
			filters: nil,
			mockProject: func(m *MockProjectRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return nil, gorm.ErrRecordNotFound
				}
			},
			mockBoard:   func(m *MockBoardRepository) {},
			wantErr:     true,
			wantErrCode: response.ErrCodeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockProjectRepo := &MockProjectRepository{}
			mockBoardRepo := &MockBoardRepository{}
			mockFieldOptionRepo := &MockFieldOptionRepository{}
			mockConverter := &MockFieldOptionConverter{}
			tt.mockProject(mockProjectRepo)
			tt.mockBoard(mockBoardRepo)
			
			mockParticipantRepo := &MockParticipantRepository{}
			logger, _ := zap.NewDevelopment()
			service := NewBoardService(mockBoardRepo, mockProjectRepo, mockFieldOptionRepo, mockParticipantRepo, &MockAttachmentRepository{}, nil, mockConverter, nil, logger)

			// When
			got, err := service.GetBoardsByProject(context.Background(), projectID, tt.filters)

			// Then
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetBoardsByProject() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if appErr, ok := err.(*response.AppError); ok {
					if appErr.Code != tt.wantErrCode {
						t.Errorf("GetBoardsByProject() error code = %v, want %v", appErr.Code, tt.wantErrCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("GetBoardsByProject() unexpected error = %v", err)
					return
				}
				if len(got) != tt.wantCount {
					t.Errorf("GetBoardsByProject() count = %v, want %v", len(got), tt.wantCount)
				}
				// Verify CustomFields are in response
				for _, board := range got {
					if board.CustomFields == nil && tt.filters != nil && tt.filters.CustomFields != nil {
						t.Error("Board.CustomFields = nil in response")
					}
				}
			}
		})
	}
}

func TestBoardService_DeleteBoard(t *testing.T) {
	boardID := uuid.New()
	
	tests := []struct {
		name        string
		boardID     uuid.UUID
		mockBoard   func(*MockBoardRepository)
		wantErr     bool
		wantErrCode string
	}{
		{
			name:    "성공: Board 삭제",
			boardID: boardID,
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return &domain.Board{
						BaseModel: domain.BaseModel{ID: boardID},
					}, nil
				}
				m.DeleteFunc = func(ctx context.Context, id uuid.UUID) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:    "실패: Board가 존재하지 않음",
			boardID: boardID,
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return nil, gorm.ErrRecordNotFound
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockBoardRepo := &MockBoardRepository{}
			mockProjectRepo := &MockProjectRepository{}
			mockFieldOptionRepo := &MockFieldOptionRepository{}
			mockConverter := &MockFieldOptionConverter{}
			tt.mockBoard(mockBoardRepo)
			
			mockParticipantRepo := &MockParticipantRepository{}
			logger, _ := zap.NewDevelopment()
			service := NewBoardService(mockBoardRepo, mockProjectRepo, mockFieldOptionRepo, mockParticipantRepo, &MockAttachmentRepository{}, nil, mockConverter, nil, logger)

			// When
			err := service.DeleteBoard(context.Background(), tt.boardID)

			// Then
			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteBoard() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if appErr, ok := err.(*response.AppError); ok {
					if appErr.Code != tt.wantErrCode {
						t.Errorf("DeleteBoard() error code = %v, want %v", appErr.Code, tt.wantErrCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("DeleteBoard() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestBoardService_GetBoardsByProject_WithParticipantIDs tests that participant IDs are included in board responses
func TestBoardService_GetBoardsByProject_WithParticipantIDs(t *testing.T) {
	projectID := uuid.New()
	user1ID := uuid.New()
	user2ID := uuid.New()
	user3ID := uuid.New()
	
	tests := []struct {
		name              string
		mockProject       func(*MockProjectRepository)
		mockBoard         func(*MockBoardRepository)
		wantParticipants  map[string][]uuid.UUID // board title -> participant IDs
		wantErr           bool
	}{
		{
			name: "성공: 참여자 ID 포함된 보드 목록 조회",
			mockProject: func(m *MockProjectRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return &domain.Project{}, nil
				}
			},
			mockBoard: func(m *MockBoardRepository) {
				m.FindByProjectIDFunc = func(ctx context.Context, pid uuid.UUID, filters interface{}) ([]*domain.Board, error) {
					return []*domain.Board{
						{
							BaseModel: domain.BaseModel{ID: uuid.New()},
							Title:     "Board with participants",
							Participants: []domain.Participant{
								{UserID: user1ID},
								{UserID: user2ID},
								{UserID: user3ID},
							},
						},
					}, nil
				}
			},
			wantParticipants: map[string][]uuid.UUID{
				"Board with participants": {user1ID, user2ID, user3ID},
			},
			wantErr: false,
		},
		{
			name: "성공: 참여자 없는 보드는 빈 배열 반환",
			mockProject: func(m *MockProjectRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return &domain.Project{}, nil
				}
			},
			mockBoard: func(m *MockBoardRepository) {
				m.FindByProjectIDFunc = func(ctx context.Context, pid uuid.UUID, filters interface{}) ([]*domain.Board, error) {
					return []*domain.Board{
						{
							BaseModel:    domain.BaseModel{ID: uuid.New()},
							Title:        "Board without participants",
							Participants: []domain.Participant{},
						},
					}, nil
				}
			},
			wantParticipants: map[string][]uuid.UUID{
				"Board without participants": {},
			},
			wantErr: false,
		},
		{
			name: "성공: 여러 보드, 각각 다른 참여자 수",
			mockProject: func(m *MockProjectRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return &domain.Project{}, nil
				}
			},
			mockBoard: func(m *MockBoardRepository) {
				m.FindByProjectIDFunc = func(ctx context.Context, pid uuid.UUID, filters interface{}) ([]*domain.Board, error) {
					return []*domain.Board{
						{
							BaseModel: domain.BaseModel{ID: uuid.New()},
							Title:     "Board 1",
							Participants: []domain.Participant{
								{UserID: user1ID},
							},
						},
						{
							BaseModel:    domain.BaseModel{ID: uuid.New()},
							Title:        "Board 2",
							Participants: []domain.Participant{},
						},
						{
							BaseModel: domain.BaseModel{ID: uuid.New()},
							Title:     "Board 3",
							Participants: []domain.Participant{
								{UserID: user2ID},
								{UserID: user3ID},
							},
						},
					}, nil
				}
			},
			wantParticipants: map[string][]uuid.UUID{
				"Board 1": {user1ID},
				"Board 2": {},
				"Board 3": {user2ID, user3ID},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockProjectRepo := &MockProjectRepository{}
			mockBoardRepo := &MockBoardRepository{}
			mockFieldOptionRepo := &MockFieldOptionRepository{}
			mockConverter := &MockFieldOptionConverter{}
			tt.mockProject(mockProjectRepo)
			tt.mockBoard(mockBoardRepo)
			
			mockParticipantRepo := &MockParticipantRepository{}
			logger, _ := zap.NewDevelopment()
			service := NewBoardService(mockBoardRepo, mockProjectRepo, mockFieldOptionRepo, mockParticipantRepo, &MockAttachmentRepository{}, nil, mockConverter, nil, logger)

			// When
			got, err := service.GetBoardsByProject(context.Background(), projectID, nil)

			// Then
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetBoardsByProject() error = nil, wantErr %v", tt.wantErr)
					return
				}
			} else {
				if err != nil {
					t.Errorf("GetBoardsByProject() unexpected error = %v", err)
					return
				}
				
				// Verify participant IDs are included in response
				for _, board := range got {
					expectedParticipants, ok := tt.wantParticipants[board.Title]
					if !ok {
						t.Errorf("Unexpected board title: %s", board.Title)
						continue
					}
					
					// Check ParticipantIDs field exists and is not nil
					if board.ParticipantIDs == nil {
						t.Errorf("Board %s: ParticipantIDs is nil, want non-nil slice", board.Title)
						continue
					}
					
					// Check participant count
					if len(board.ParticipantIDs) != len(expectedParticipants) {
						t.Errorf("Board %s: ParticipantIDs count = %d, want %d", 
							board.Title, len(board.ParticipantIDs), len(expectedParticipants))
						continue
					}
					
					// Check each participant ID
					for i, expectedID := range expectedParticipants {
						if board.ParticipantIDs[i] != expectedID {
							t.Errorf("Board %s: ParticipantIDs[%d] = %v, want %v", 
								board.Title, i, board.ParticipantIDs[i], expectedID)
						}
					}
				}
			}
		})
	}
}

// TestBoardService_toBoardResponse_ParticipantIDs tests the toBoardResponse method directly
func TestBoardService_toBoardResponse_ParticipantIDs(t *testing.T) {
	user1ID := uuid.New()
	user2ID := uuid.New()
	
	tests := []struct {
		name             string
		board            *domain.Board
		wantParticipants []uuid.UUID
	}{
		{
			name: "참여자 ID 추출: 여러 참여자",
			board: &domain.Board{
				BaseModel: domain.BaseModel{ID: uuid.New()},
				Title:     "Test Board",
				Participants: []domain.Participant{
					{UserID: user1ID},
					{UserID: user2ID},
				},
			},
			wantParticipants: []uuid.UUID{user1ID, user2ID},
		},
		{
			name: "참여자 ID 추출: 참여자 없음 (빈 배열)",
			board: &domain.Board{
				BaseModel:    domain.BaseModel{ID: uuid.New()},
				Title:        "Test Board",
				Participants: []domain.Participant{},
			},
			wantParticipants: []uuid.UUID{},
		},
		{
			name: "참여자 ID 추출: nil 참여자 슬라이스",
			board: &domain.Board{
				BaseModel:    domain.BaseModel{ID: uuid.New()},
				Title:        "Test Board",
				Participants: nil,
			},
			wantParticipants: []uuid.UUID{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockBoardRepo := &MockBoardRepository{}
			mockProjectRepo := &MockProjectRepository{}
			mockFieldOptionRepo := &MockFieldOptionRepository{}
			mockConverter := &MockFieldOptionConverter{}
			
			mockParticipantRepo := &MockParticipantRepository{}
			logger, _ := zap.NewDevelopment()
			service := NewBoardService(mockBoardRepo, mockProjectRepo, mockFieldOptionRepo, mockParticipantRepo, &MockAttachmentRepository{}, nil, mockConverter, nil, logger).(*boardServiceImpl)

			// When
			response := service.toBoardResponse(tt.board)

			// Then
			if response.ParticipantIDs == nil {
				t.Error("ParticipantIDs is nil, want non-nil slice")
				return
			}
			
			if len(response.ParticipantIDs) != len(tt.wantParticipants) {
				t.Errorf("ParticipantIDs count = %d, want %d", 
					len(response.ParticipantIDs), len(tt.wantParticipants))
				return
			}
			
			for i, expectedID := range tt.wantParticipants {
				if response.ParticipantIDs[i] != expectedID {
					t.Errorf("ParticipantIDs[%d] = %v, want %v", 
						i, response.ParticipantIDs[i], expectedID)
				}
			}
		})
	}
}

// TestBoardService_toBoardResponse_Attachments tests attachment conversion in toBoardResponse
func TestBoardService_toBoardResponse_Attachments(t *testing.T) {
	mockBoardRepo := &MockBoardRepository{}
	mockProjectRepo := &MockProjectRepository{}
	mockFieldOptionRepo := &MockFieldOptionRepository{}
	mockConverter := &MockFieldOptionConverter{}

	mockParticipantRepo := &MockParticipantRepository{}
			logger, _ := zap.NewDevelopment()
			service := NewBoardService(mockBoardRepo, mockProjectRepo, mockFieldOptionRepo, mockParticipantRepo, &MockAttachmentRepository{}, nil, mockConverter, nil, logger)
	boardService := service.(*boardServiceImpl)

	tests := []struct {
		name        string
		board       *domain.Board
		wantAttachments int
	}{
		{
			name: "첨부파일 변환: 여러 첨부파일",
			board: &domain.Board{
				BaseModel: domain.BaseModel{
					ID:        uuid.New(),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				ProjectID:    uuid.New(),
				AuthorID:     uuid.New(),
				Title:        "Test Board",
				Content:      "Test Content",
				CustomFields: []byte(`{}`),
				Participants: []domain.Participant{},
				Attachments: []domain.Attachment{
					{
						BaseModel: domain.BaseModel{
							ID:        uuid.New(),
							CreatedAt: time.Now(),
						},
						EntityType:  domain.EntityTypeBoard,
						EntityID:    func() *uuid.UUID { id := uuid.New(); return &id }(),
						FileName:    "document1.pdf",
						FileURL:     "https://s3.amazonaws.com/bucket/file1.pdf",
						FileSize:    1024000,
						ContentType: "application/pdf",
						UploadedBy:  uuid.New(),
					},
					{
						BaseModel: domain.BaseModel{
							ID:        uuid.New(),
							CreatedAt: time.Now(),
						},
						EntityType:  domain.EntityTypeBoard,
						EntityID:    func() *uuid.UUID { id := uuid.New(); return &id }(),
						FileName:    "image.png",
						FileURL:     "https://s3.amazonaws.com/bucket/image.png",
						FileSize:    512000,
						ContentType: "image/png",
						UploadedBy:  uuid.New(),
					},
				},
			},
			wantAttachments: 2,
		},
		{
			name: "첨부파일 변환: 첨부파일 없음 (빈 배열)",
			board: &domain.Board{
				BaseModel: domain.BaseModel{
					ID:        uuid.New(),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				ProjectID:    uuid.New(),
				AuthorID:     uuid.New(),
				Title:        "Test Board",
				Content:      "Test Content",
				CustomFields: []byte(`{}`),
				Participants: []domain.Participant{},
				Attachments:  []domain.Attachment{},
			},
			wantAttachments: 0,
		},
		{
			name: "첨부파일 변환: nil 첨부파일 슬라이스",
			board: &domain.Board{
				BaseModel: domain.BaseModel{
					ID:        uuid.New(),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				ProjectID:    uuid.New(),
				AuthorID:     uuid.New(),
				Title:        "Test Board",
				Content:      "Test Content",
				CustomFields: []byte(`{}`),
				Participants: []domain.Participant{},
				Attachments:  nil,
			},
			wantAttachments: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := boardService.toBoardResponse(tt.board)

			if response == nil {
				t.Fatal("toBoardResponse() returned nil")
			}

			if response.Attachments == nil {
				t.Error("Attachments field should not be nil, expected empty array")
			}

			if len(response.Attachments) != tt.wantAttachments {
				t.Errorf("Attachments count = %d, want %d", len(response.Attachments), tt.wantAttachments)
			}

			// Verify attachment details if present
			if tt.wantAttachments > 0 {
				for i, attachment := range response.Attachments {
					if attachment.ID == uuid.Nil {
						t.Errorf("Attachment[%d].ID is nil", i)
					}
					if attachment.FileName == "" {
						t.Errorf("Attachment[%d].FileName is empty", i)
					}
					if attachment.FileURL == "" {
						t.Errorf("Attachment[%d].FileURL is empty", i)
					}
					if attachment.FileSize == 0 {
						t.Errorf("Attachment[%d].FileSize is 0", i)
					}
					if attachment.ContentType == "" {
						t.Errorf("Attachment[%d].ContentType is empty", i)
					}
					if attachment.UploadedBy == uuid.Nil {
						t.Errorf("Attachment[%d].UploadedBy is nil", i)
					}
				}
			}
		})
	}
}

// TestCreateBoard_DateValidation tests date validation when creating a board
func TestCreateBoard_DateValidation(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()
	
	// Create test dates
	startDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	dueDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) // Before start date
	
	mockProjectRepo := &MockProjectRepository{
		FindByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
			return &domain.Project{
				BaseModel: domain.BaseModel{ID: projectID},
			}, nil
		},
	}
	
	mockBoardRepo := &MockBoardRepository{}
	mockFieldOptionRepo := &MockFieldOptionRepository{}
	mockConverter := &MockFieldOptionConverter{}
	
	mockParticipantRepo := &MockParticipantRepository{}
			logger, _ := zap.NewDevelopment()
			service := NewBoardService(mockBoardRepo, mockProjectRepo, mockFieldOptionRepo, mockParticipantRepo, &MockAttachmentRepository{}, nil, mockConverter, nil, logger)
	
	ctx := context.WithValue(context.Background(), "user_id", userID)
	
	req := &dto.CreateBoardRequest{
		ProjectID: projectID,
		Title:     "Test Board",
		Content:   "Test Content",
		StartDate: &startDate,
		DueDate:   &dueDate,
	}
	
	// Should return validation error
	_, err := service.CreateBoard(ctx, req)
	
	if err == nil {
		t.Fatal("Expected validation error, got nil")
	}
	
	appErr, ok := err.(*response.AppError)
	if !ok {
		t.Fatalf("Expected AppError, got %T", err)
	}
	
	if appErr.Code != response.ErrCodeValidation {
		t.Errorf("Expected error code %s, got %s", response.ErrCodeValidation, appErr.Code)
	}
	
	if appErr.Message != "Start date cannot be after due date" {
		t.Errorf("Expected error message 'Start date cannot be after due date', got '%s'", appErr.Message)
	}
}

// TestCreateBoard_ValidDateRange tests creating a board with valid date range
func TestCreateBoard_ValidDateRange(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()
	boardID := uuid.New()
	
	// Create test dates - valid range
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	dueDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	
	mockProjectRepo := &MockProjectRepository{
		FindByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
			return &domain.Project{
				BaseModel: domain.BaseModel{ID: projectID},
			}, nil
		},
	}
	
	mockBoardRepo := &MockBoardRepository{
		CreateFunc: func(ctx context.Context, board *domain.Board) error {
			board.ID = boardID
			return nil
		},
	}
	
	mockFieldOptionRepo := &MockFieldOptionRepository{}
	mockConverter := &MockFieldOptionConverter{
		ConvertValuesToIDsFunc: func(ctx context.Context, projectID uuid.UUID, customFields map[string]interface{}) (map[string]interface{}, error) {
			return customFields, nil
		},
	}
	
	mockParticipantRepo := &MockParticipantRepository{}
			logger, _ := zap.NewDevelopment()
			service := NewBoardService(mockBoardRepo, mockProjectRepo, mockFieldOptionRepo, mockParticipantRepo, &MockAttachmentRepository{}, nil, mockConverter, nil, logger)
	
	ctx := context.WithValue(context.Background(), "user_id", userID)
	
	req := &dto.CreateBoardRequest{
		ProjectID: projectID,
		Title:     "Test Board",
		Content:   "Test Content",
		StartDate: &startDate,
		DueDate:   &dueDate,
	}
	
	// Should succeed
	result, err := service.CreateBoard(ctx, req)
	
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	
	if result.StartDate == nil || !result.StartDate.Equal(startDate) {
		t.Errorf("Expected start date %v, got %v", startDate, result.StartDate)
	}
	
	if result.DueDate == nil || !result.DueDate.Equal(dueDate) {
		t.Errorf("Expected due date %v, got %v", dueDate, result.DueDate)
	}
}

// TestUpdateBoard_DateValidation tests date validation when updating a board
func TestUpdateBoard_DateValidation(t *testing.T) {
	boardID := uuid.New()
	projectID := uuid.New()
	
	// Create test dates
	existingStartDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	newDueDate := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC) // Before existing start date
	
	mockBoardRepo := &MockBoardRepository{
		FindByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
			return &domain.Board{
				BaseModel: domain.BaseModel{ID: boardID},
				ProjectID: projectID,
				Title:     "Test Board",
				StartDate: &existingStartDate,
			}, nil
		},
	}
	
	mockProjectRepo := &MockProjectRepository{}
	mockFieldOptionRepo := &MockFieldOptionRepository{}
	mockConverter := &MockFieldOptionConverter{}
	
	mockParticipantRepo := &MockParticipantRepository{}
			logger, _ := zap.NewDevelopment()
			service := NewBoardService(mockBoardRepo, mockProjectRepo, mockFieldOptionRepo, mockParticipantRepo, &MockAttachmentRepository{}, nil, mockConverter, nil, logger)
	
	ctx := context.Background()
	
	req := &dto.UpdateBoardRequest{
		DueDate: &newDueDate,
	}
	
	// Should return validation error
	_, err := service.UpdateBoard(ctx, boardID, req)
	
	if err == nil {
		t.Fatal("Expected validation error, got nil")
	}
	
	appErr, ok := err.(*response.AppError)
	if !ok {
		t.Fatalf("Expected AppError, got %T", err)
	}
	
	if appErr.Code != response.ErrCodeValidation {
		t.Errorf("Expected error code %s, got %s", response.ErrCodeValidation, appErr.Code)
	}
}

// TestUpdateBoard_ValidDateUpdate tests updating a board with valid dates
func TestUpdateBoard_ValidDateUpdate(t *testing.T) {
	boardID := uuid.New()
	projectID := uuid.New()
	
	// Create test dates - valid range
	existingStartDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	newDueDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	
	mockBoardRepo := &MockBoardRepository{
		FindByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
			return &domain.Board{
				BaseModel: domain.BaseModel{ID: boardID},
				ProjectID: projectID,
				Title:     "Test Board",
				StartDate: &existingStartDate,
			}, nil
		},
		UpdateFunc: func(ctx context.Context, board *domain.Board) error {
			return nil
		},
	}
	
	mockProjectRepo := &MockProjectRepository{}
	mockFieldOptionRepo := &MockFieldOptionRepository{}
	mockConverter := &MockFieldOptionConverter{}
	
	mockParticipantRepo := &MockParticipantRepository{}
			logger, _ := zap.NewDevelopment()
			service := NewBoardService(mockBoardRepo, mockProjectRepo, mockFieldOptionRepo, mockParticipantRepo, &MockAttachmentRepository{}, nil, mockConverter, nil, logger)
	
	ctx := context.Background()
	
	req := &dto.UpdateBoardRequest{
		DueDate: &newDueDate,
	}
	
	// Should succeed
	result, err := service.UpdateBoard(ctx, boardID, req)
	
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
}
