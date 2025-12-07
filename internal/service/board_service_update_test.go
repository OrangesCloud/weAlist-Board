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
				var updatedBoard *domain.Board
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					if updatedBoard != nil {
						return updatedBoard, nil
					}
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
					updatedBoard = board
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
				var updatedBoard *domain.Board
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					if updatedBoard != nil {
						return updatedBoard, nil
					}
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
					updatedBoard = board
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
		name           string
		existingFields map[string]interface{}
		updateFields   map[string]interface{}
		wantFields     map[string]interface{}
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
