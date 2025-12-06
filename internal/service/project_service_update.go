package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
	"project-board-api/internal/response"
)

func (s *projectServiceImpl) UpdateProject(ctx context.Context, projectID, userID uuid.UUID, req *dto.UpdateProjectRequest) (*dto.ProjectResponse, error) {
	// Fetch project from repository
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError("Project not found", "")
		}
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch project", err.Error())
	}

	// Check if user is the project owner
	member, err := s.projectRepo.FindMemberByProjectAndUser(ctx, projectID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewForbiddenError("You are not a member of this project", "")
		}
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to check membership", err.Error())
	}
	if member.RoleName != domain.ProjectRoleOwner {
		return nil, response.NewForbiddenError("Only project owner can update project", "")
	}

	// Determine the effective start and due dates for validation
	effectiveStartDate := project.StartDate
	effectiveDueDate := project.DueDate

	if req.StartDate != nil {
		effectiveStartDate = req.StartDate
	}
	if req.DueDate != nil {
		effectiveDueDate = req.DueDate
	}

	// Validate date range with effective dates
	if err := validateProjectDateRange(effectiveStartDate, effectiveDueDate); err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.Name != nil {
		project.Name = *req.Name
	}
	if req.Description != nil {
		project.Description = *req.Description
	}
	if req.StartDate != nil {
		project.StartDate = req.StartDate
	}
	if req.DueDate != nil {
		project.DueDate = req.DueDate
	}

	// Save to repository
	if err := s.projectRepo.Update(ctx, project); err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to update project", err.Error())
	}

	// 🔥 attachmentIds 처리 (기존 삭제 후 새로 추가)
	if req.AttachmentIDs != nil {
		// 1. 기존 attachments 조회
		existingAttachments, err := s.attachmentRepo.FindByEntityID(ctx, domain.EntityTypeProject, project.ID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Error("Failed to fetch existing attachments for replacement",
				zap.String("project_id", project.ID.String()),
				zap.Error(err))
			return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch existing attachments", err.Error())
		}

		// 2. 🚨 성능 개선: 기존 attachments 삭제 로직을 비동기 고루틴으로 분리
		// S3 파일 삭제는 네트워크 I/O가 발생하여 응답 시간을 지연시키므로, 응답에 필수적이지 않은 이 작업은 고루틴으로 실행합니다.
		if len(existingAttachments) > 0 {
			// Context.Background()를 사용하여 HTTP 요청 Context의 수명과 분리
			go s.deleteAttachmentsWithS3(context.Background(), existingAttachments)
			s.logger.Debug("Asynchronously initiated deletion of existing attachments",
				zap.String("project_id", project.ID.String()),
				zap.Int("count", len(existingAttachments)))
		}

		// 3. 새 attachments confirm (ConfirmAttachments 내부에서 TEMP 검증) - DB I/O (응답에 필수)
		if len(req.AttachmentIDs) > 0 {
			if err := s.attachmentRepo.ConfirmAttachments(ctx, req.AttachmentIDs, project.ID); err != nil {
				s.logger.Error("Failed to confirm new attachments during project update",
					zap.String("project_id", project.ID.String()),
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
	}

	// 4. 최신 attachments 조회 (응답에 필수)
	allAttachments, err := s.attachmentRepo.FindByEntityID(ctx, domain.EntityTypeProject, project.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Warn("Failed to fetch all attachments after update", zap.Error(err))
	}
	project.Attachments = toDomainAttachments(allAttachments)

	return s.toProjectResponse(project), nil
}

// DeleteProject soft deletes a project and its associated attachments (OWNER only)
