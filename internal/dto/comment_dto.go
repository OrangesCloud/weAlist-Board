package dto

import (
	"time"

	"github.com/google/uuid"
)

// CreateCommentRequest represents the request to create a new comment
// @Description Request body for creating a new comment with optional attachments
// @Description attachmentIds is an optional array of attachment IDs to link to the comment
type CreateCommentRequest struct {
	BoardID       uuid.UUID   `json:"boardId" binding:"required"`
	Content       string      `json:"content" binding:"required,min=1"`
	AttachmentIDs []uuid.UUID `json:"attachmentIds,omitempty" binding:"omitempty,dive,uuid" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"`
}

// UpdateCommentRequest represents the request to update a comment
// @Description Request body for updating a comment with optional attachments
// @Description attachmentIds is an optional array of attachment IDs to add to the comment
type UpdateCommentRequest struct {
	Content       string      `json:"content" binding:"required,min=1"`
	AttachmentIDs []uuid.UUID `json:"attachmentIds,omitempty" binding:"omitempty,dive,uuid" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"`
}

// CommentResponse represents the comment response
type CommentResponse struct {
	CommentID   uuid.UUID            `json:"commentId"`
	BoardID     uuid.UUID            `json:"boardId"`
	UserID      uuid.UUID            `json:"userId"`
	Content     string               `json:"content"`
	Attachments []AttachmentResponse `json:"attachments"`
	CreatedAt   time.Time            `json:"createdAt"`
	UpdatedAt   time.Time            `json:"updatedAt"`
}
