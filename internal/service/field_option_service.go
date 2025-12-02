package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
	"project-board-api/internal/repository"
	"project-board-api/internal/response"
)

// FieldOptionService defines the interface for field option business logic
type FieldOptionService interface {
	GetFieldOptions(ctx context.Context, fieldType domain.FieldType) ([]*dto.FieldOptionResponse, error)
	CreateFieldOption(ctx context.Context, req *dto.CreateFieldOptionRequest) (*dto.FieldOptionResponse, error)
	UpdateFieldOption(ctx context.Context, optionID uuid.UUID, req *dto.UpdateFieldOptionRequest) (*dto.FieldOptionResponse, error)
	DeleteFieldOption(ctx context.Context, optionID uuid.UUID) error
}

// fieldOptionServiceImpl is the implementation of FieldOptionService
type fieldOptionServiceImpl struct {
	fieldOptionRepo repository.FieldOptionRepository
}

// NewFieldOptionService creates a new instance of FieldOptionService
func NewFieldOptionService(fieldOptionRepo repository.FieldOptionRepository) FieldOptionService {
	return &fieldOptionServiceImpl{
		fieldOptionRepo: fieldOptionRepo,
	}
}

// GetFieldOptions retrieves all field options for a specific field type
func (s *fieldOptionServiceImpl) GetFieldOptions(ctx context.Context, fieldType domain.FieldType) ([]*dto.FieldOptionResponse, error) {
	// Validate field type
	if !isValidFieldType(fieldType) {
		return nil, response.NewValidationError(fmt.Sprintf("Invalid field type: %s", fieldType), "")
	}

	// Fetch field options from repository
	fieldOptions, err := s.fieldOptionRepo.FindByFieldType(ctx, fieldType)
	if err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch field options", err.Error())
	}

	// Convert to response DTOs
	responses := make([]*dto.FieldOptionResponse, len(fieldOptions))
	for i, option := range fieldOptions {
		responses[i] = s.toFieldOptionResponse(option)
	}

	return responses, nil
}

// CreateFieldOption creates a new field option with duplicate validation
func (s *fieldOptionServiceImpl) CreateFieldOption(ctx context.Context, req *dto.CreateFieldOptionRequest) (*dto.FieldOptionResponse, error) {
	// Validate field type
	fieldType := domain.FieldType(req.FieldType)
	if !isValidFieldType(fieldType) {
		return nil, response.NewValidationError(fmt.Sprintf("Invalid field type: %s", req.FieldType), "")
	}

	// Check for duplicate value within the same field type
	existingOptions, err := s.fieldOptionRepo.FindByFieldType(ctx, fieldType)
	if err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to check for duplicates", err.Error())
	}

	for _, option := range existingOptions {
		if option.Value == req.Value {
			return nil, response.NewValidationError(fmt.Sprintf("Field option with value '%s' already exists for field type '%s'", req.Value, req.FieldType), "")
		}
	}

	// Create domain model from request
	fieldOption := &domain.FieldOption{
		FieldType:       fieldType,
		Value:           req.Value,
		Label:           req.Label,
		Color:           req.Color,
		DisplayOrder:    req.DisplayOrder,
		IsSystemDefault: false, // User-created options are never system defaults
	}

	// Save to repository
	if err := s.fieldOptionRepo.Create(ctx, fieldOption); err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to create field option", err.Error())
	}

	// Convert to response DTO
	return s.toFieldOptionResponse(fieldOption), nil
}

// UpdateFieldOption updates an existing field option
func (s *fieldOptionServiceImpl) UpdateFieldOption(ctx context.Context, optionID uuid.UUID, req *dto.UpdateFieldOptionRequest) (*dto.FieldOptionResponse, error) {
	// Fetch field option from repository
	fieldOption, err := s.fieldOptionRepo.FindByID(ctx, optionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError("Field option not found", "")
		}
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch field option", err.Error())
	}

	// Update fields if provided
	if req.Label != nil {
		fieldOption.Label = *req.Label
	}
	if req.Color != nil {
		fieldOption.Color = *req.Color
	}
	if req.DisplayOrder != nil {
		fieldOption.DisplayOrder = *req.DisplayOrder
	}

	// Save to repository
	if err := s.fieldOptionRepo.Update(ctx, fieldOption); err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to update field option", err.Error())
	}

	// Convert to response DTO
	return s.toFieldOptionResponse(fieldOption), nil
}

// DeleteFieldOption soft deletes a field option (prevents deletion of system defaults)
func (s *fieldOptionServiceImpl) DeleteFieldOption(ctx context.Context, optionID uuid.UUID) error {
	// Fetch field option from repository
	fieldOption, err := s.fieldOptionRepo.FindByID(ctx, optionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError("Field option not found", "")
		}
		return response.NewAppError(response.ErrCodeInternal, "Failed to fetch field option", err.Error())
	}

	// Prevent deletion of system default options
	if fieldOption.IsSystemDefault {
		return response.NewValidationError("Cannot delete system default field option", "")
	}

	// Delete from repository
	if err := s.fieldOptionRepo.Delete(ctx, optionID); err != nil {
		return response.NewAppError(response.ErrCodeInternal, "Failed to delete field option", err.Error())
	}

	return nil
}

// toFieldOptionResponse converts domain.FieldOption to dto.FieldOptionResponse
func (s *fieldOptionServiceImpl) toFieldOptionResponse(option *domain.FieldOption) *dto.FieldOptionResponse {
	return &dto.FieldOptionResponse{
		OptionID:        option.ID,
		FieldType:       string(option.FieldType),
		Value:           option.Value,
		Label:           option.Label,
		Color:           option.Color,
		DisplayOrder:    option.DisplayOrder,
		IsSystemDefault: option.IsSystemDefault,
		CreatedAt:       option.CreatedAt,
		UpdatedAt:       option.UpdatedAt,
	}
}

// isValidFieldType validates if the field type is one of the allowed types
func isValidFieldType(fieldType domain.FieldType) bool {
	switch fieldType {
	case domain.FieldTypeStage, domain.FieldTypeRole, domain.FieldTypeImportance:
		return true
	default:
		return false
	}
}
