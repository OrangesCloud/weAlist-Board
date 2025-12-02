package service

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"project-board-api/internal/client"
	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
	"project-board-api/internal/metrics"
	"project-board-api/internal/repository"
	"project-board-api/internal/response"
)

// 💡 [추가] S3Client 인터페이스 정의: project_service.go가 이 타입을 사용하므로 정의가 필요합니다.
// 이는 client.S3ClientInterface가 구현하는 메서드들을 포함해야 합니다.
type S3Client interface {
	GenerateFileKey(entityType, workspaceID, fileExt string) (string, error)
	GeneratePresignedURL(ctx context.Context, entityType, workspaceID, fileName, contentType string) (string, string, error)
	UploadFile(ctx context.Context, key string, file io.Reader, contentType string) (string, error)
	DeleteFile(ctx context.Context, key string) error
	GetFileURL(key string) string // 🚨 [핵심 수정] 이 메서드가 누락되어 오류가 발생했습니다.
}

// ProjectService defines the interface for project business logic
type ProjectService interface {
	CreateProject(ctx context.Context, req *dto.CreateProjectRequest, userID uuid.UUID, token string) (*dto.ProjectResponse, error)
	GetProjectsByWorkspace(ctx context.Context, workspaceID, userID uuid.UUID, token string) ([]*dto.ProjectResponse, error)
	GetDefaultProject(ctx context.Context, workspaceID, userID uuid.UUID, token string) (*dto.ProjectResponse, error)
	GetProject(ctx context.Context, projectID, userID uuid.UUID, token string) (*dto.ProjectResponse, error)
	UpdateProject(ctx context.Context, projectID, userID uuid.UUID, req *dto.UpdateProjectRequest) (*dto.ProjectResponse, error)
	DeleteProject(ctx context.Context, projectID, userID uuid.UUID) error
	SearchProjects(ctx context.Context, workspaceID, userID uuid.UUID, query string, page, limit int, token string) (*dto.PaginatedProjectsResponse, error)
	GetProjectInitSettings(ctx context.Context, projectID, userID uuid.UUID, token string) (*dto.ProjectInitSettingsResponse, error)
}

// projectServiceImpl is the implementation of ProjectService
type projectServiceImpl struct {
	projectRepo     repository.ProjectRepository
	fieldOptionRepo repository.FieldOptionRepository
	attachmentRepo  repository.AttachmentRepository
	s3Client        S3Client // 이 타입 정의가 상단에 추가되었습니다.
	userClient      client.UserClient
	metrics         *metrics.Metrics
	logger          *zap.Logger
}

// NewProjectService creates a new instance of ProjectService
func NewProjectService(projectRepo repository.ProjectRepository, fieldOptionRepo repository.FieldOptionRepository, attachmentRepo repository.AttachmentRepository, s3Client S3Client, userClient client.UserClient, m *metrics.Metrics, logger *zap.Logger) ProjectService {
	return &projectServiceImpl{
		projectRepo:     projectRepo,
		fieldOptionRepo: fieldOptionRepo,
		attachmentRepo:  attachmentRepo,
		s3Client:        s3Client,
		userClient:      userClient,
		metrics:         m,
		logger:          logger,
	}
}

// CreateProject creates a new project
func (s *projectServiceImpl) CreateProject(ctx context.Context, req *dto.CreateProjectRequest, userID uuid.UUID, token string) (*dto.ProjectResponse, error) {
	// Validate workspace membership
	isValid, err := s.userClient.ValidateWorkspaceMember(ctx, req.WorkspaceID, userID, token)
	if err != nil {
		// Log error but continue with graceful degradation
		// Return forbidden error if validation explicitly fails
		return nil, response.NewAppError(response.ErrCodeForbidden, "You are not a member of this workspace", "")
	}
	if !isValid {
		return nil, response.NewAppError(response.ErrCodeForbidden, "You are not a member of this workspace", "")
	}

	// Validate date range
	if err := validateProjectDateRange(req.StartDate, req.DueDate); err != nil {
		return nil, err
	}

	// Validate and confirm attachments if provided
	if len(req.AttachmentIDs) > 0 {
		if err := s.validateAndConfirmAttachments(ctx, req.AttachmentIDs, domain.EntityTypeProject); err != nil {
			return nil, err
		}
	}

	// Create domain model from request
	project := &domain.Project{
		WorkspaceID: req.WorkspaceID,
		OwnerID:     userID,
		Name:        req.Name,
		Description: req.Description,
		StartDate:   req.StartDate,
		DueDate:     req.DueDate,
		IsDefault:   false, // Default to false, can be changed later
		IsPublic:    false, // Default to private
	}

	// Save to repository
	if err := s.projectRepo.Create(ctx, project); err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to create project", err.Error())
	}

	// ✅ 수정: Confirm attachments after project creation
	var createdAttachments []*domain.Attachment
	if len(req.AttachmentIDs) > 0 {
		// ✅ 에러 발생 시 프로젝트도 롤백
		if err := s.attachmentRepo.ConfirmAttachments(ctx, req.AttachmentIDs, project.ID); err != nil {
			s.logger.Error("Failed to confirm attachments, rolling back project creation",
				zap.String("project_id", project.ID.String()),
				zap.Strings("attachment_ids", func() []string {
					ids := make([]string, len(req.AttachmentIDs))
					for i, id := range req.AttachmentIDs {
						ids[i] = id.String()
					}
					return ids
				}()),
				zap.Error(err))

			// ✅ 프로젝트 삭제 (롤백)
			if deleteErr := s.projectRepo.Delete(ctx, project.ID); deleteErr != nil {
				s.logger.Error("Failed to rollback project after attachment confirmation failure",
					zap.String("project_id", project.ID.String()),
					zap.Error(deleteErr))
			}

			// ✅ 에러 반환
			return nil, response.NewAppError(response.ErrCodeInternal,
				"Failed to confirm attachments: "+err.Error(),
				"Please ensure all attachment IDs are valid and not already used")
		}

		// 💡 [수정] Confirm 후 Attachments 메타데이터를 조회하여 project 객체에 할당
		// FindByIDs는 []*domain.Attachment를 반환한다고 가정합니다.
		attachments, err := s.attachmentRepo.FindByIDs(ctx, req.AttachmentIDs)
		if err != nil {
			s.logger.Warn("Failed to fetch confirmed attachments for response", zap.Error(err))
		} else {
			createdAttachments = attachments
		}
	}

	// Add creator as OWNER member
	member := &domain.ProjectMember{
		ProjectID: project.ID,
		UserID:    userID,
		RoleName:  domain.ProjectRoleOwner,
	}
	if err := s.projectRepo.AddMember(ctx, member); err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to add project owner", err.Error())
	}

	// Create default field options for the project
	if err := s.createDefaultFieldOptions(ctx, project.ID); err != nil {
		// Rollback project creation if field options fail
		s.projectRepo.Delete(ctx, project.ID)
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to create default field options", err.Error())
	}

	// Increment project creation metric
	if s.metrics != nil {
		s.metrics.IncrementProjectCreated()
	}

	// Convert to response DTO
	// 💡 [수정] 생성된 Attachments를 Project 객체에 임시 할당 (타입 변환 적용)
	project.Attachments = toDomainAttachments(createdAttachments)
	return s.toProjectResponse(project), nil
}

// GetProjectsByWorkspace retrieves all projects for a workspace
func (s *projectServiceImpl) GetProjectsByWorkspace(ctx context.Context, workspaceID, userID uuid.UUID, token string) ([]*dto.ProjectResponse, error) {
	// Validate workspace membership
	isValid, err := s.userClient.ValidateWorkspaceMember(ctx, workspaceID, userID, token)
	if err != nil {
		// Log error but continue with graceful degradation
		// Return forbidden error if validation explicitly fails
		return nil, response.NewAppError(response.ErrCodeForbidden, "You are not a member of this workspace", "")
	}
	if !isValid {
		return nil, response.NewAppError(response.ErrCodeForbidden, "You are not a member of this workspace", "")
	}

	// Fetch projects from repository
	projects, err := s.projectRepo.FindByWorkspaceID(ctx, workspaceID)
	if err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch projects", err.Error())
	}

	// 빈 배열 명시적 처리 - nil이거나 길이가 0이면 빈 배열 반환
	if projects == nil || len(projects) == 0 {
		return []*dto.ProjectResponse{}, nil
	}

	// Convert to response DTOs with owner profile information
	// 동적으로 append하여 개별 프로젝트 변환 실패 시 전체 실패 방지
	responses := make([]*dto.ProjectResponse, 0, len(projects))
	for i, project := range projects {
		// nil 프로젝트 스킵
		if project == nil {
			continue
		}

		// 💡 [추가] Project 목록 조회 시 Attachments 로드 (효율을 위해 bulk load 고려 가능)
		attachments, err := s.attachmentRepo.FindByEntityID(ctx, domain.EntityTypeProject, project.ID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Error("Failed to fetch attachments for project list", zap.String("project_id", project.ID.String()), zap.Error(err))
		}
		project.Attachments = toDomainAttachments(attachments) // 🚨 타입 변환 적용

		// 개별 변환 실패 시 해당 프로젝트만 스킵
		projectResp := s.toProjectResponseWithProfile(ctx, project, token)
		if projectResp != nil {
			responses = append(responses, projectResp)
		} else {
			// Log when a project response is nil to help debugging
			_ = i // Avoid unused variable warning
		}
	}

	return responses, nil
}

// GetDefaultProject retrieves the default project for a workspace
func (s *projectServiceImpl) GetDefaultProject(ctx context.Context, workspaceID, userID uuid.UUID, token string) (*dto.ProjectResponse, error) {
	// Validate workspace membership
	isValid, err := s.userClient.ValidateWorkspaceMember(ctx, workspaceID, userID, token)
	if err != nil {
		// Log error but continue with graceful degradation
		// Return forbidden error if validation explicitly fails
		return nil, response.NewAppError(response.ErrCodeForbidden, "You are not a member of this workspace", "")
	}
	if !isValid {
		return nil, response.NewAppError(response.ErrCodeForbidden, "You are not a member of this workspace", "")
	}

	// Fetch default project from repository
	project, err := s.projectRepo.FindDefaultByWorkspaceID(ctx, workspaceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError("Default project not found", "")
		}
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch default project", err.Error())
	}

	// 💡 [추가] Attachments 로드 (타입 변환 적용)
	attachments, err := s.attachmentRepo.FindByEntityID(ctx, domain.EntityTypeProject, project.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Error("Failed to fetch attachments for default project", zap.String("project_id", project.ID.String()), zap.Error(err))
	}
	project.Attachments = toDomainAttachments(attachments) // 🚨 타입 변환 적용

	// Convert to response DTO with owner profile information
	return s.toProjectResponseWithProfile(ctx, project, token), nil
}

// toProjectResponse converts domain.Project to dto.ProjectResponse
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
func (s *projectServiceImpl) GetProject(ctx context.Context, projectID, userID uuid.UUID, token string) (*dto.ProjectResponse, error) {
	// Fetch project from repository
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError("Project not found", "")
		}
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch project", err.Error())
	}

	// Check if user is a project member
	isMember, err := s.projectRepo.IsProjectMember(ctx, projectID, userID)
	if err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to check membership", err.Error())
	}

	// If not a project member, check workspace membership
	// TODO: 향후 프로젝트별 권한 관리 기능 구현 시 수정 필요
	if !isMember {
		s.logger.Debug("User is not a project member, checking workspace membership",
			zap.String("project_id", projectID.String()),
			zap.String("workspace_id", project.WorkspaceID.String()),
			zap.String("user_id", userID.String()),
		)

		isWorkspaceMember, err := s.userClient.ValidateWorkspaceMember(ctx, project.WorkspaceID, userID, token)
		if err != nil {
			s.logger.Error("Failed to validate workspace membership",
				zap.Error(err),
				zap.String("project_id", projectID.String()),
				zap.String("workspace_id", project.WorkspaceID.String()),
				zap.String("user_id", userID.String()),
			)
			return nil, response.NewForbiddenError("You are not a member of this project or workspace", "")
		}

		if !isWorkspaceMember {
			s.logger.Warn("Access denied: user is neither project member nor workspace member",
				zap.String("project_id", projectID.String()),
				zap.String("workspace_id", project.WorkspaceID.String()),
				zap.String("user_id", userID.String()),
			)
			return nil, response.NewForbiddenError("You are not a member of this project or workspace", "")
		}

		// Workspace member access granted - log for future audit and permission management
		// Note: This allows workspace members to access all projects in their workspace
		// until project-level permission management is implemented
		s.logger.Info("Access granted via workspace membership",
			zap.String("access_type", "workspace_member"),
			zap.String("project_id", projectID.String()),
			zap.String("workspace_id", project.WorkspaceID.String()),
			zap.String("user_id", userID.String()),
			zap.String("project_name", project.Name),
			zap.String("note", "Project-level permissions not yet implemented"),
		)
	} else {
		s.logger.Debug("Access granted via project membership",
			zap.String("access_type", "project_member"),
			zap.String("project_id", projectID.String()),
			zap.String("user_id", userID.String()),
		)
	}

	// 💡 [추가] Attachments 로드 (타입 변환 적용)
	attachments, err := s.attachmentRepo.FindByEntityID(ctx, domain.EntityTypeProject, project.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Error("Failed to fetch attachments for project", zap.String("project_id", project.ID.String()), zap.Error(err))
		// Continue with graceful degradation
	}
	project.Attachments = toDomainAttachments(attachments) // 🚨 타입 변환 적용

	// Convert to response DTO with owner profile information
	return s.toProjectResponseWithProfile(ctx, project, token), nil
}

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
func (s *projectServiceImpl) DeleteProject(ctx context.Context, projectID, userID uuid.UUID) error {
	// Fetch project from repository
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError("Project not found", "")
		}
		return response.NewAppError(response.ErrCodeInternal, "Failed to fetch project", err.Error())
	}

	// Check if user is the project owner
	member, err := s.projectRepo.FindMemberByProjectAndUser(ctx, projectID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewForbiddenError("You are not a member of this project", "")
		}
		return response.NewAppError(response.ErrCodeInternal, "Failed to check membership", err.Error())
	}
	if member.RoleName != domain.ProjectRoleOwner {
		return response.NewForbiddenError("Only project owner can delete project", "")
	}

	// Find all attachments associated with this project
	attachments, err := s.attachmentRepo.FindByEntityID(ctx, domain.EntityTypeProject, projectID)
	if err != nil {
		s.logger.Warn("Failed to fetch attachments for project deletion",
			zap.String("project_id", projectID.String()),
			zap.Error(err))
		// Continue with project deletion even if attachment fetch fails
	}

	// Delete attachments from S3 and database
	if len(attachments) > 0 {
		// 💡 [수정] DeleteProject에서도 S3 삭제는 비동기로 처리하여 응답 시간을 개선할 수 있으나,
		// 데이터 정합성 관점에서 (프로젝트가 DB에서 삭제되었으므로 파일 삭제는 필수),
		// 여기서는 동기적으로 유지하거나 (가장 보수적), 고루틴을 사용하되 waitGroup을 사용해 완료를 기다리는 방식(가장 이상적)이 좋습니다.
		// 현재는 기존 로직을 유지하여 동기적으로 처리합니다.
		s.deleteAttachmentsWithS3(ctx, attachments)
	}

	// Delete from repository
	if err := s.projectRepo.Delete(ctx, project.ID); err != nil {
		return response.NewAppError(response.ErrCodeInternal, "Failed to delete project", err.Error())
	}

	return nil
}

// SearchProjects searches projects by name or description with workspace membership validation
func (s *projectServiceImpl) SearchProjects(ctx context.Context, workspaceID, userID uuid.UUID, query string, page, limit int, token string) (*dto.PaginatedProjectsResponse, error) {
	// Validate workspace membership
	isValid, err := s.userClient.ValidateWorkspaceMember(ctx, workspaceID, userID, token)
	if err != nil {
		return nil, response.NewAppError(response.ErrCodeForbidden, "You are not a member of this workspace", "")
	}
	if !isValid {
		return nil, response.NewAppError(response.ErrCodeForbidden, "You are not a member of this workspace", "")
	}

	// Validate query parameter
	if query == "" {
		return nil, response.NewValidationError("Search query cannot be empty", "")
	}

	// Set default pagination values
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Search projects from repository
	projects, total, err := s.projectRepo.Search(ctx, workspaceID, query, page, limit)
	if err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to search projects", err.Error())
	}

	// Convert to response DTOs with owner profile information
	responses := make([]dto.ProjectResponse, len(projects))
	for i, project := range projects {

		// 💡 [추가] 검색 목록 조회 시 Attachments 로드 (타입 변환 적용)
		attachments, err := s.attachmentRepo.FindByEntityID(ctx, domain.EntityTypeProject, project.ID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Error("Failed to fetch attachments for project list", zap.String("project_id", project.ID.String()), zap.Error(err))
		}
		project.Attachments = toDomainAttachments(attachments) // 🚨 타입 변환 적용

		responses[i] = *s.toProjectResponseWithProfile(ctx, project, token)
	}

	return &dto.PaginatedProjectsResponse{
		Projects: responses,
		Total:    total,
		Page:     page,
		Limit:    limit,
	}, nil
}

// createDefaultFieldOptions creates default field options for a new project
// using hardcoded default values
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
func (s *projectServiceImpl) GetProjectInitSettings(ctx context.Context, projectID, userID uuid.UUID, token string) (*dto.ProjectInitSettingsResponse, error) {
	// Fetch project from repository
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError("Project not found", "")
		}
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch project", err.Error())
	}

	// Check if user is a project member
	isMember, err := s.projectRepo.IsProjectMember(ctx, projectID, userID)
	if err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to check membership", err.Error())
	}

	// If not a project member, check workspace membership
	// TODO: 향후 프로젝트별 권한 관리 기능 구현 시 수정 필요
	if !isMember {
		s.logger.Debug("User is not a project member, checking workspace membership for init settings",
			zap.String("project_id", projectID.String()),
			zap.String("workspace_id", project.WorkspaceID.String()),
			zap.String("user_id", userID.String()),
		)

		isWorkspaceMember, err := s.userClient.ValidateWorkspaceMember(ctx, project.WorkspaceID, userID, token)
		if err != nil {
			s.logger.Error("Failed to validate workspace membership for init settings",
				zap.Error(err),
				zap.String("project_id", projectID.String()),
				zap.String("workspace_id", project.WorkspaceID.String()),
				zap.String("user_id", userID.String()),
			)
			return nil, response.NewForbiddenError("You are not a member of this project or workspace", "")
		}

		if !isWorkspaceMember {
			s.logger.Warn("Access denied to init settings: user is neither project member nor workspace member",
				zap.String("project_id", projectID.String()),
				zap.String("workspace_id", project.WorkspaceID.String()),
				zap.String("user_id", userID.String()),
			)
			return nil, response.NewForbiddenError("You are not a member of this project or workspace", "")
		}

		// Workspace member access granted - log for future audit and permission management
		// Note: This allows workspace members to access all projects in their workspace
		// until project-level permission management is implemented
		s.logger.Info("Access granted to init settings via workspace membership",
			zap.String("access_type", "workspace_member"),
			zap.String("project_id", projectID.String()),
			zap.String("workspace_id", project.WorkspaceID.String()),
			zap.String("user_id", userID.String()),
			zap.String("project_name", project.Name),
			zap.String("note", "Project-level permissions not yet implemented"),
		)
	} else {
		s.logger.Debug("Access granted to init settings via project membership",
			zap.String("access_type", "project_member"),
			zap.String("project_id", projectID.String()),
			zap.String("user_id", userID.String()),
		)
	}

	// Fetch workspace information
	workspace, err := s.userClient.GetWorkspace(ctx, project.WorkspaceID, token)
	if err != nil {
		// Log error but continue with graceful degradation
		workspace = &client.Workspace{
			ID:   project.WorkspaceID,
			Name: "",
		}
	}

	// Fetch owner profile information
	ownerProfile, err := s.userClient.GetWorkspaceProfile(ctx, project.WorkspaceID, project.OwnerID, token)
	if err != nil {
		// Log error but continue with graceful degradation
		s.logger.Warn("Failed to fetch owner profile for project init settings",
			zap.Error(err),
			zap.String("project_id", projectID.String()),
			zap.String("owner_id", project.OwnerID.String()),
		)
	}

	// Build project basic info with workspace and owner details
	projectInfo := dto.ProjectBasicInfo{
		ProjectID:      project.ID,
		WorkspaceID:    project.WorkspaceID,
		WorkspaceName:  workspace.Name,
		WorkspaceEmail: workspace.OwnerEmail,
		Name:           project.Name,
		Description:    project.Description,
		OwnerID:        project.OwnerID,
		IsPublic:       project.IsPublic,
		StartDate:      project.StartDate,
		DueDate:        project.DueDate,
		CreatedAt:      project.CreatedAt,
		UpdatedAt:      project.UpdatedAt,
	}

	// Add owner profile information if available
	if ownerProfile != nil {
		projectInfo.OwnerEmail = ownerProfile.Email
		projectInfo.OwnerName = ownerProfile.NickName
	}

	// Fetch project-specific field options from database
	stageOptions, err := s.fieldOptionRepo.FindByProjectAndFieldType(ctx, projectID, domain.FieldTypeStage)
	if err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch stage options", err.Error())
	}

	roleOptions, err := s.fieldOptionRepo.FindByProjectAndFieldType(ctx, projectID, domain.FieldTypeRole)
	if err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch role options", err.Error())
	}

	importanceOptions, err := s.fieldOptionRepo.FindByProjectAndFieldType(ctx, projectID, domain.FieldTypeImportance)
	if err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch importance options", err.Error())
	}

	// Convert field options to DTO format
	stageFieldOptions := make([]dto.FieldOption, len(stageOptions))
	for i, opt := range stageOptions {
		stageFieldOptions[i] = dto.FieldOption{
			OptionID:     opt.ID.String(),
			OptionLabel:  opt.Label,
			OptionValue:  opt.Value,
			Color:        opt.Color,
			DisplayOrder: opt.DisplayOrder,
			FieldID:      "stage",
		}
	}

	roleFieldOptions := make([]dto.FieldOption, len(roleOptions))
	for i, opt := range roleOptions {
		roleFieldOptions[i] = dto.FieldOption{
			OptionID:     opt.ID.String(),
			OptionLabel:  opt.Label,
			OptionValue:  opt.Value,
			Color:        opt.Color,
			DisplayOrder: opt.DisplayOrder,
			FieldID:      "role",
		}
	}

	importanceFieldOptions := make([]dto.FieldOption, len(importanceOptions))
	for i, opt := range importanceOptions {
		importanceFieldOptions[i] = dto.FieldOption{
			OptionID:     opt.ID.String(),
			OptionLabel:  opt.Label,
			OptionValue:  opt.Value,
			Color:        opt.Color,
			DisplayOrder: opt.DisplayOrder,
			FieldID:      "importance",
		}
	}

	// Define field definitions with options from database
	fields := []dto.FieldWithOptionsResponse{
		{
			FieldID:     "stage",
			FieldName:   "Stage",
			FieldType:   "select",
			IsRequired:  true,
			Description: "Current stage of the board",
			Options:     stageFieldOptions,
		},
		{
			FieldID:     "importance",
			FieldName:   "Importance",
			FieldType:   "select",
			IsRequired:  true,
			Description: "Priority level of the board",
			Options:     importanceFieldOptions,
		},
		{
			FieldID:     "role",
			FieldName:   "Role",
			FieldType:   "select",
			IsRequired:  true,
			Description: "Role responsible for the board",
			Options:     roleFieldOptions,
		},
	}

	// Define field types
	fieldTypes := []dto.FieldTypeInfo{
		{
			TypeID:      "select",
			TypeName:    "Select",
			Description: "Single selection from predefined options",
		},
		{
			TypeID:      "text",
			TypeName:    "Text",
			Description: "Free text input",
		},
		{
			TypeID:      "date",
			TypeName:    "Date",
			Description: "Date selection",
		},
		{
			TypeID:      "user",
			TypeName:    "User",
			Description: "User selection",
		},
	}

	return &dto.ProjectInitSettingsResponse{
		Project:       projectInfo,
		Fields:        fields,
		FieldTypes:    fieldTypes,
		DefaultViewID: nil, // Can be extended later to support custom views
	}, nil
}

// validateProjectDateRange validates that startDate is not after dueDate
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
