package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"project-board-api/internal/domain"
)

// FieldOptionRepository defines the interface for field option data access
type FieldOptionRepository interface {
	Create(ctx context.Context, fieldOption *domain.FieldOption) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.FieldOption, error)
	FindByFieldType(ctx context.Context, fieldType domain.FieldType) ([]*domain.FieldOption, error)
	FindByProjectAndFieldType(ctx context.Context, projectID uuid.UUID, fieldType domain.FieldType) ([]*domain.FieldOption, error)
	FindByFieldTypeAndValue(ctx context.Context, fieldType, value string) (*domain.FieldOption, error)
	FindSystemDefaults(ctx context.Context) ([]*domain.FieldOption, error)
	CreateBatch(ctx context.Context, fieldOptions []*domain.FieldOption) error
	Update(ctx context.Context, fieldOption *domain.FieldOption) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.FieldOption, error)
	FindByProjectAndFieldTypeAndValue(ctx context.Context, projectID uuid.UUID, fieldType domain.FieldType, value string) (*domain.FieldOption, error)
}

// fieldOptionRepositoryImpl is the GORM implementation of FieldOptionRepository
type fieldOptionRepositoryImpl struct {
	db *gorm.DB
}

// NewFieldOptionRepository creates a new instance of FieldOptionRepository
func NewFieldOptionRepository(db *gorm.DB) FieldOptionRepository {
	return &fieldOptionRepositoryImpl{db: db}
}

// Create creates a new field option
func (r *fieldOptionRepositoryImpl) Create(ctx context.Context, fieldOption *domain.FieldOption) error {
	if err := r.db.WithContext(ctx).Create(fieldOption).Error; err != nil {
		return err
	}
	return nil
}

// FindByID finds a field option by ID
func (r *fieldOptionRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*domain.FieldOption, error) {
	var fieldOption domain.FieldOption
	if err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&fieldOption).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}
	return &fieldOption, nil
}

// FindByFieldType finds all field options by field type, ordered by display_order
func (r *fieldOptionRepositoryImpl) FindByFieldType(ctx context.Context, fieldType domain.FieldType) ([]*domain.FieldOption, error) {
	var fieldOptions []*domain.FieldOption
	if err := r.db.WithContext(ctx).
		Where("field_type = ?", fieldType).
		Order("display_order ASC").
		Find(&fieldOptions).Error; err != nil {
		return nil, err
	}
	return fieldOptions, nil
}

// FindByFieldTypeAndValue finds a field option by field type and value
func (r *fieldOptionRepositoryImpl) FindByFieldTypeAndValue(ctx context.Context, fieldType, value string) (*domain.FieldOption, error) {
	var fieldOption domain.FieldOption
	if err := r.db.WithContext(ctx).
		Where("field_type = ? AND value = ?", fieldType, value).
		First(&fieldOption).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &fieldOption, nil
}

// Update updates a field option
func (r *fieldOptionRepositoryImpl) Update(ctx context.Context, fieldOption *domain.FieldOption) error {
	if err := r.db.WithContext(ctx).Save(fieldOption).Error; err != nil {
		return err
	}
	return nil
}

// Delete soft deletes a field option
func (r *fieldOptionRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&domain.FieldOption{}, id).Error; err != nil {
		return err
	}
	return nil
}

// FindByProjectAndFieldType finds all field options for a specific project and field type
func (r *fieldOptionRepositoryImpl) FindByProjectAndFieldType(ctx context.Context, projectID uuid.UUID, fieldType domain.FieldType) ([]*domain.FieldOption, error) {
	var fieldOptions []*domain.FieldOption
	if err := r.db.WithContext(ctx).
		Where("project_id = ? AND field_type = ?", projectID, fieldType).
		Order("display_order ASC").
		Find(&fieldOptions).Error; err != nil {
		return nil, err
	}
	return fieldOptions, nil
}

// FindSystemDefaults finds all system default field options (project_id is NULL)
func (r *fieldOptionRepositoryImpl) FindSystemDefaults(ctx context.Context) ([]*domain.FieldOption, error) {
	var fieldOptions []*domain.FieldOption
	if err := r.db.WithContext(ctx).
		Where("project_id IS NULL AND is_system_default = ?", true).
		Order("field_type ASC, display_order ASC").
		Find(&fieldOptions).Error; err != nil {
		return nil, err
	}
	return fieldOptions, nil
}

// CreateBatch creates multiple field options in a single transaction
func (r *fieldOptionRepositoryImpl) CreateBatch(ctx context.Context, fieldOptions []*domain.FieldOption) error {
	if len(fieldOptions) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).Create(&fieldOptions).Error; err != nil {
		return err
	}
	return nil
}

// FindByIDs finds multiple field options by their IDs in a single query
func (r *fieldOptionRepositoryImpl) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.FieldOption, error) {
	if len(ids) == 0 {
		return []*domain.FieldOption{}, nil
	}
	
	var fieldOptions []*domain.FieldOption
	if err := r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&fieldOptions).Error; err != nil {
		return nil, err
	}
	return fieldOptions, nil
}

// FindByProjectAndFieldTypeAndValue finds a field option by project ID, field type, and value
func (r *fieldOptionRepositoryImpl) FindByProjectAndFieldTypeAndValue(ctx context.Context, projectID uuid.UUID, fieldType domain.FieldType, value string) (*domain.FieldOption, error) {
	var fieldOption domain.FieldOption
	if err := r.db.WithContext(ctx).
		Where("project_id = ? AND field_type = ? AND value = ?", projectID, fieldType, value).
		First(&fieldOption).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &fieldOption, nil
}
