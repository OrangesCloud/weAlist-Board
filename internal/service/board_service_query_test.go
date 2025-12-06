package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
	"project-board-api/internal/response"
)

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
