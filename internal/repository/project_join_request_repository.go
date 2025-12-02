package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"project-board-api/internal/domain"
)

// ProjectJoinRequestRepository defines the interface for project join request data access
type ProjectJoinRequestRepository interface {
	Create(ctx context.Context, request *domain.ProjectJoinRequest) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.ProjectJoinRequest, error)
	FindByProjectID(ctx context.Context, projectID uuid.UUID, status *domain.ProjectJoinRequestStatus) ([]*domain.ProjectJoinRequest, error)
	FindPendingByProjectAndUser(ctx context.Context, projectID, userID uuid.UUID) (*domain.ProjectJoinRequest, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ProjectJoinRequestStatus) error
}

// projectJoinRequestRepositoryImpl is the GORM implementation of ProjectJoinRequestRepository
type projectJoinRequestRepositoryImpl struct {
	db *gorm.DB
}

// NewProjectJoinRequestRepository creates a new instance of ProjectJoinRequestRepository
func NewProjectJoinRequestRepository(db *gorm.DB) ProjectJoinRequestRepository {
	return &projectJoinRequestRepositoryImpl{db: db}
}

// Create creates a new project join request
func (r *projectJoinRequestRepositoryImpl) Create(ctx context.Context, request *domain.ProjectJoinRequest) error {
	return r.db.WithContext(ctx).Create(request).Error
}

// FindByID finds a join request by its ID
func (r *projectJoinRequestRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*domain.ProjectJoinRequest, error) {
	var request domain.ProjectJoinRequest
	if err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&request).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}
	return &request, nil
}

// FindByProjectID finds all join requests for a project, optionally filtered by status
func (r *projectJoinRequestRepositoryImpl) FindByProjectID(ctx context.Context, projectID uuid.UUID, status *domain.ProjectJoinRequestStatus) ([]*domain.ProjectJoinRequest, error) {
	var requests []*domain.ProjectJoinRequest
	query := r.db.WithContext(ctx).Where("project_id = ?", projectID)
	
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	
	if err := query.Order("requested_at DESC").Find(&requests).Error; err != nil {
		return nil, err
	}
	return requests, nil
}

// FindPendingByProjectAndUser finds a pending join request for a specific user and project
func (r *projectJoinRequestRepositoryImpl) FindPendingByProjectAndUser(ctx context.Context, projectID, userID uuid.UUID) (*domain.ProjectJoinRequest, error) {
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

// UpdateStatus updates the status of a join request
func (r *projectJoinRequestRepositoryImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ProjectJoinRequestStatus) error {
	return r.db.WithContext(ctx).
		Model(&domain.ProjectJoinRequest{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": gorm.Expr("NOW()"),
		}).Error
}
