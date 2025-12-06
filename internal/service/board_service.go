package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
	"project-board-api/internal/metrics"
	"project-board-api/internal/repository"
	"project-board-api/internal/response"
)

// BoardService defines the interface for board business logic
type BoardService interface {
	CreateBoard(ctx context.Context, req *dto.CreateBoardRequest) (*dto.BoardResponse, error)
	GetBoard(ctx context.Context, boardID uuid.UUID) (*dto.BoardDetailResponse, error)
	GetBoardsByProject(ctx context.Context, projectID uuid.UUID, filters *dto.BoardFilters) ([]*dto.BoardResponse, error)
	UpdateBoard(ctx context.Context, boardID uuid.UUID, req *dto.UpdateBoardRequest) (*dto.BoardResponse, error)
	DeleteBoard(ctx context.Context, boardID uuid.UUID) error
}

// boardServiceImpl is the implementation of BoardService
type boardServiceImpl struct {
	boardRepo            repository.BoardRepository
	projectRepo          repository.ProjectRepository
	fieldOptionRepo      repository.FieldOptionRepository
	participantRepo      repository.ParticipantRepository
	attachmentRepo       repository.AttachmentRepository
	s3Client             S3Client
	fieldOptionConverter FieldOptionConverter
	metrics              *metrics.Metrics
	logger               *zap.Logger
}

// FieldOptionConverter handles conversion between field option values and IDs
type FieldOptionConverter interface {
	ConvertValuesToIDs(ctx context.Context, projectID uuid.UUID, customFields map[string]interface{}) (map[string]interface{}, error)
	ConvertIDsToValues(ctx context.Context, customFields map[string]interface{}) (map[string]interface{}, error)
	ConvertIDsToValuesBatch(ctx context.Context, boards []*domain.Board) error
}

// NewBoardService creates a new instance of BoardService
func NewBoardService(
	boardRepo repository.BoardRepository,
	projectRepo repository.ProjectRepository,
	fieldOptionRepo repository.FieldOptionRepository,
	participantRepo repository.ParticipantRepository,
	attachmentRepo repository.AttachmentRepository,
	s3Client S3Client,
	fieldOptionConverter FieldOptionConverter,
	m *metrics.Metrics,
	logger *zap.Logger,
) BoardService {
	return &boardServiceImpl{
		boardRepo:            boardRepo,
		projectRepo:          projectRepo,
		fieldOptionRepo:      fieldOptionRepo,
		participantRepo:      participantRepo,
		attachmentRepo:       attachmentRepo,
		s3Client:             s3Client,
		fieldOptionConverter: fieldOptionConverter,
		metrics:              m,
		logger:               logger,
	}
}

// CreateBoard creates a new board
func (s *boardServiceImpl) CreateBoard(ctx context.Context, req *dto.CreateBoardRequest) (*dto.BoardResponse, error) {
	// Extract user_id from context (set by auth middleware as uuid.UUID)
	authorID, exists := ctx.Value("user_id").(uuid.UUID)
	if !exists {
		return nil, response.NewAppError(response.ErrCodeUnauthorized, "User ID not found in context", "")
	}

	// Validate date range
	if err := validateDateRange(req.StartDate, req.DueDate); err != nil {
		return nil, err
	}

	// Verify project exists
	_, err := s.projectRepo.FindByID(ctx, req.ProjectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewAppError(response.ErrCodeNotFound, "Project not found", "")
		}
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to verify project", err.Error())
	}

	// Convert CustomFields from values to IDs, then to datatypes.JSON
	var customFieldsJSON datatypes.JSON
	if req.CustomFields != nil {
		// Convert values to IDs
		convertedFields, err := s.fieldOptionConverter.ConvertValuesToIDs(ctx, req.ProjectID, req.CustomFields)
		if err != nil {
			return nil, response.NewAppError(response.ErrCodeValidation, "Invalid custom field values", err.Error())
		}

		jsonBytes, err := json.Marshal(convertedFields)
		if err != nil {
			return nil, response.NewAppError(response.ErrCodeInternal, "Failed to marshal custom fields", err.Error())
		}
		customFieldsJSON = jsonBytes
	}

	// Set assigneeID: use provided value, or default to authorID if not provided
	assigneeID := req.AssigneeID
	if assigneeID == nil {
		assigneeID = &authorID
	}

	// Validate and confirm attachments if provided
	if len(req.AttachmentIDs) > 0 {
		if err := s.validateAndConfirmAttachments(ctx, req.AttachmentIDs, domain.EntityTypeBoard, uuid.Nil); err != nil {
			return nil, err
		}
	}

	// Create domain model from request with AuthorID
	board := &domain.Board{
		ProjectID:    req.ProjectID,
		AuthorID:     authorID,
		Title:        req.Title,
		Content:      req.Content,
		CustomFields: customFieldsJSON,
		AssigneeID:   assigneeID,
		StartDate:    req.StartDate,
		DueDate:      req.DueDate,
	}

	// Save to repository
	if err := s.boardRepo.Create(ctx, board); err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to create board", err.Error())
	}

	// Confirm attachments after board creation
	var createdAttachments []*domain.Attachment
	if len(req.AttachmentIDs) > 0 {
		// 에러 발생 시 board도 롤백
		if err := s.attachmentRepo.ConfirmAttachments(ctx, req.AttachmentIDs, board.ID); err != nil {
			s.logger.Error("Failed to confirm attachments, rolling back board creation",
				zap.String("board_id", board.ID.String()),
				zap.Strings("attachment_ids", func() []string {
					ids := make([]string, len(req.AttachmentIDs))
					for i, id := range req.AttachmentIDs {
						ids[i] = id.String()
					}
					return ids
				}()),
				zap.Error(err))

			// board 삭제 (롤백)
			if deleteErr := s.boardRepo.Delete(ctx, board.ID); deleteErr != nil {
				s.logger.Error("Failed to rollback board after attachment confirmation failure",
					zap.String("board_id", board.ID.String()),
					zap.Error(deleteErr))
			}

			return nil, response.NewAppError(response.ErrCodeInternal,
				"Failed to confirm attachments: "+err.Error(),
				"Please ensure all attachment IDs are valid and not already used")
		}

		// Confirm 후 Attachments 메타데이터를 조회하여 board 객체에 할당
		attachments, err := s.attachmentRepo.FindByIDs(ctx, req.AttachmentIDs)
		if err != nil {
			s.logger.Warn("Failed to fetch confirmed attachments for response", zap.Error(err))
		} else {
			createdAttachments = attachments
		}
	}

	// Increment board creation metric
	if s.metrics != nil {
		s.metrics.IncrementBoardCreated()
	}

	// Add participants if provided
	if len(req.Participants) > 0 {
		successCount, err := s.addParticipantsInternal(ctx, board.ID, req.Participants)
		if err != nil {
			s.logger.Warn("Error occurred while adding participants during board creation",
				zap.String("board_id", board.ID.String()),
				zap.Int("success_count", successCount),
				zap.Error(err))
		}

		// Reload board with participants to include them in response
		reloadedBoard, err := s.boardRepo.FindByID(ctx, board.ID)
		if err != nil {
			s.logger.Warn("Failed to reload board with participants",
				zap.String("board_id", board.ID.String()),
				zap.Error(err))
			// Continue with original board if reload fails
		} else {
			board = reloadedBoard
		}
	}

	// 생성된 Attachments를 Board 객체에 할당 (타입 변환 적용)
	board.Attachments = toDomainAttachments(createdAttachments)

	// Convert to response DTO
	return s.toBoardResponse(board), nil
}

// GetBoard retrieves a board by ID with participants and comments
func (s *boardServiceImpl) GetBoard(ctx context.Context, boardID uuid.UUID) (*dto.BoardDetailResponse, error) {
	// Fetch board from repository
	board, err := s.boardRepo.FindByID(ctx, boardID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewAppError(response.ErrCodeNotFound, "Board not found", "")
		}
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch board", err.Error())
	}

	// Attachments 로드 (타입 변환 적용)
	attachments, err := s.attachmentRepo.FindByEntityID(ctx, domain.EntityTypeBoard, board.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Error("Failed to fetch attachments for board", zap.String("board_id", board.ID.String()), zap.Error(err))
		// Continue with graceful degradation
	}
	board.Attachments = toDomainAttachments(attachments)

	// Convert IDs to values in customFields
	if err := s.convertBoardCustomFieldsToValues(ctx, board); err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to convert custom fields", err.Error())
	}

	// Convert to detailed response DTO
	return s.toBoardDetailResponse(board), nil
}

// GetBoardsByProject retrieves all boards for a project with optional filters
func (s *boardServiceImpl) GetBoardsByProject(ctx context.Context, projectID uuid.UUID, filters *dto.BoardFilters) ([]*dto.BoardResponse, error) {
	// Verify project exists
	_, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewAppError(response.ErrCodeNotFound, "Project not found", "")
		}
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to verify project", err.Error())
	}

	// Prepare filter parameter for repository
	var filterParam interface{}
	if filters != nil && filters.CustomFields != nil {
		filterParam = filters.CustomFields
	}

	// Fetch boards from repository with filters
	boards, err := s.boardRepo.FindByProjectID(ctx, projectID, filterParam)
	if err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch boards", err.Error())
	}

	// Board 목록 조회 시 Attachments 로드 (효율을 위해 각 board별로 로드)
	for _, board := range boards {
		attachments, err := s.attachmentRepo.FindByEntityID(ctx, domain.EntityTypeBoard, board.ID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Error("Failed to fetch attachments for board list", zap.String("board_id", board.ID.String()), zap.Error(err))
		}
		board.Attachments = toDomainAttachments(attachments)
	}

	// Convert IDs to values in batch for all boards
	if err := s.fieldOptionConverter.ConvertIDsToValuesBatch(ctx, boards); err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to convert custom fields", err.Error())
	}

	// Convert to response DTOs
	responses := make([]*dto.BoardResponse, len(boards))
	for i, board := range boards {
		responses[i] = s.toBoardResponse(board)
	}

	return responses, nil
}

// UpdateBoard updates a board's attributes
func (s *boardServiceImpl) DeleteBoard(ctx context.Context, boardID uuid.UUID) error {
	// Verify board exists
	_, err := s.boardRepo.FindByID(ctx, boardID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewAppError(response.ErrCodeNotFound, "Board not found", "")
		}
		return response.NewAppError(response.ErrCodeInternal, "Failed to verify board", err.Error())
	}

	// Find all attachments associated with this board
	attachments, err := s.attachmentRepo.FindByEntityID(ctx, domain.EntityTypeBoard, boardID)
	if err != nil {
		s.logger.Warn("Failed to fetch attachments for board deletion",
			zap.String("board_id", boardID.String()),
			zap.Error(err))
		// Continue with board deletion even if attachment fetch fails
	}

	// Delete attachments from S3 and database
	if len(attachments) > 0 {
		s.deleteAttachmentsWithS3(ctx, attachments)
	}

	// Delete board
	if err := s.boardRepo.Delete(ctx, boardID); err != nil {
		return response.NewAppError(response.ErrCodeInternal, "Failed to delete board", err.Error())
	}

	return nil
}

// convertBoardCustomFieldsToValues converts a single board's customFields from IDs to values
