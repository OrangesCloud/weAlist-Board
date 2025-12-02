package dto

import (
	"time"

	"github.com/google/uuid"
)

// CreateProjectRequest represents the request to create a new project
// @Description Request body for creating a new project with optional start and due dates
// @Description startDate and dueDate are optional, but startDate must be before or equal to dueDate if both are provided
// @Description attachmentIds is an optional array of attachment IDs to link to the project
type CreateProjectRequest struct {
	WorkspaceID   uuid.UUID   `json:"workspaceId" binding:"required" example:"539167fb-b599-41ba-9ead-344a6d0b3a2f"`
	Name          string      `json:"name" binding:"required,min=2,max=100" example:"Q1 2024 Product Launch"`
	Description   string      `json:"description" binding:"max=500" example:"Project for launching new product features in Q1 2024"`
	StartDate     *time.Time  `json:"startDate,omitempty" example:"2024-01-01T00:00:00Z"`
	DueDate       *time.Time  `json:"dueDate,omitempty" example:"2024-03-31T23:59:59Z"`
	AttachmentIDs []uuid.UUID `json:"attachmentIds,omitempty" binding:"omitempty,dive,uuid" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"`
}

// UpdateProjectRequest represents the request to update a project
// @Description Request body for updating a project. All fields are optional.
// @Description startDate must be before or equal to dueDate if both are provided
// @Description attachmentIds is an optional array of attachment IDs to add to the project
type UpdateProjectRequest struct {
	Name          *string     `json:"name" binding:"omitempty,min=2,max=100" example:"Q1 2024 Product Launch - Updated"`
	Description   *string     `json:"description" binding:"omitempty,max=500" example:"Updated project description"`
	StartDate     *time.Time  `json:"startDate,omitempty" example:"2024-01-15T00:00:00Z"`
	DueDate       *time.Time  `json:"dueDate,omitempty" example:"2024-04-15T23:59:59Z"`
	AttachmentIDs []uuid.UUID `json:"attachmentIds,omitempty" binding:"omitempty,dive,uuid" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"`
}

// ProjectResponse represents the project response
// @Description Project response with optional start/due dates and attachments
// @Description startDate and dueDate are included only if they were set
// @Description attachments is an array of file metadata (empty array if no attachments)
type ProjectResponse struct {
	ID          uuid.UUID            `json:"projectId" example:"539167fb-b599-41ba-9ead-344a6d0b3a2f"`
	WorkspaceID uuid.UUID            `json:"workspaceId" example:"a1b2c3d4-e5f6-7890-abcd-ef1234567890"`
	OwnerID     uuid.UUID            `json:"ownerId" example:"b2c3d4e5-f6a7-8901-bcde-f12345678901"`
	OwnerEmail  string               `json:"ownerEmail,omitempty" example:"owner@example.com"`
	OwnerName   string               `json:"ownerName,omitempty" example:"John Doe"`
	Name        string               `json:"name" example:"Q1 2024 Product Launch"`
	Description string               `json:"description" example:"Project for launching new product features in Q1 2024"`
	IsPublic    bool                 `json:"isPublic" example:"true"`
	StartDate   *time.Time           `json:"startDate,omitempty" example:"2024-01-01T00:00:00Z"`
	DueDate     *time.Time           `json:"dueDate,omitempty" example:"2024-03-31T23:59:59Z"`
	Attachments []AttachmentResponse `json:"attachments"`
	CreatedAt   time.Time            `json:"createdAt" example:"2024-01-15T10:30:00Z"`
	UpdatedAt   time.Time            `json:"updatedAt" example:"2024-01-15T14:20:00Z"`
}

// ProjectMemberResponse represents a project member
type ProjectMemberResponse struct {
	MemberID  uuid.UUID `json:"memberId"`
	ProjectID uuid.UUID `json:"projectId"`
	UserID    uuid.UUID `json:"userId"`
	UserEmail string    `json:"userEmail,omitempty"`
	UserName  string    `json:"userName,omitempty"`
	RoleName  string    `json:"roleName"`
	JoinedAt  time.Time `json:"joinedAt"`
}

// UpdateProjectMemberRoleRequest represents the request to update member role
type UpdateProjectMemberRoleRequest struct {
	RoleName string `json:"roleName" binding:"required,oneof=OWNER ADMIN MEMBER"`
}

// ProjectJoinRequestResponse represents a join request
type ProjectJoinRequestResponse struct {
	RequestID   uuid.UUID `json:"requestId"`
	ProjectID   uuid.UUID `json:"projectId"`
	UserID      uuid.UUID `json:"userId"`
	UserEmail   string    `json:"userEmail,omitempty"`
	UserName    string    `json:"userName,omitempty"`
	Status      string    `json:"status"`
	RequestedAt time.Time `json:"requestedAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// CreateProjectJoinRequestRequest represents the request to join a project
type CreateProjectJoinRequestRequest struct {
	ProjectID uuid.UUID `json:"projectId" binding:"required"`
}

// UpdateProjectJoinRequestRequest represents the request to update join request status
type UpdateProjectJoinRequestRequest struct {
	Status string `json:"status" binding:"required,oneof=APPROVED REJECTED"`
}

// PaginatedProjectsResponse represents paginated projects response
type PaginatedProjectsResponse struct {
	Projects []ProjectResponse `json:"projects"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	Limit    int               `json:"limit"`
}

// ProjectInitSettingsResponse represents the initial settings for a project
type ProjectInitSettingsResponse struct {
	Project       ProjectBasicInfo           `json:"project"`
	Fields        []FieldWithOptionsResponse `json:"fields"`
	FieldTypes    []FieldTypeInfo            `json:"fieldTypes"`
	DefaultViewID *uuid.UUID                 `json:"defaultViewId,omitempty"`
}

// ProjectBasicInfo represents basic project information
type ProjectBasicInfo struct {
	ProjectID      uuid.UUID  `json:"projectId"`
	WorkspaceID    uuid.UUID  `json:"workspaceId"`
	WorkspaceName  string     `json:"workspaceName,omitempty"`
	WorkspaceEmail string     `json:"workspaceEmail,omitempty"`
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	OwnerID        uuid.UUID  `json:"ownerId"`
	OwnerEmail     string     `json:"ownerEmail,omitempty"`
	OwnerName      string     `json:"ownerName,omitempty"`
	IsPublic       bool       `json:"isPublic"`
	StartDate      *time.Time `json:"startDate,omitempty"`
	DueDate        *time.Time `json:"dueDate,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// FieldWithOptionsResponse represents a field definition with its options
type FieldWithOptionsResponse struct {
	FieldID     string        `json:"fieldId"`
	FieldName   string        `json:"fieldName"`
	FieldType   string        `json:"fieldType"`
	IsRequired  bool          `json:"isRequired"`
	Options     []FieldOption `json:"options"`
	Description string        `json:"description,omitempty"`
}

// FieldOption represents an option for a field
type FieldOption struct {
	OptionID     string `json:"optionId"`
	OptionLabel  string `json:"optionLabel"`
	OptionValue  string `json:"optionValue"`
	Color        string `json:"color,omitempty"`
	DisplayOrder int    `json:"displayOrder"`
	FieldID      string `json:"fieldId,omitempty"`
	Description  string `json:"description,omitempty"`
}

// FieldTypeInfo represents information about a field type
type FieldTypeInfo struct {
	TypeID      string `json:"typeId"`
	TypeName    string `json:"typeName"`
	Description string `json:"description,omitempty"`
}
