package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"project-board-api/internal/domain"
)

// ProjectRepository defines the interface for project data access
type ProjectRepository interface {
	Create(ctx context.Context, project *domain.Project) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Project, error)
	FindByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) ([]*domain.Project, error)
	FindDefaultByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) (*domain.Project, error)
	Search(ctx context.Context, workspaceID uuid.UUID, query string, page, limit int) ([]*domain.Project, int64, error)
	Update(ctx context.Context, project *domain.Project) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Member management
	AddMember(ctx context.Context, member *domain.ProjectMember) error
	FindMembersByProjectID(ctx context.Context, projectID uuid.UUID) ([]*domain.ProjectMember, error)
	FindMemberByProjectAndUser(ctx context.Context, projectID, userID uuid.UUID) (*domain.ProjectMember, error)
	UpdateMemberRole(ctx context.Context, memberID uuid.UUID, role domain.ProjectRole) error
	RemoveMember(ctx context.Context, memberID uuid.UUID) error
	IsProjectMember(ctx context.Context, projectID, userID uuid.UUID) (bool, error)

	// Join request management
	CreateJoinRequest(ctx context.Context, request *domain.ProjectJoinRequest) error
	FindJoinRequestsByProjectID(ctx context.Context, projectID uuid.UUID, status *domain.ProjectJoinRequestStatus) ([]*domain.ProjectJoinRequest, error)
	FindJoinRequestByID(ctx context.Context, requestID uuid.UUID) (*domain.ProjectJoinRequest, error)
	FindPendingByProjectAndUser(ctx context.Context, projectID, userID uuid.UUID) (*domain.ProjectJoinRequest, error)
	UpdateJoinRequestStatus(ctx context.Context, requestID uuid.UUID, status domain.ProjectJoinRequestStatus) error
}

// projectRepositoryImpl is the GORM implementation of ProjectRepository
type projectRepositoryImpl struct {
	db *gorm.DB
}

// NewProjectRepository creates a new instance of ProjectRepository
func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &projectRepositoryImpl{db: db}
}

// Create creates a new project
func (r *projectRepositoryImpl) Create(ctx context.Context, project *domain.Project) error {
	if err := r.db.WithContext(ctx).Create(project).Error; err != nil {
		return err
	}
	return nil
}

// FindByID finds a project by ID
// ✅ 수정: Preload("Attachments") 제거 - service에서 별도 로드
func (r *projectRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
	var project domain.Project
	if err := r.db.WithContext(ctx).
		// Preload("Attachments"). // ✅ 제거
		Where("id = ?", id).
		First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}
	return &project, nil
}

// FindByWorkspaceID finds all projects by workspace ID
// ✅ 수정: Preload("Attachments") 제거 - service에서 별도 로드
func (r *projectRepositoryImpl) FindByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) ([]*domain.Project, error) {
	// Explicitly initialize empty array to prevent nil return
	projects := make([]*domain.Project, 0)
	if err := r.db.WithContext(ctx).
		// Preload("Attachments"). // ✅ 제거
		Where("workspace_id = ?", workspaceID).
		Find(&projects).Error; err != nil {
		return nil, err
	}
	return projects, nil
}

// FindDefaultByWorkspaceID finds the default project by workspace ID
func (r *projectRepositoryImpl) FindDefaultByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) (*domain.Project, error) {
	var project domain.Project
	if err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND is_default = ?", workspaceID, true).
		First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}
	return &project, nil
}

// Update updates a project
func (r *projectRepositoryImpl) Update(ctx context.Context, project *domain.Project) error {
	if err := r.db.WithContext(ctx).Save(project).Error; err != nil {
		return err
	}
	return nil
}

// Delete soft deletes a project
func (r *projectRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&domain.Project{}, id).Error; err != nil {
		return err
	}
	return nil
}

// Search searches projects by name or description
func (r *projectRepositoryImpl) Search(ctx context.Context, workspaceID uuid.UUID, query string, page, limit int) ([]*domain.Project, int64, error) {
	var projects []*domain.Project
	var total int64

	db := r.db.WithContext(ctx).Where("workspace_id = ?", workspaceID)

	if query != "" {
		searchPattern := "%" + query + "%"
		db = db.Where("name ILIKE ? OR description ILIKE ?", searchPattern, searchPattern)
	}

	// Count total
	if err := db.Model(&domain.Project{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	offset := (page - 1) * limit
	if err := db.Offset(offset).Limit(limit).Find(&projects).Error; err != nil {
		return nil, 0, err
	}

	return projects, total, nil
}

// AddMember adds a member to a project
func (r *projectRepositoryImpl) AddMember(ctx context.Context, member *domain.ProjectMember) error {
	return r.db.WithContext(ctx).Create(member).Error
}

// FindMembersByProjectID finds all members of a project
func (r *projectRepositoryImpl) FindMembersByProjectID(ctx context.Context, projectID uuid.UUID) ([]*domain.ProjectMember, error) {
	var members []*domain.ProjectMember
	if err := r.db.WithContext(ctx).Where("project_id = ?", projectID).Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
}

// FindMemberByProjectAndUser finds a specific member by project and user ID
func (r *projectRepositoryImpl) FindMemberByProjectAndUser(ctx context.Context, projectID, userID uuid.UUID) (*domain.ProjectMember, error) {
	var member domain.ProjectMember
	if err := r.db.WithContext(ctx).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}
	return &member, nil
}

// UpdateMemberRole updates a member's role
func (r *projectRepositoryImpl) UpdateMemberRole(ctx context.Context, memberID uuid.UUID, role domain.ProjectRole) error {
	return r.db.WithContext(ctx).
		Model(&domain.ProjectMember{}).
		Where("id = ?", memberID).
		Update("role_name", role).Error
}

// RemoveMember removes a member from a project
func (r *projectRepositoryImpl) RemoveMember(ctx context.Context, memberID uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.ProjectMember{}, memberID).Error
}

// IsProjectMember checks if a user is a member of a project
func (r *projectRepositoryImpl) IsProjectMember(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&domain.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// CreateJoinRequest creates a new join request
func (r *projectRepositoryImpl) CreateJoinRequest(ctx context.Context, request *domain.ProjectJoinRequest) error {
	return r.db.WithContext(ctx).Create(request).Error
}

// FindJoinRequestsByProjectID finds all join requests for a project
func (r *projectRepositoryImpl) FindJoinRequestsByProjectID(ctx context.Context, projectID uuid.UUID, status *domain.ProjectJoinRequestStatus) ([]*domain.ProjectJoinRequest, error) {
	var requests []*domain.ProjectJoinRequest
	db := r.db.WithContext(ctx).Where("project_id = ?", projectID)

	if status != nil {
		db = db.Where("status = ?", *status)
	}

	if err := db.Find(&requests).Error; err != nil {
		return nil, err
	}
	return requests, nil
}

// FindJoinRequestByID finds a join request by ID
func (r *projectRepositoryImpl) FindJoinRequestByID(ctx context.Context, requestID uuid.UUID) (*domain.ProjectJoinRequest, error) {
	var request domain.ProjectJoinRequest
	if err := r.db.WithContext(ctx).Where("id = ?", requestID).First(&request).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}
	return &request, nil
}

// FindPendingByProjectAndUser finds a pending join request for a specific user and project
func (r *projectRepositoryImpl) FindPendingByProjectAndUser(ctx context.Context, projectID, userID uuid.UUID) (*domain.ProjectJoinRequest, error) {
	var request domain.ProjectJoinRequest
	if err := r.db.WithContext(ctx).
		Where("project_id = ? AND user_id = ? AND status = ?", projectID, userID, domain.JoinRequestPending).
		First(&request).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}
	return &request, nil
}

// UpdateJoinRequestStatus updates the status of a join request
func (r *projectRepositoryImpl) UpdateJoinRequestStatus(ctx context.Context, requestID uuid.UUID, status domain.ProjectJoinRequestStatus) error {
	return r.db.WithContext(ctx).
		Model(&domain.ProjectJoinRequest{}).
		Where("id = ?", requestID).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": gorm.Expr("NOW()"),
		}).Error
}
