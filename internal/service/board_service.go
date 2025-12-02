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
func (s *boardServiceImpl) UpdateBoard(ctx context.Context, boardID uuid.UUID, req *dto.UpdateBoardRequest) (*dto.BoardResponse, error) {
	// Fetch existing board
	board, err := s.boardRepo.FindByID(ctx, boardID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewAppError(response.ErrCodeNotFound, "Board not found", "")
		}
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch board", err.Error())
	}

	// Determine the effective start and due dates for validation
	effectiveStartDate := board.StartDate
	effectiveDueDate := board.DueDate

	if req.StartDate != nil {
		effectiveStartDate = req.StartDate
	}
	if req.DueDate != nil {
		effectiveDueDate = req.DueDate
	}

	// Validate date range with effective dates
	if err := validateDateRange(effectiveStartDate, effectiveDueDate); err != nil {
		return nil, err
	}

	// Validate and confirm attachments if provided
	if len(req.AttachmentIDs) > 0 {
		if err := s.validateAndConfirmAttachments(ctx, req.AttachmentIDs, domain.EntityTypeBoard, uuid.Nil); err != nil {
			return nil, err
		}
	}

	// Update fields if provided
	if req.Title != nil {
		board.Title = *req.Title
	}
	if req.Content != nil {
		board.Content = *req.Content
	}
	if req.CustomFields != nil {
		// Convert values to IDs
		convertedFields, err := s.fieldOptionConverter.ConvertValuesToIDs(ctx, board.ProjectID, *req.CustomFields)
		if err != nil {
			return nil, response.NewAppError(response.ErrCodeValidation, "Invalid custom field values", err.Error())
		}

		// Convert CustomFields to datatypes.JSON
		jsonBytes, err := json.Marshal(convertedFields)
		if err != nil {
			return nil, response.NewAppError(response.ErrCodeInternal, "Failed to marshal custom fields", err.Error())
		}
		board.CustomFields = jsonBytes
	}
	if req.AssigneeID != nil {
		if *req.AssigneeID == uuid.Nil {
			board.AssigneeID = nil
		} else {
			board.AssigneeID = req.AssigneeID
		}
	}
	if req.StartDate != nil {
		board.StartDate = req.StartDate
	}
	if req.DueDate != nil {
		board.DueDate = req.DueDate
	}

	// Update board first
	if err := s.boardRepo.Update(ctx, board); err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to update board", err.Error())
	}

	// Attachments 처리 로직 개선 및 Confirm
	if len(req.AttachmentIDs) > 0 {
		// 에러 발생 시 업데이트 실패 처리
		if err := s.attachmentRepo.ConfirmAttachments(ctx, req.AttachmentIDs, board.ID); err != nil {
			s.logger.Error("Failed to confirm attachments during board update",
				zap.String("board_id", board.ID.String()),
				zap.Strings("attachment_ids", func() []string {
					ids := make([]string, len(req.AttachmentIDs))
					for i, id := range req.AttachmentIDs {
						ids[i] = id.String()
					}
					return ids
				}()),
				zap.Error(err))

			return nil, response.NewAppError(response.ErrCodeInternal,
				"Failed to confirm attachments: "+err.Error(),
				"Please ensure all attachment IDs are valid and not already used")
		}
	}

	// ✅ [수정] Participants 업데이트 로직 - board 업데이트 후 처리
	if req.Participants != nil {
		// 1. 기존 참여자 모두 조회
		existingParticipants, err := s.participantRepo.FindByBoardID(ctx, boardID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Warn("Failed to fetch existing participants for update",
				zap.String("board_id", boardID.String()),
				zap.Error(err))
		}

		// 2. 기존 참여자 모두 삭제
		if len(existingParticipants) > 0 {
			s.logger.Info("Deleting existing participants",
				zap.String("board_id", boardID.String()),
				zap.Int("count", len(existingParticipants)))

			for _, p := range existingParticipants {
				if err := s.participantRepo.Delete(ctx, boardID, p.UserID); err != nil {
					s.logger.Warn("Failed to delete existing participant",
						zap.String("board_id", boardID.String()),
						zap.String("user_id", p.UserID.String()),
						zap.Error(err))
				}
			}
		}

		// 3. 새로운 참여자 추가
		if len(req.Participants) > 0 {
			s.logger.Info("Adding new participants",
				zap.String("board_id", boardID.String()),
				zap.Int("count", len(req.Participants)))

			uniqueUserIDs := removeDuplicateUUIDs(req.Participants)
			for _, userID := range uniqueUserIDs {
				participant := &domain.Participant{
					BoardID: boardID,
					UserID:  userID,
				}
				if err := s.participantRepo.Create(ctx, participant); err != nil {
					s.logger.Warn("Failed to add new participant",
						zap.String("board_id", boardID.String()),
						zap.String("user_id", userID.String()),
						zap.Error(err))
				}
			}
		}
	}

	// board와 연결된 모든 Attachments를 다시 조회합니다. (타입 변환 적용)
	allAttachments, err := s.attachmentRepo.FindByEntityID(ctx, domain.EntityTypeBoard, board.ID)
	if err != nil {
		s.logger.Warn("Failed to fetch all confirmed attachments after update", zap.Error(err))
	} else {
		board.Attachments = toDomainAttachments(allAttachments)
	}

	// ✅ [수정] 업데이트된 participants를 다시 로드
	reloadedBoard, err := s.boardRepo.FindByID(ctx, board.ID)
	if err != nil {
		s.logger.Warn("Failed to reload board with participants after update",
			zap.String("board_id", board.ID.String()),
			zap.Error(err))
	} else {
		board = reloadedBoard
		// Attachments는 위에서 이미 로드했으므로 다시 할당
		board.Attachments = toDomainAttachments(allAttachments)
	}

	// Convert to response DTO
	return s.toBoardResponse(board), nil
}

// DeleteBoard soft deletes a board and its associated attachments
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
func (s *boardServiceImpl) convertBoardCustomFieldsToValues(ctx context.Context, board *domain.Board) error {
	if board.CustomFields == nil || len(board.CustomFields) == 0 {
		return nil
	}

	var customFields map[string]interface{}
	if err := json.Unmarshal(board.CustomFields, &customFields); err != nil {
		return err
	}

	convertedFields, err := s.fieldOptionConverter.ConvertIDsToValues(ctx, customFields)
	if err != nil {
		return err
	}

	jsonBytes, err := json.Marshal(convertedFields)
	if err != nil {
		return err
	}

	board.CustomFields = jsonBytes
	return nil
}

// toBoardResponse converts domain.Board to dto.BoardResponse
func (s *boardServiceImpl) toBoardResponse(board *domain.Board) *dto.BoardResponse {
	// Convert datatypes.JSON to map[string]interface{}
	var customFields map[string]interface{}
	if len(board.CustomFields) > 0 {
		_ = json.Unmarshal(board.CustomFields, &customFields)
	}

	// Extract participant IDs from board participants
	participantIDs := make([]uuid.UUID, 0, len(board.Participants))
	for _, p := range board.Participants {
		participantIDs = append(participantIDs, p.UserID)
	}

	// Convert attachments to response DTOs with s3Client.GetFileURL
	attachments := make([]dto.AttachmentResponse, 0, len(board.Attachments))
	for _, a := range board.Attachments {
		// s3Client.GetFileURL을 사용하여 FileURL 필드 채우기 (DB의 FileURL은 S3 Key)
		fileURL := s.s3Client.GetFileURL(a.FileURL)

		attachments = append(attachments, dto.AttachmentResponse{
			ID:          a.ID,
			FileName:    a.FileName,
			FileURL:     fileURL, // full URL 반환
			FileSize:    a.FileSize,
			ContentType: a.ContentType,
			UploadedBy:  a.UploadedBy,
			UploadedAt:  a.CreatedAt,
		})
	}

	return &dto.BoardResponse{
		ID:             board.ID,
		ProjectID:      board.ProjectID,
		AuthorID:       board.AuthorID,
		AssigneeID:     board.AssigneeID,
		Title:          board.Title,
		Content:        board.Content,
		CustomFields:   customFields,
		StartDate:      board.StartDate,
		DueDate:        board.DueDate,
		ParticipantIDs: participantIDs,
		Attachments:    attachments,
		CreatedAt:      board.CreatedAt,
		UpdatedAt:      board.UpdatedAt,
	}
}

// toBoardDetailResponse converts domain.Board to dto.BoardDetailResponse
func (s *boardServiceImpl) toBoardDetailResponse(board *domain.Board) *dto.BoardDetailResponse {
	// Convert participants
	participants := make([]dto.ParticipantResponse, len(board.Participants))
	for i, p := range board.Participants {
		participants[i] = dto.ParticipantResponse{
			ID:        p.ID,
			BoardID:   p.BoardID,
			UserID:    p.UserID,
			CreatedAt: p.CreatedAt,
		}
	}

	// Convert comments
	comments := make([]dto.CommentResponse, len(board.Comments))
	for i, c := range board.Comments {
		comments[i] = dto.CommentResponse{
			CommentID: c.ID,
			BoardID:   c.BoardID,
			UserID:    c.UserID,
			Content:   c.Content,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		}
	}

	return &dto.BoardDetailResponse{
		BoardResponse: *s.toBoardResponse(board),
		Participants:  participants,
		Comments:      comments,
	}
}

// addParticipantsInternal is an internal helper to add participants during board creation
// It does not verify board existence (assumes board was just created)
// Returns the number of successfully added participants and any errors
func (s *boardServiceImpl) addParticipantsInternal(ctx context.Context, boardID uuid.UUID, userIDs []uuid.UUID) (int, error) {
	// Remove duplicates from the user IDs
	uniqueUserIDs := removeDuplicateUUIDs(userIDs)

	successCount := 0
	var failedUserIDs []uuid.UUID

	// Process each participant individually
	for _, userID := range uniqueUserIDs {
		// Check if participant already exists
		existing, err := s.participantRepo.FindByBoardAndUser(ctx, boardID, userID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Warn("Failed to check existing participant",
				zap.String("board_id", boardID.String()),
				zap.String("user_id", userID.String()),
				zap.Error(err))
			failedUserIDs = append(failedUserIDs, userID)
			continue
		}
		if existing != nil {
			// Participant already exists, skip
			continue
		}

		// Create domain model
		participant := &domain.Participant{
			BoardID: boardID,
			UserID:  userID,
		}

		// Save to repository
		if err := s.participantRepo.Create(ctx, participant); err != nil {
			// Check for unique constraint violation
			if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
				// Participant already exists, skip
				continue
			}
			s.logger.Warn("Failed to add participant",
				zap.String("board_id", boardID.String()),
				zap.String("user_id", userID.String()),
				zap.Error(err))
			failedUserIDs = append(failedUserIDs, userID)
			continue
		}

		successCount++
	}

	// Log summary if there were failures
	if len(failedUserIDs) > 0 {
		s.logger.Warn("Some participants failed to be added during board creation",
			zap.String("board_id", boardID.String()),
			zap.Int("success_count", successCount),
			zap.Int("failed_count", len(failedUserIDs)),
			zap.Any("failed_user_ids", failedUserIDs))
	}

	return successCount, nil
}

// validateAndConfirmAttachments validates that attachments exist and are in TEMP status
func (s *boardServiceImpl) validateAndConfirmAttachments(ctx context.Context, attachmentIDs []uuid.UUID, entityType domain.EntityType, entityID uuid.UUID) error {
	if len(attachmentIDs) == 0 {
		return nil
	}

	// Fetch attachments by IDs
	attachments, err := s.attachmentRepo.FindByIDs(ctx, attachmentIDs)
	if err != nil {
		return response.NewAppError(response.ErrCodeInternal, "Failed to fetch attachments", err.Error())
	}

	// Check if all attachments exist
	if len(attachments) != len(attachmentIDs) {
		return response.NewAppError(response.ErrCodeValidation, "One or more attachments not found", "")
	}

	// Validate each attachment
	for _, attachment := range attachments {
		// Check if attachment is in TEMP status
		if attachment.Status != domain.AttachmentStatusTemp {
			return response.NewAppError(response.ErrCodeValidation, "Attachment is not in temporary status and cannot be reused", "")
		}

		// Check if attachment entity type matches
		if attachment.EntityType != entityType {
			return response.NewAppError(response.ErrCodeValidation, "Attachment entity type does not match", "")
		}
	}

	return nil
}

// deleteAttachmentsWithS3 deletes attachments from both S3 and database
func (s *boardServiceImpl) deleteAttachmentsWithS3(ctx context.Context, attachments []*domain.Attachment) {
	attachmentIDs := make([]uuid.UUID, 0, len(attachments))

	// Delete files from S3
	for _, attachment := range attachments {
		// Extract S3 key from FileURL
		fileKey := extractS3KeyFromURL(attachment.FileURL)
		if fileKey == "" {
			s.logger.Warn("Failed to extract S3 key from URL",
				zap.String("attachment_id", attachment.ID.String()),
				zap.String("file_url", attachment.FileURL))
			continue
		}

		// Delete from S3
		if err := s.s3Client.DeleteFile(ctx, fileKey); err != nil {
			s.logger.Warn("Failed to delete file from S3",
				zap.String("attachment_id", attachment.ID.String()),
				zap.String("file_key", fileKey),
				zap.Error(err))
			// Continue even if S3 deletion fails
		}

		attachmentIDs = append(attachmentIDs, attachment.ID)
	}

	// Delete from database
	if len(attachmentIDs) > 0 {
		if err := s.attachmentRepo.DeleteBatch(ctx, attachmentIDs); err != nil {
			s.logger.Warn("Failed to delete attachments from database",
				zap.Int("count", len(attachmentIDs)),
				zap.Error(err))
		}
	}
}
