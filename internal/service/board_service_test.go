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

// Mock implementations are in mock_test.go

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

