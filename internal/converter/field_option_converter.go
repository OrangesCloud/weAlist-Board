package converter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"project-board-api/internal/domain"
	"project-board-api/internal/repository"
)

// FieldOptionConverter handles conversion between field option values and IDs
type FieldOptionConverter interface {
	// ConvertValuesToIDs converts customFields from value strings to UUIDs
	// Input: {"importance": "high", "stage": "in_progress"}
	// Output: {"importance": "uuid-1", "stage": "uuid-2"}
	ConvertValuesToIDs(ctx context.Context, projectID uuid.UUID, customFields map[string]interface{}) (map[string]interface{}, error)

	// ConvertIDsToValues converts customFields from UUIDs to value strings
	// Input: {"importance": "uuid-1", "stage": "uuid-2"}
	// Output: {"importance": "high", "stage": "in_progress"}
	ConvertIDsToValues(ctx context.Context, customFields map[string]interface{}) (map[string]interface{}, error)

	// ConvertIDsToValuesBatch converts customFields for multiple boards efficiently
	ConvertIDsToValuesBatch(ctx context.Context, boards []*domain.Board) error
}

// fieldOptionConverterImpl is the implementation of FieldOptionConverter
type fieldOptionConverterImpl struct {
	fieldOptionRepo repository.FieldOptionRepository
}

// NewFieldOptionConverter creates a new instance of FieldOptionConverter
func NewFieldOptionConverter(fieldOptionRepo repository.FieldOptionRepository) FieldOptionConverter {
	return &fieldOptionConverterImpl{
		fieldOptionRepo: fieldOptionRepo,
	}
}

// ConvertValuesToIDs converts customFields from value strings to UUIDs
func (c *fieldOptionConverterImpl) ConvertValuesToIDs(
	ctx context.Context,
	projectID uuid.UUID,
	customFields map[string]interface{},
) (map[string]interface{}, error) {
	if customFields == nil || len(customFields) == 0 {
		return customFields, nil
	}

	result := make(map[string]interface{})

	for fieldType, value := range customFields {
		valueStr, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("invalid value type for field '%s': expected string, got %T", fieldType, value)
		}

		// Query field option by project, field type, and value
		option, err := c.fieldOptionRepo.FindByProjectAndFieldTypeAndValue(
			ctx,
			projectID,
			domain.FieldType(fieldType),
			valueStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to find field option for field '%s': %w", fieldType, err)
		}
		if option == nil {
			return nil, fmt.Errorf("invalid field option value '%s' for field type '%s'", valueStr, fieldType)
		}

		result[fieldType] = option.ID.String()
	}

	return result, nil
}

// ConvertIDsToValues converts customFields from UUIDs to value strings
func (c *fieldOptionConverterImpl) ConvertIDsToValues(
	ctx context.Context,
	customFields map[string]interface{},
) (map[string]interface{}, error) {
	if customFields == nil || len(customFields) == 0 {
		return customFields, nil
	}

	result := make(map[string]interface{})

	// Collect all UUIDs
	var ids []uuid.UUID
	idToFieldType := make(map[string]string)

	for fieldType, value := range customFields {
		idStr, ok := value.(string)
		if !ok {
			// If not a string, keep as is
			result[fieldType] = value
			continue
		}

		id, err := uuid.Parse(idStr)
		if err != nil {
			// If not a valid UUID, keep as is
			result[fieldType] = value
			continue
		}

		ids = append(ids, id)
		idToFieldType[id.String()] = fieldType
	}

	if len(ids) == 0 {
		return customFields, nil
	}

	// Batch query: SELECT * FROM field_options WHERE id IN (...)
	options, err := c.fieldOptionRepo.FindByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to find field options by IDs: %w", err)
	}

	// Create ID → value mapping
	idToValue := make(map[string]string)
	for _, option := range options {
		idToValue[option.ID.String()] = option.Value
	}

	// Convert
	for idStr, fieldType := range idToFieldType {
		if val, exists := idToValue[idStr]; exists {
			result[fieldType] = val
		} else {
			// Field option not found, return empty string
			result[fieldType] = ""
		}
	}

	return result, nil
}

// ConvertIDsToValuesBatch converts customFields for multiple boards efficiently
func (c *fieldOptionConverterImpl) ConvertIDsToValuesBatch(
	ctx context.Context,
	boards []*domain.Board,
) error {
	if len(boards) == 0 {
		return nil
	}

	// Collect all unique field option IDs from all boards
	idSet := make(map[uuid.UUID]bool)

	for _, board := range boards {
		if board.CustomFields == nil {
			continue
		}

		var customFields map[string]interface{}
		if err := json.Unmarshal(board.CustomFields, &customFields); err != nil {
			// Skip boards with invalid JSON
			continue
		}

		for _, value := range customFields {
			idStr, ok := value.(string)
			if !ok {
				continue
			}

			id, err := uuid.Parse(idStr)
			if err != nil {
				continue
			}

			idSet[id] = true
		}
	}

	if len(idSet) == 0 {
		return nil
	}

	// Convert set to slice
	ids := make([]uuid.UUID, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	// Single batch query
	options, err := c.fieldOptionRepo.FindByIDs(ctx, ids)
	if err != nil {
		return fmt.Errorf("failed to find field options by IDs: %w", err)
	}

	// Create ID → value mapping
	idToValue := make(map[string]string)
	for _, option := range options {
		idToValue[option.ID.String()] = option.Value
	}

	// Convert each board's customFields
	for _, board := range boards {
		if board.CustomFields == nil {
			continue
		}

		var customFields map[string]interface{}
		if err := json.Unmarshal(board.CustomFields, &customFields); err != nil {
			continue
		}

		converted := make(map[string]interface{})
		for fieldType, value := range customFields {
			idStr, ok := value.(string)
			if !ok {
				converted[fieldType] = value
				continue
			}

			if val, exists := idToValue[idStr]; exists {
				converted[fieldType] = val
			} else {
				converted[fieldType] = ""
			}
		}

		// Update board's customFields in memory
		jsonBytes, err := json.Marshal(converted)
		if err != nil {
			return fmt.Errorf("failed to marshal converted customFields: %w", err)
		}
		board.CustomFields = jsonBytes
	}

	return nil
}
