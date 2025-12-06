package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
	"project-board-api/internal/response"
)

func (s *projectServiceImpl) toProjectResponse(project *domain.Project) *dto.ProjectResponse {
	// Convert attachments to response DTOs
	attachments := make([]dto.AttachmentResponse, 0, len(project.Attachments))
	for _, a := range project.Attachments {

		// 💡 [수정] s3Client.GetFileURL을 사용하여 FileURL 필드 채우기 (DB의 FileURL은 S3 Key)
		fileURL := s.s3Client.GetFileURL(a.FileURL)

		attachments = append(attachments, dto.AttachmentResponse{
			ID:       a.ID,
			FileName: a.FileName,
			// 💡 FileURL 필드 채우기: S3 Key를 통해 다운로드 URL 생성
			FileURL:     fileURL,
			FileSize:    a.FileSize,
			ContentType: a.ContentType,
			UploadedBy:  a.UploadedBy,
			UploadedAt:  a.CreatedAt,
		})
	}

	return &dto.ProjectResponse{
		ID:          project.ID,
		WorkspaceID: project.WorkspaceID,
		OwnerID:     project.OwnerID,
		Name:        project.Name,
		Description: project.Description,
		StartDate:   project.StartDate,
		DueDate:     project.DueDate,
		IsPublic:    project.IsPublic,
		Attachments: attachments,
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
	}
}

// toProjectResponseWithProfile converts domain.Project to dto.ProjectResponse with owner profile

func (s *projectServiceImpl) toProjectResponseWithProfile(ctx context.Context, project *domain.Project, token string) *dto.ProjectResponse {
	// nil 체크 - project가 nil이면 nil 반환
	if project == nil {
		return nil
	}

	response := s.toProjectResponse(project)
	// response가 nil이면 nil 반환
	if response == nil {
		return nil
	}

	// Fetch workspace profile for owner - graceful degradation
	profile, err := s.userClient.GetWorkspaceProfile(ctx, project.WorkspaceID, project.OwnerID, token)
	if err != nil {
		// 에러 발생 시 owner 정보 없이 반환 (graceful degradation)
		return response
	}

	// profile이 nil이 아닐 때만 정보 추가
	if profile != nil {
		response.OwnerEmail = profile.Email
		response.OwnerName = profile.NickName
	}

	return response
}

// GetProject retrieves a project by ID with membership validation

func (s *projectServiceImpl) createDefaultFieldOptions(ctx context.Context, projectID uuid.UUID) error {
	// Get hardcoded default options
	templates := getDefaultFieldOptions()

	// Create project-specific options from templates
	projectOptions := make([]*domain.FieldOption, len(templates))
	for i, template := range templates {
		projectOptions[i] = &domain.FieldOption{
			ProjectID:       &projectID,
			FieldType:       template.FieldType,
			Value:           template.Value,
			Label:           template.Label,
			Color:           template.Color,
			DisplayOrder:    template.DisplayOrder,
			IsSystemDefault: false,
		}
	}

	// Batch create all project options
	if err := s.fieldOptionRepo.CreateBatch(ctx, projectOptions); err != nil {
		return err
	}

	return nil
}

// GetProjectInitSettings retrieves initial settings for a project including field definitions

func validateProjectDateRange(startDate, dueDate *time.Time) error {
	if startDate != nil && dueDate != nil {
		if startDate.After(*dueDate) {
			return response.NewAppError(response.ErrCodeValidation, "Start date cannot be after due date", "")
		}
	}
	return nil
}

// validateAndConfirmAttachments validates that attachments exist and are in TEMP status

func (s *projectServiceImpl) validateAndConfirmAttachments(ctx context.Context, attachmentIDs []uuid.UUID, entityType domain.EntityType) error {
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

func (s *projectServiceImpl) deleteAttachmentsWithS3(ctx context.Context, attachments []*domain.Attachment) {
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
		// 💡 [개선] S3 파일 삭제도 병렬 처리가 가능하도록 고루틴을 활용할 수 있으나,
		// 현재는 상위 UpdateProject에서 전체 호출을 비동기화했으므로, 이 함수 자체는 동기적으로 유지해도 됩니다.
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
