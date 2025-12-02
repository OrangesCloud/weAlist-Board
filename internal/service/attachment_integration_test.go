package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
	"project-board-api/internal/response"
)

// TestCreateBoardWithAttachments tests creating a board with attachments
func TestCreateBoardWithAttachments(t *testing.T) {
	ctx := context.WithValue(context.Background(), "user_id", uuid.New())
	logger := zap.NewNop()

	t.Run("Success - Create board with valid temp attachments", func(t *testing.T) {
		projectID := uuid.New()
		attachmentID1 := uuid.New()
		attachmentID2 := uuid.New()

		mockBoardRepo := &MockBoardRepository{
			CreateFunc: func(ctx context.Context, board *domain.Board) error {
				board.ID = uuid.New()
				return nil
			},
		}

		mockProjectRepo := &MockProjectRepository{
			FindByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
				return &domain.Project{BaseModel: domain.BaseModel{ID: projectID}}, nil
			},
		}

		mockAttachmentRepo := &MockAttachmentRepository{
			FindByIDsFunc: func(ctx context.Context, ids []uuid.UUID) ([]*domain.Attachment, error) {
				return []*domain.Attachment{
					{
						BaseModel:  domain.BaseModel{ID: attachmentID1},
						EntityType: domain.EntityTypeBoard,
						Status:     domain.AttachmentStatusTemp,
					},
					{
						BaseModel:  domain.BaseModel{ID: attachmentID2},
						EntityType: domain.EntityTypeBoard,
						Status:     domain.AttachmentStatusTemp,
					},
				}, nil
			},
			ConfirmAttachmentsFunc: func(ctx context.Context, attachmentIDs []uuid.UUID, entityID uuid.UUID) error {
				if len(attachmentIDs) != 2 {
					t.Errorf("Expected 2 attachments to be confirmed, got %d", len(attachmentIDs))
				}
				return nil
			},
		}

		mockFieldOptionConverter := &MockFieldOptionConverter{}
		mockParticipantRepo := &MockParticipantRepository{}
		mockFieldOptionRepo := &MockFieldOptionRepository{}

		service := NewBoardService(
			mockBoardRepo,
			mockProjectRepo,
			mockFieldOptionRepo,
			mockParticipantRepo,
			mockAttachmentRepo,
			nil, // s3Client
			mockFieldOptionConverter,
			nil, // metrics
			logger,
		)

		req := &dto.CreateBoardRequest{
			ProjectID:     projectID,
			Title:         "Test Board",
			Content:       "Test Content",
			AttachmentIDs: []uuid.UUID{attachmentID1, attachmentID2},
		}

		result, err := service.CreateBoard(ctx, req)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}
	})

	t.Run("Error - Attachment not found", func(t *testing.T) {
		projectID := uuid.New()
		attachmentID := uuid.New()

		mockBoardRepo := &MockBoardRepository{}
		mockProjectRepo := &MockProjectRepository{
			FindByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
				return &domain.Project{BaseModel: domain.BaseModel{ID: projectID}}, nil
			},
		}

		mockAttachmentRepo := &MockAttachmentRepository{
			FindByIDsFunc: func(ctx context.Context, ids []uuid.UUID) ([]*domain.Attachment, error) {
				// Return empty slice - attachment not found
				return []*domain.Attachment{}, nil
			},
		}

		mockFieldOptionConverter := &MockFieldOptionConverter{}
		mockParticipantRepo := &MockParticipantRepository{}
		mockFieldOptionRepo := &MockFieldOptionRepository{}

		service := NewBoardService(
			mockBoardRepo,
			mockProjectRepo,
			mockFieldOptionRepo,
			mockParticipantRepo,
			mockAttachmentRepo,
			nil, // s3Client
			mockFieldOptionConverter,
			nil, // metrics
			logger,
		)

		req := &dto.CreateBoardRequest{
			ProjectID:     projectID,
			Title:         "Test Board",
			Content:       "Test Content",
			AttachmentIDs: []uuid.UUID{attachmentID},
		}

		_, err := service.CreateBoard(ctx, req)
		if err == nil {
			t.Fatal("Expected error for missing attachment, got nil")
		}

		appErr, ok := err.(*response.AppError)
		if !ok {
			t.Fatalf("Expected AppError, got %T", err)
		}

		if appErr.Code != response.ErrCodeValidation {
			t.Errorf("Expected validation error, got %s", appErr.Code)
		}
	})

	t.Run("Error - Attachment already confirmed", func(t *testing.T) {
		projectID := uuid.New()
		attachmentID := uuid.New()

		mockBoardRepo := &MockBoardRepository{}
		mockProjectRepo := &MockProjectRepository{
			FindByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
				return &domain.Project{BaseModel: domain.BaseModel{ID: projectID}}, nil
			},
		}

		mockAttachmentRepo := &MockAttachmentRepository{
			FindByIDsFunc: func(ctx context.Context, ids []uuid.UUID) ([]*domain.Attachment, error) {
				return []*domain.Attachment{
					{
						BaseModel:  domain.BaseModel{ID: attachmentID},
						EntityType: domain.EntityTypeBoard,
						Status:     domain.AttachmentStatusConfirmed, // Already confirmed
					},
				}, nil
			},
		}

		mockFieldOptionConverter := &MockFieldOptionConverter{}
		mockParticipantRepo := &MockParticipantRepository{}
		mockFieldOptionRepo := &MockFieldOptionRepository{}

		service := NewBoardService(
			mockBoardRepo,
			mockProjectRepo,
			mockFieldOptionRepo,
			mockParticipantRepo,
			mockAttachmentRepo,
			nil, // s3Client
			mockFieldOptionConverter,
			nil, // metrics
			logger,
		)

		req := &dto.CreateBoardRequest{
			ProjectID:     projectID,
			Title:         "Test Board",
			Content:       "Test Content",
			AttachmentIDs: []uuid.UUID{attachmentID},
		}

		_, err := service.CreateBoard(ctx, req)
		if err == nil {
			t.Fatal("Expected error for confirmed attachment, got nil")
		}

		appErr, ok := err.(*response.AppError)
		if !ok {
			t.Fatalf("Expected AppError, got %T", err)
		}

		if appErr.Code != response.ErrCodeValidation {
			t.Errorf("Expected validation error, got %s", appErr.Code)
		}
	})

	t.Run("Error - Attachment entity type mismatch", func(t *testing.T) {
		projectID := uuid.New()
		attachmentID := uuid.New()

		mockBoardRepo := &MockBoardRepository{}
		mockProjectRepo := &MockProjectRepository{
			FindByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
				return &domain.Project{BaseModel: domain.BaseModel{ID: projectID}}, nil
			},
		}

		mockAttachmentRepo := &MockAttachmentRepository{
			FindByIDsFunc: func(ctx context.Context, ids []uuid.UUID) ([]*domain.Attachment, error) {
				return []*domain.Attachment{
					{
						BaseModel:  domain.BaseModel{ID: attachmentID},
						EntityType: domain.EntityTypeComment, // Wrong entity type
						Status:     domain.AttachmentStatusTemp,
					},
				}, nil
			},
		}

		mockFieldOptionConverter := &MockFieldOptionConverter{}
		mockParticipantRepo := &MockParticipantRepository{}
		mockFieldOptionRepo := &MockFieldOptionRepository{}

		service := NewBoardService(
			mockBoardRepo,
			mockProjectRepo,
			mockFieldOptionRepo,
			mockParticipantRepo,
			mockAttachmentRepo,
			nil, // s3Client
			mockFieldOptionConverter,
			nil, // metrics
			logger,
		)

		req := &dto.CreateBoardRequest{
			ProjectID:     projectID,
			Title:         "Test Board",
			Content:       "Test Content",
			AttachmentIDs: []uuid.UUID{attachmentID},
		}

		_, err := service.CreateBoard(ctx, req)
		if err == nil {
			t.Fatal("Expected error for entity type mismatch, got nil")
		}

		appErr, ok := err.(*response.AppError)
		if !ok {
			t.Fatalf("Expected AppError, got %T", err)
		}

		if appErr.Code != response.ErrCodeValidation {
			t.Errorf("Expected validation error, got %s", appErr.Code)
		}
	})
}

// TestCreateCommentWithAttachments tests creating a comment with attachments
func TestCreateCommentWithAttachments(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	t.Run("Success - Create comment with valid temp attachments", func(t *testing.T) {
		boardID := uuid.New()
		attachmentID := uuid.New()

		mockCommentRepo := &MockCommentRepository{
			CreateFunc: func(ctx context.Context, comment *domain.Comment) error {
				comment.ID = uuid.New()
				return nil
			},
		}

		mockBoardRepo := &MockBoardRepository{
			FindByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
				return &domain.Board{BaseModel: domain.BaseModel{ID: boardID}}, nil
			},
		}

		mockAttachmentRepo := &MockAttachmentRepository{
			FindByIDsFunc: func(ctx context.Context, ids []uuid.UUID) ([]*domain.Attachment, error) {
				return []*domain.Attachment{
					{
						BaseModel:  domain.BaseModel{ID: attachmentID},
						EntityType: domain.EntityTypeComment,
						Status:     domain.AttachmentStatusTemp,
					},
				}, nil
			},
			ConfirmAttachmentsFunc: func(ctx context.Context, attachmentIDs []uuid.UUID, entityID uuid.UUID) error {
				if len(attachmentIDs) != 1 {
					t.Errorf("Expected 1 attachment to be confirmed, got %d", len(attachmentIDs))
				}
				return nil
			},
		}

		service := NewCommentService(mockCommentRepo, mockBoardRepo, mockAttachmentRepo, nil, logger)

		req := &dto.CreateCommentRequest{
			BoardID:       boardID,
			Content:       "Test Comment",
			AttachmentIDs: []uuid.UUID{attachmentID},
		}

		userID := uuid.New()
		result, err := service.CreateComment(ctx, userID, req)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}
	})

	t.Run("Error - Attachment not found", func(t *testing.T) {
		boardID := uuid.New()
		attachmentID := uuid.New()

		mockCommentRepo := &MockCommentRepository{}
		mockBoardRepo := &MockBoardRepository{
			FindByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
				return &domain.Board{BaseModel: domain.BaseModel{ID: boardID}}, nil
			},
		}

		mockAttachmentRepo := &MockAttachmentRepository{
			FindByIDsFunc: func(ctx context.Context, ids []uuid.UUID) ([]*domain.Attachment, error) {
				return []*domain.Attachment{}, nil
			},
		}

		service := NewCommentService(mockCommentRepo, mockBoardRepo, mockAttachmentRepo, nil, logger)

		req := &dto.CreateCommentRequest{
			BoardID:       boardID,
			Content:       "Test Comment",
			AttachmentIDs: []uuid.UUID{attachmentID},
		}

		userID := uuid.New()
		_, err := service.CreateComment(ctx, userID, req)
		if err == nil {
			t.Fatal("Expected error for missing attachment, got nil")
		}
	})
}

// TestCreateProjectWithAttachments tests creating a project with attachments
func TestCreateProjectWithAttachments(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	t.Run("Success - Create project with valid temp attachments", func(t *testing.T) {
		workspaceID := uuid.New()
		userID := uuid.New()
		attachmentID := uuid.New()

		mockProjectRepo := &MockProjectRepository{
			CreateFunc: func(ctx context.Context, project *domain.Project) error {
				project.ID = uuid.New()
				return nil
			},
			AddMemberFunc: func(ctx context.Context, member *domain.ProjectMember) error {
				return nil
			},
		}

		mockFieldOptionRepo := &MockFieldOptionRepository{
			CreateBatchFunc: func(ctx context.Context, fieldOptions []*domain.FieldOption) error {
				return nil
			},
		}

		mockAttachmentRepo := &MockAttachmentRepository{
			FindByIDsFunc: func(ctx context.Context, ids []uuid.UUID) ([]*domain.Attachment, error) {
				return []*domain.Attachment{
					{
						BaseModel:  domain.BaseModel{ID: attachmentID},
						EntityType: domain.EntityTypeProject,
						Status:     domain.AttachmentStatusTemp,
					},
				}, nil
			},
			ConfirmAttachmentsFunc: func(ctx context.Context, attachmentIDs []uuid.UUID, entityID uuid.UUID) error {
				if len(attachmentIDs) != 1 {
					t.Errorf("Expected 1 attachment to be confirmed, got %d", len(attachmentIDs))
				}
				return nil
			},
		}

		mockUserClient := &MockUserClient{
			ValidateWorkspaceMemberFunc: func(ctx context.Context, workspaceID, userID uuid.UUID, token string) (bool, error) {
				return true, nil
			},
		}

		service := NewProjectService(mockProjectRepo, mockFieldOptionRepo, mockAttachmentRepo, nil, mockUserClient, nil, logger)

		req := &dto.CreateProjectRequest{
			WorkspaceID:   workspaceID,
			Name:          "Test Project",
			Description:   "Test Description",
			AttachmentIDs: []uuid.UUID{attachmentID},
		}

		result, err := service.CreateProject(ctx, req, userID, "test-token")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}
	})

	t.Run("Error - Attachment not found", func(t *testing.T) {
		workspaceID := uuid.New()
		userID := uuid.New()
		attachmentID := uuid.New()

		mockProjectRepo := &MockProjectRepository{}
		mockFieldOptionRepo := &MockFieldOptionRepository{}

		mockAttachmentRepo := &MockAttachmentRepository{
			FindByIDsFunc: func(ctx context.Context, ids []uuid.UUID) ([]*domain.Attachment, error) {
				return []*domain.Attachment{}, nil
			},
		}

		mockUserClient := &MockUserClient{
			ValidateWorkspaceMemberFunc: func(ctx context.Context, workspaceID, userID uuid.UUID, token string) (bool, error) {
				return true, nil
			},
		}

		service := NewProjectService(mockProjectRepo, mockFieldOptionRepo, mockAttachmentRepo, nil, mockUserClient, nil, logger)

		req := &dto.CreateProjectRequest{
			WorkspaceID:   workspaceID,
			Name:          "Test Project",
			Description:   "Test Description",
			AttachmentIDs: []uuid.UUID{attachmentID},
		}

		_, err := service.CreateProject(ctx, req, userID, "test-token")
		if err == nil {
			t.Fatal("Expected error for missing attachment, got nil")
		}
	})
}
