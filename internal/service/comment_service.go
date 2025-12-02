package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
	"project-board-api/internal/repository"
	"project-board-api/internal/response"
)

// CommentService defines the interface for comment business logic
type CommentService interface {
	CreateComment(ctx context.Context, userID uuid.UUID, req *dto.CreateCommentRequest) (*dto.CommentResponse, error)
	GetComments(ctx context.Context, boardID uuid.UUID) ([]*dto.CommentResponse, error)
	UpdateComment(ctx context.Context, commentID uuid.UUID, req *dto.UpdateCommentRequest) (*dto.CommentResponse, error)
	DeleteComment(ctx context.Context, commentID uuid.UUID) error
}

// commentServiceImpl is the implementation of CommentService
type commentServiceImpl struct {
	commentRepo    repository.CommentRepository
	boardRepo      repository.BoardRepository
	attachmentRepo repository.AttachmentRepository
	s3Client       S3Client
	logger         *zap.Logger
}

// NewCommentService creates a new instance of CommentService
func NewCommentService(commentRepo repository.CommentRepository, boardRepo repository.BoardRepository, attachmentRepo repository.AttachmentRepository, s3Client S3Client, logger *zap.Logger) CommentService {
	return &commentServiceImpl{
		commentRepo:    commentRepo,
		boardRepo:      boardRepo,
		attachmentRepo: attachmentRepo,
		s3Client:       s3Client,
		logger:         logger,
	}
}

// CreateComment creates a new comment on a board
func (s *commentServiceImpl) CreateComment(ctx context.Context, userID uuid.UUID, req *dto.CreateCommentRequest) (*dto.CommentResponse, error) {
	// Verify board exists
	_, err := s.boardRepo.FindByID(ctx, req.BoardID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewAppError(response.ErrCodeNotFound, "Board not found", "")
		}
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to verify board", err.Error())
	}

	// Validate and confirm attachments if provided
	if len(req.AttachmentIDs) > 0 {
		if err := s.validateAndConfirmAttachments(ctx, req.AttachmentIDs, domain.EntityTypeComment); err != nil {
			return nil, err
		}
	}

	// Create domain model from request
	comment := &domain.Comment{
		BoardID: req.BoardID,
		UserID:  userID,
		Content: req.Content,
	}

	// Save to repository
	if err := s.commentRepo.Create(ctx, comment); err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to create comment", err.Error())
	}

	// Confirm attachments after comment creation
	var createdAttachments []*domain.Attachment
	if len(req.AttachmentIDs) > 0 {
		// 에러 발생 시 comment도 롤백
		if err := s.attachmentRepo.ConfirmAttachments(ctx, req.AttachmentIDs, comment.ID); err != nil {
			s.logger.Error("Failed to confirm attachments, rolling back comment creation",
				zap.String("comment_id", comment.ID.String()),
				zap.Strings("attachment_ids", func() []string {
					ids := make([]string, len(req.AttachmentIDs))
					for i, id := range req.AttachmentIDs {
						ids[i] = id.String()
					}
					return ids
				}()),
				zap.Error(err))

			// comment 삭제 (롤백)
			if deleteErr := s.commentRepo.Delete(ctx, comment.ID); deleteErr != nil {
				s.logger.Error("Failed to rollback comment after attachment confirmation failure",
					zap.String("comment_id", comment.ID.String()),
					zap.Error(deleteErr))
			}

			return nil, response.NewAppError(response.ErrCodeInternal,
				"Failed to confirm attachments: "+err.Error(),
				"Please ensure all attachment IDs are valid and not already used")
		}

		// Confirm 후 Attachments 메타데이터를 조회하여 comment 객체에 할당
		attachments, err := s.attachmentRepo.FindByIDs(ctx, req.AttachmentIDs)
		if err != nil {
			s.logger.Warn("Failed to fetch confirmed attachments for response", zap.Error(err))
		} else {
			createdAttachments = attachments
		}
	}

	// 생성된 Attachments를 Comment 객체에 할당 (타입 변환 적용)
	comment.Attachments = toDomainAttachments(createdAttachments)

	// Convert to response DTO
	return s.toCommentResponse(comment), nil
}

// GetComments retrieves all comments for a board
func (s *commentServiceImpl) GetComments(ctx context.Context, boardID uuid.UUID) ([]*dto.CommentResponse, error) {
	// Verify board exists
	_, err := s.boardRepo.FindByID(ctx, boardID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewAppError(response.ErrCodeNotFound, "Board not found", "")
		}
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to verify board", err.Error())
	}

	// Fetch comments from repository
	comments, err := s.commentRepo.FindByBoardID(ctx, boardID)
	if err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch comments", err.Error())
	}

	// Comment 목록 조회 시 Attachments 로드 (각 comment별로 로드)
	for _, comment := range comments {
		attachments, err := s.attachmentRepo.FindByEntityID(ctx, domain.EntityTypeComment, comment.ID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Error("Failed to fetch attachments for comment list", zap.String("comment_id", comment.ID.String()), zap.Error(err))
		}
		comment.Attachments = toDomainAttachments(attachments)
	}

	// Convert to response DTOs
	responses := make([]*dto.CommentResponse, len(comments))
	for i, comment := range comments {
		responses[i] = s.toCommentResponse(comment)
	}

	return responses, nil
}

// UpdateComment updates a comment's content
func (s *commentServiceImpl) UpdateComment(ctx context.Context, commentID uuid.UUID, req *dto.UpdateCommentRequest) (*dto.CommentResponse, error) {
	// Fetch existing comment
	comment, err := s.commentRepo.FindByID(ctx, commentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewAppError(response.ErrCodeNotFound, "Comment not found", "")
		}
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch comment", err.Error())
	}

	// Validate and confirm attachments if provided
	if len(req.AttachmentIDs) > 0 {
		if err := s.validateAndConfirmAttachments(ctx, req.AttachmentIDs, domain.EntityTypeComment); err != nil {
			return nil, err
		}
	}

	// Update content
	comment.Content = req.Content

	// Save updated comment
	if err := s.commentRepo.Update(ctx, comment); err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to update comment", err.Error())
	}

	// Attachments 처리 로직 개선 및 Confirm
	if len(req.AttachmentIDs) > 0 {
		// 에러 발생 시 업데이트 실패 처리
		if err := s.attachmentRepo.ConfirmAttachments(ctx, req.AttachmentIDs, comment.ID); err != nil {
			s.logger.Error("Failed to confirm attachments during comment update",
				zap.String("comment_id", comment.ID.String()),
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

	// comment와 연결된 모든 Attachments를 다시 조회합니다. (타입 변환 적용)
	allAttachments, err := s.attachmentRepo.FindByEntityID(ctx, domain.EntityTypeComment, comment.ID)
	if err != nil {
		s.logger.Warn("Failed to fetch all confirmed attachments after update", zap.Error(err))
		// 치명적인 오류가 아니므로 계속 진행
	} else {
		// DB에서 최신 Attachments 목록을 로드하여 comment 객체에 할당
		comment.Attachments = toDomainAttachments(allAttachments)
	}

	// Convert to response DTO
	return s.toCommentResponse(comment), nil
}

// DeleteComment soft deletes a comment and its associated attachments
func (s *commentServiceImpl) DeleteComment(ctx context.Context, commentID uuid.UUID) error {
	// Verify comment exists
	_, err := s.commentRepo.FindByID(ctx, commentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewAppError(response.ErrCodeNotFound, "Comment not found", "")
		}
		return response.NewAppError(response.ErrCodeInternal, "Failed to verify comment", err.Error())
	}

	// Find all attachments associated with this comment
	attachments, err := s.attachmentRepo.FindByEntityID(ctx, domain.EntityTypeComment, commentID)
	if err != nil {
		s.logger.Warn("Failed to fetch attachments for comment deletion",
			zap.String("comment_id", commentID.String()),
			zap.Error(err))
		// Continue with comment deletion even if attachment fetch fails
	}

	// Delete attachments from S3 and database
	if len(attachments) > 0 {
		s.deleteAttachmentsWithS3(ctx, attachments)
	}

	// Delete comment
	if err := s.commentRepo.Delete(ctx, commentID); err != nil {
		return response.NewAppError(response.ErrCodeInternal, "Failed to delete comment", err.Error())
	}

	return nil
}

// toCommentResponse converts domain.Comment to dto.CommentResponse
func (s *commentServiceImpl) toCommentResponse(comment *domain.Comment) *dto.CommentResponse {
	// Convert attachments to response DTOs with s3Client.GetFileURL
	attachments := make([]dto.AttachmentResponse, 0, len(comment.Attachments))
	for _, a := range comment.Attachments {
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

	return &dto.CommentResponse{
		CommentID:   comment.ID,
		BoardID:     comment.BoardID,
		UserID:      comment.UserID,
		Content:     comment.Content,
		Attachments: attachments,
		CreatedAt:   comment.CreatedAt,
		UpdatedAt:   comment.UpdatedAt,
	}
}

// validateAndConfirmAttachments validates that attachments exist and are in TEMP status
func (s *commentServiceImpl) validateAndConfirmAttachments(ctx context.Context, attachmentIDs []uuid.UUID, entityType domain.EntityType) error {
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
func (s *commentServiceImpl) deleteAttachmentsWithS3(ctx context.Context, attachments []*domain.Attachment) {
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
