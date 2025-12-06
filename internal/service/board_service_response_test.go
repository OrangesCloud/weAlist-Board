package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
	"project-board-api/internal/response"
)

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
			service := NewBoardService(mockBoardRepo, mockProjectRepo, mockFieldOptionRepo, mockParticipantRepo, &MockAttachmentRepository{}, &MockS3Client{}, mockConverter, nil, logger)
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
