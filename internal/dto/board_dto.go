package dto

import (
	"time"

	"github.com/google/uuid"
)

// CreateBoardRequest represents the request to create a new board
// @Description Request body for creating a new board with value-based customFields
// @Description customFields should contain field type as key and value string as value
// @Description Valid field types: stage, role, importance
// @Description Example values: stage="in_progress", role="developer", importance="high"
// @Description participants is an optional array of user IDs to add as board participants (max 50)
// @Description attachmentIds is an optional array of attachment IDs to link to the board
type CreateBoardRequest struct {
	ProjectID     uuid.UUID              `json:"projectId" binding:"required" example:"539167fb-b599-41ba-9ead-344a6d0b3a2f"`
	Title         string                 `json:"title" binding:"required,min=1,max=200" example:"Implement user authentication"`
	Content       string                 `json:"content" binding:"max=5000" example:"Add JWT-based authentication to the API"`
	CustomFields  map[string]interface{} `json:"customFields" swaggertype:"object,string" example:"importance:high"`
	AssigneeID    *uuid.UUID             `json:"assigneeId" example:"a1b2c3d4-e5f6-7890-abcd-ef1234567890"`
	StartDate     *time.Time             `json:"startDate" example:"2024-01-01T00:00:00Z"`
	DueDate       *time.Time             `json:"dueDate" example:"2024-12-31T23:59:59Z"`
	Participants  []uuid.UUID            `json:"participants,omitempty" binding:"omitempty,max=50,dive,uuid" example:"a1b2c3d4-e5f6-7890-abcd-ef1234567890,b2c3d4e5-f6a7-8901-bcde-f12345678901"`
	AttachmentIDs []uuid.UUID            `json:"attachmentIds,omitempty" binding:"omitempty,dive,uuid" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"`
}

// UpdateBoardRequest represents the request to update a board
// @Description Request body for updating a board with value-based customFields
// @Description customFields should contain field type as key and value string as value
// @Description Valid field types: stage, role, importance
// @Description Example values: stage="completed", role="designer", importance="medium"
// @Description attachmentIds is an optional array of attachment IDs to add to the board
type UpdateBoardRequest struct {
	Title         *string                 `json:"title" binding:"omitempty,min=1,max=200" example:"Update user authentication"`
	Content       *string                 `json:"content" binding:"omitempty,max=5000" example:"Refactor JWT implementation"`
	CustomFields  *map[string]interface{} `json:"customFields" swaggertype:"object,string" example:"importance:medium"`
	AssigneeID    *uuid.UUID              `json:"assigneeId" example:"a1b2c3d4-e5f6-7890-abcd-ef1234567890"`
	StartDate     *time.Time              `json:"startDate" example:"2024-01-01T00:00:00Z"`
	DueDate       *time.Time              `json:"dueDate" example:"2024-12-31T23:59:59Z"`
	Participants  []uuid.UUID             `json:"participants,omitempty" binding:"omitempty,max=50,dive,uuid"`
	AttachmentIDs []uuid.UUID             `json:"attachmentIds,omitempty" binding:"omitempty,dive,uuid" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"`
}

// UpdateBoardFieldRequest represents the request to update a single board field
type UpdateBoardFieldRequest struct {
	FieldID string `json:"fieldId" binding:"required,oneof=stage importance role"`
	Value   string `json:"value" binding:"required"`
}

// AttachmentResponse represents file attachment metadata
// @Description File attachment metadata for boards and projects
// @Description Contains information about uploaded files including S3 URL, size, and content type
type AttachmentResponse struct {
	ID          uuid.UUID `json:"id" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"`
	FileName    string    `json:"fileName" example:"document.pdf"`
	FileURL     string    `json:"fileUrl" example:"https://s3.amazonaws.com/bucket/file.pdf"`
	FileSize    int64     `json:"fileSize" example:"1024000"`
	ContentType string    `json:"contentType" example:"application/pdf"`
	UploadedBy  uuid.UUID `json:"uploadedBy" example:"a1b2c3d4-e5f6-7890-abcd-ef1234567890"`
	UploadedAt  time.Time `json:"uploadedAt" example:"2024-01-15T10:30:00Z"`
}

// BoardResponse represents the board response
// @Description Board response with value-based customFields and participant IDs
// @Description customFields contains field type as key and value string as value (not UUIDs)
// @Description Example: {"importance": "high", "role": "developer", "stage": "in_progress"}
// @Description participantIds contains an array of user IDs who are participants of the board
type BoardResponse struct {
	ID             uuid.UUID              `json:"boardId" example:"1275eac5-f0f9-4bee-8235-576a0042f42b"`
	ProjectID      uuid.UUID              `json:"projectId" example:"539167fb-b599-41ba-9ead-344a6d0b3a2f"`
	AuthorID       uuid.UUID              `json:"authorId" example:"b2c3d4e5-f6a7-8901-bcde-f12345678901"`
	AssigneeID     *uuid.UUID             `json:"assigneeId" example:"a1b2c3d4-e5f6-7890-abcd-ef1234567890"`
	Title          string                 `json:"title" example:"Implement user authentication"`
	Content        string                 `json:"content" example:"Add JWT-based authentication to the API"`
	CustomFields   map[string]interface{} `json:"customFields" swaggertype:"object,string" example:"importance:high"`
	StartDate      *time.Time             `json:"startDate,omitempty" example:"2024-01-01T00:00:00Z"`
	DueDate        *time.Time             `json:"dueDate,omitempty" example:"2024-12-31T23:59:59Z"`
	ParticipantIDs []uuid.UUID            `json:"participantIds" example:"a1b2c3d4-e5f6-7890-abcd-ef1234567890,b2c3d4e5-f6a7-8901-bcde-f12345678901"`
	Attachments    []AttachmentResponse   `json:"attachments"`
	CreatedAt      time.Time              `json:"createdAt" example:"2024-01-15T10:30:00Z"`
	UpdatedAt      time.Time              `json:"updatedAt" example:"2024-01-15T14:20:00Z"`
}

// PaginatedBoardsResponse represents a paginated list of boards with metadata.
type PaginatedBoardsResponse struct {
	Boards []BoardResponse `json:"boards"`
	Total  int64           `json:"total"`
	Page   int             `json:"page"`
	Limit  int             `json:"limit"`
}

// BoardDetailResponse represents the detailed board response with participants and comments
// @Description Detailed board response with value-based customFields, participants, and comments
// @Description customFields contains field type as key and value string as value (not UUIDs)
// @Description Example: {"importance": "high", "role": "developer", "stage": "in_progress"}
type BoardDetailResponse struct {
	BoardResponse
	Participants []ParticipantResponse `json:"participants"`
	Comments     []CommentResponse     `json:"comments"`
}

// BoardFilters represents the filter parameters for board queries
type BoardFilters struct {
	CustomFields map[string]interface{} `json:"customFields,omitempty"`
}

// MoveBoardRequest represents the request to move a board
type MoveBoardRequest struct {
	ProjectID        string  `json:"projectId" binding:"required" example:"539167fb-b599-41ba-9ead-344a6d0b3a2f"`
	GroupByFieldName string  `json:"groupByFieldName" binding:"required" example:"stage"`
	NewFieldValue    *string `json:"newFieldValue" example:"in_progress"`
}

// MoveBoardResponse represents response after moving a board
type MoveBoardResponse struct {
	BoardID       string `json:"boardId"`
	NewFieldValue string `json:"newFieldValue"`
	Message       string `json:"message"`
}
