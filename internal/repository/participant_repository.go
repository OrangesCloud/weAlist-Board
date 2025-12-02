package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"project-board-api/internal/domain"
)

// ParticipantRepository defines the interface for participant data access
type ParticipantRepository interface {
	Create(ctx context.Context, participant *domain.Participant) error
	FindByBoardID(ctx context.Context, boardID uuid.UUID) ([]*domain.Participant, error)
	FindByBoardAndUser(ctx context.Context, boardID, userID uuid.UUID) (*domain.Participant, error)
	Delete(ctx context.Context, boardID, userID uuid.UUID) error
}

// participantRepositoryImpl is the GORM implementation of ParticipantRepository
type participantRepositoryImpl struct {
	db *gorm.DB
}

// NewParticipantRepository creates a new instance of ParticipantRepository
func NewParticipantRepository(db *gorm.DB) ParticipantRepository {
	return &participantRepositoryImpl{db: db}
}

// Create creates a new participant
func (r *participantRepositoryImpl) Create(ctx context.Context, participant *domain.Participant) error {
	if err := r.db.WithContext(ctx).Create(participant).Error; err != nil {
		return err
	}
	return nil
}

// FindByBoardID finds all participants by board ID
func (r *participantRepositoryImpl) FindByBoardID(ctx context.Context, boardID uuid.UUID) ([]*domain.Participant, error) {
	var participants []*domain.Participant
	if err := r.db.WithContext(ctx).
		Where("board_id = ?", boardID).
		Find(&participants).Error; err != nil {
		return nil, err
	}
	return participants, nil
}

// FindByBoardAndUser finds a participant by board ID and user ID
func (r *participantRepositoryImpl) FindByBoardAndUser(ctx context.Context, boardID, userID uuid.UUID) (*domain.Participant, error) {
	var participant domain.Participant
	if err := r.db.WithContext(ctx).
		Where("board_id = ? AND user_id = ?", boardID, userID).
		First(&participant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}
	return &participant, nil
}

// Delete soft deletes a participant by board ID and user ID
func (r *participantRepositoryImpl) Delete(ctx context.Context, boardID, userID uuid.UUID) error {
	if err := r.db.WithContext(ctx).
		Where("board_id = ? AND user_id = ?", boardID, userID).
		Delete(&domain.Participant{}).Error; err != nil {
		return err
	}
	return nil
}
