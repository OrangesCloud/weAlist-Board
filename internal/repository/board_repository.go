package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"project-board-api/internal/domain"
)

// BoardRepository defines the interface for board data access
type BoardRepository interface {
	Create(ctx context.Context, board *domain.Board) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Board, error)
	FindByProjectID(ctx context.Context, projectID uuid.UUID, filters interface{}) ([]*domain.Board, error)
	Update(ctx context.Context, board *domain.Board) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// boardRepositoryImpl is the GORM implementation of BoardRepository
type boardRepositoryImpl struct {
	db *gorm.DB
}

// NewBoardRepository creates a new instance of BoardRepository
func NewBoardRepository(db *gorm.DB) BoardRepository {
	return &boardRepositoryImpl{db: db}
}

// Create creates a new board
func (r *boardRepositoryImpl) Create(ctx context.Context, board *domain.Board) error {
	if err := r.db.WithContext(ctx).Create(board).Error; err != nil {
		return err
	}
	return nil
}

// FindByID finds a board by ID with preloaded participants and comments
// ✅ 수정: Preload("Attachments") 제거 - service에서 별도 로드
func (r *boardRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
	var board domain.Board
	if err := r.db.WithContext(ctx).
		Preload("Participants").
		Preload("Comments").
		// Preload("Attachments"). // ✅ 제거
		Where("id = ?", id).
		First(&board).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}
	return &board, nil
}

// FindByProjectID finds all boards by project ID with optional filters
// ✅ 수정: Preload("Attachments") 제거 - service에서 별도 로드
func (r *boardRepositoryImpl) FindByProjectID(ctx context.Context, projectID uuid.UUID, filters interface{}) ([]*domain.Board, error) {
	var boards []*domain.Board

	// Start building the query with Participants preload
	query := r.db.WithContext(ctx).
		Preload("Participants").
		// Preload("Attachments"). // ✅ 제거
		Where("project_id = ?", projectID)

	// Apply filters if provided
	if filters != nil {
		// Type assertion to get customFields map
		if customFields, ok := filters.(map[string]interface{}); ok {
			// Apply JSONB filtering for each custom field
			for key, value := range customFields {
				// Use JSONB operator ->> to extract text value and compare
				query = query.Where("custom_fields->>? = ?", key, value)
			}
		}
	}

	// Execute the query
	if err := query.Find(&boards).Error; err != nil {
		return nil, err
	}

	return boards, nil
}

// Update updates a board
func (r *boardRepositoryImpl) Update(ctx context.Context, board *domain.Board) error {
	if err := r.db.WithContext(ctx).Save(board).Error; err != nil {
		return err
	}
	return nil
}

// Delete soft deletes a board
func (r *boardRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&domain.Board{}, id).Error; err != nil {
		return err
	}
	return nil
}
