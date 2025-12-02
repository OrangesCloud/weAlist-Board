package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"project-board-api/internal/domain"
)

// ProjectMemberRepository defines the interface for project member data access
type ProjectMemberRepository interface {
	Create(ctx context.Context, member *domain.ProjectMember) error
	FindByProjectID(ctx context.Context, projectID uuid.UUID) ([]*domain.ProjectMember, error)
	FindByProjectAndUser(ctx context.Context, projectID, userID uuid.UUID) (*domain.ProjectMember, error)
	UpdateRole(ctx context.Context, projectID, userID uuid.UUID, role domain.ProjectRole) error
	Delete(ctx context.Context, projectID, userID uuid.UUID) error
	IsProjectMember(ctx context.Context, projectID, userID uuid.UUID) (bool, error)
}

// projectMemberRepositoryImpl is the GORM implementation of ProjectMemberRepository
type projectMemberRepositoryImpl struct {
	db *gorm.DB
}

// NewProjectMemberRepository creates a new instance of ProjectMemberRepository
func NewProjectMemberRepository(db *gorm.DB) ProjectMemberRepository {
	return &projectMemberRepositoryImpl{db: db}
}

// Create creates a new project member
func (r *projectMemberRepositoryImpl) Create(ctx context.Context, member *domain.ProjectMember) error {
	return r.db.WithContext(ctx).Create(member).Error
}

// FindByProjectID finds all members of a project
func (r *projectMemberRepositoryImpl) FindByProjectID(ctx context.Context, projectID uuid.UUID) ([]*domain.ProjectMember, error) {
	var members []*domain.ProjectMember
	if err := r.db.WithContext(ctx).Where("project_id = ?", projectID).Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
}

// FindByProjectAndUser finds a specific member by project and user ID
func (r *projectMemberRepositoryImpl) FindByProjectAndUser(ctx context.Context, projectID, userID uuid.UUID) (*domain.ProjectMember, error) {
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

// UpdateRole updates a member's role
func (r *projectMemberRepositoryImpl) UpdateRole(ctx context.Context, projectID, userID uuid.UUID, role domain.ProjectRole) error {
	return r.db.WithContext(ctx).
		Model(&domain.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Update("role_name", role).Error
}

// Delete removes a member from a project
func (r *projectMemberRepositoryImpl) Delete(ctx context.Context, projectID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Delete(&domain.ProjectMember{}).Error
}

// IsProjectMember checks if a user is a member of a project
func (r *projectMemberRepositoryImpl) IsProjectMember(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&domain.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
