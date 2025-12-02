package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"project-board-api/internal/domain"
)

// AttachmentRepository defines the interface for attachment data access
type AttachmentRepository interface {
	Create(ctx context.Context, attachment *domain.Attachment) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Attachment, error)
	FindByEntityID(ctx context.Context, entityType domain.EntityType, entityID uuid.UUID) ([]*domain.Attachment, error)
	FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.Attachment, error)
	Delete(ctx context.Context, id uuid.UUID) error
	FindExpiredTempAttachments(ctx context.Context) ([]*domain.Attachment, error)
	ConfirmAttachments(ctx context.Context, attachmentIDs []uuid.UUID, entityID uuid.UUID) error
	DeleteBatch(ctx context.Context, attachmentIDs []uuid.UUID) error
}

// attachmentRepositoryImpl is the GORM implementation of AttachmentRepository
type attachmentRepositoryImpl struct {
	db *gorm.DB
}

// NewAttachmentRepository creates a new instance of AttachmentRepository
func NewAttachmentRepository(db *gorm.DB) AttachmentRepository {
	return &attachmentRepositoryImpl{db: db}
}

// Create creates a new attachment
func (r *attachmentRepositoryImpl) Create(ctx context.Context, attachment *domain.Attachment) error {
	if err := r.db.WithContext(ctx).Create(attachment).Error; err != nil {
		return err
	}
	return nil
}

// FindByID finds an attachment by its ID
func (r *attachmentRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*domain.Attachment, error) {
	var attachment domain.Attachment
	if err := r.db.WithContext(ctx).First(&attachment, id).Error; err != nil {
		return nil, err
	}
	return &attachment, nil
}

// FindByEntityID finds all attachments by entity type and entity ID
func (r *attachmentRepositoryImpl) FindByEntityID(ctx context.Context, entityType domain.EntityType, entityID uuid.UUID) ([]*domain.Attachment, error) {
	var attachments []*domain.Attachment
	if err := r.db.WithContext(ctx).
		Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Order("created_at DESC").
		Find(&attachments).Error; err != nil {
		return nil, err
	}
	return attachments, nil
}

// FindByIDs finds attachments by their IDs
func (r *attachmentRepositoryImpl) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.Attachment, error) {
	if len(ids) == 0 {
		return []*domain.Attachment{}, nil
	}

	var attachments []*domain.Attachment
	if err := r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&attachments).Error; err != nil {
		return nil, err
	}
	return attachments, nil
}

// Delete soft deletes an attachment by ID
func (r *attachmentRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&domain.Attachment{}, id).Error; err != nil {
		return err
	}
	return nil
}

// FindExpiredTempAttachments finds all temporary attachments that have exceeded their expiration time
func (r *attachmentRepositoryImpl) FindExpiredTempAttachments(ctx context.Context) ([]*domain.Attachment, error) {
	var attachments []*domain.Attachment
	if err := r.db.WithContext(ctx).
		Where("status = ? AND expires_at < ?", domain.AttachmentStatusTemp, time.Now()).
		Find(&attachments).Error; err != nil {
		return nil, err
	}
	return attachments, nil
}

func (r *attachmentRepositoryImpl) ConfirmAttachments(ctx context.Context, attachmentIDs []uuid.UUID, entityID uuid.UUID) error {
	if len(attachmentIDs) == 0 {
		return nil
	}

	// ✅ TEMP 상태만 업데이트, 결과 검증
	result := r.db.WithContext(ctx).
		Model(&domain.Attachment{}).
		Where("id IN ? AND status = ?", attachmentIDs, domain.AttachmentStatusTemp). // ✅
		Updates(map[string]interface{}{
			"status":    domain.AttachmentStatusConfirmed,
			"entity_id": entityID,
		})

	if result.Error != nil {
		return result.Error
	}

	// ✅ 업데이트된 행 수 검증
	if result.RowsAffected == 0 {
		return fmt.Errorf("no attachments were confirmed: all %d attachment(s) are either not found or already confirmed",
			len(attachmentIDs))
	}

	if result.RowsAffected != int64(len(attachmentIDs)) {
		return fmt.Errorf("expected to confirm %d attachment(s) but only confirmed %d",
			len(attachmentIDs), result.RowsAffected)
	}

	return nil
}

// DeleteBatch deletes multiple attachments by their IDs
func (r *attachmentRepositoryImpl) DeleteBatch(ctx context.Context, attachmentIDs []uuid.UUID) error {
	if len(attachmentIDs) == 0 {
		return nil
	}

	if err := r.db.WithContext(ctx).
		Where("id IN ?", attachmentIDs).
		Delete(&domain.Attachment{}).Error; err != nil {
		return err
	}
	return nil
}
