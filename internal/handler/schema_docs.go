package handler

import (
	"project-board-api/internal/dto"
)

// This file exists solely to ensure all DTO schemas are included in Swagger documentation
// even if they are not directly used in current handler endpoints.
// swag will parse these type references and include them in the generated swagger definitions.

// SchemaDocumentation is a placeholder struct that references all DTOs
// This ensures swag includes them in swagger.yaml definitions section
type SchemaDocumentation struct {
	// Board related DTOs
	BoardFilters            dto.BoardFilters            `json:"boardFilters"`
	PaginatedBoardsResponse dto.PaginatedBoardsResponse `json:"paginatedBoardsResponse"`
	UpdateBoardFieldRequest dto.UpdateBoardFieldRequest `json:"updateBoardFieldRequest"`
	AttachmentResponse      dto.AttachmentResponse      `json:"attachmentResponse"`
}

// GetSchemaDocumentation is a dummy handler that will never be called
// It exists only to make swag parse the SchemaDocumentation struct
// @Summary      Schema Documentation (Not a real endpoint)
// @Description  This endpoint does not exist. It's used to document DTO schemas.
// @Tags         internal
// @Produce      json
// @Success      200 {object} SchemaDocumentation
// @Router       /internal/schemas [get]
func GetSchemaDocumentation() {
	// This function is never called - it exists only for swagger documentation
}
