package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
	"project-board-api/internal/response"
)

// convertBoardCustomFieldsToValues converts custom field IDs to values for display
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
