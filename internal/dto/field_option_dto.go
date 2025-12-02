package dto

import (
	"time"

	"github.com/google/uuid"
)

// FieldOptionResponse represents the field option response
type FieldOptionResponse struct {
	OptionID        uuid.UUID `json:"optionId"`
	FieldType       string    `json:"fieldType"`
	Value           string    `json:"value"`
	Label           string    `json:"label"`
	Color           string    `json:"color"`
	DisplayOrder    int       `json:"displayOrder"`
	IsSystemDefault bool      `json:"isSystemDefault"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// CreateFieldOptionRequest represents the request to create a new field option
type CreateFieldOptionRequest struct {
	FieldType    string `json:"fieldType" binding:"required,oneof=stage role importance"`
	Value        string `json:"value" binding:"required,max=100"`
	Label        string `json:"label" binding:"required,max=200"`
	Color        string `json:"color" binding:"required,hexcolor"`
	DisplayOrder int    `json:"displayOrder"`
}

// UpdateFieldOptionRequest represents the request to update a field option
type UpdateFieldOptionRequest struct {
	Label        *string `json:"label" binding:"omitempty,max=200"`
	Color        *string `json:"color" binding:"omitempty,hexcolor"`
	DisplayOrder *int    `json:"displayOrder"`
}
