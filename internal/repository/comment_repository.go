package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"project-board-api/internal/domain"
)

// CommentRepository defines the interface for comment data access
type CommentRepository interface {
	Create(ctx context.Context, comment *domain.Comment) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Comment, error)
	FindByBoardID(ctx context.Context, boardID uuid.UUID) ([]*domain.Comment, error)
	Update(ctx context.Context, comment *domain.Comment) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// commentRepositoryImpl is the GORM implementation of CommentRepository
type commentRepositoryImpl struct {
	db *gorm.DB
}

// NewCommentRepository creates a new instance of CommentRepository
func NewCommentRepository(db *gorm.DB) CommentRepository {
	return &commentRepositoryImpl{db: db}
}

// Create creates a new comment
func (r *commentRepositoryImpl) Create(ctx context.Context, comment *domain.Comment) error {
	if err := r.db.WithContext(ctx).Create(comment).Error; err != nil {
		return err
	}
	return nil
}

// FindByID finds a comment by ID
// ✅ 수정: Preload("Attachments") 제거 - service에서 별도 로드
func (r *commentRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*domain.Comment, error) {
	var comment domain.Comment
	if err := r.db.WithContext(ctx).
		// Preload("Attachments"). // ✅ 제거
		Where("id = ?", id).
		First(&comment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}
	return &comment, nil
}

// FindByBoardID finds all comments by board ID, ordered by creation time
// ✅ 수정: Preload("Attachments") 제거 - service에서 별도 로드
func (r *commentRepositoryImpl) FindByBoardID(ctx context.Context, boardID uuid.UUID) ([]*domain.Comment, error) {
	var comments []*domain.Comment
	if err := r.db.WithContext(ctx).
		// Preload("Attachments"). // ✅ 제거
		Where("board_id = ?", boardID).
		Order("created_at ASC").
		Find(&comments).Error; err != nil {
		return nil, err
	}
	return comments, nil
}

// Update updates a comment
func (r *commentRepositoryImpl) Update(ctx context.Context, comment *domain.Comment) error {
	if err := r.db.WithContext(ctx).Save(comment).Error; err != nil {
		return err
	}
	return nil
}

// Delete soft deletes a comment
func (r *commentRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&domain.Comment{}, id).Error; err != nil {
		return err
	}
	return nil
}
