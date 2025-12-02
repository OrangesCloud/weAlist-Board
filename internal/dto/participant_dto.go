package dto

import (
	"time"

	"github.com/google/uuid"
)

// AddParticipantsRequest represents the request to add one or more participants to a board
// @Description Request to add single or multiple participants to a board
// @Description For single participant: provide array with 1 element
// @Description For multiple participants: provide array with up to 50 elements
// @Description Duplicate userIds in the request will be automatically removed
type AddParticipantsRequest struct {
	BoardID uuid.UUID   `json:"boardId" binding:"required" example:"1275eac5-f0f9-4bee-8235-576a0042f42b"`
	UserIDs []uuid.UUID `json:"userIds" binding:"required,min=1,max=50" example:"a1b2c3d4-e5f6-7890-abcd-ef1234567890,b2c3d4e5-f6a7-8901-bcde-f12345678901"`
}

// ParticipantResult represents the result of adding a single participant
// @Description Result for each participant addition attempt
// @Description success=true means participant was added successfully
// @Description success=false means addition failed, error field contains reason
type ParticipantResult struct {
	UserID  uuid.UUID `json:"userId" example:"a1b2c3d4-e5f6-7890-abcd-ef1234567890"`
	Success bool      `json:"success" example:"true"`
	Error   string    `json:"error,omitempty" example:"Participant already exists"`
}

// AddParticipantsResponse represents the response for adding participants
// @Description Response for bulk participant addition
// @Description HTTP 201: All participants added successfully (totalFailed=0)
// @Description HTTP 207: Partial success (totalSuccess>0 and totalFailed>0)
// @Description HTTP 400: All participants failed (totalSuccess=0)
type AddParticipantsResponse struct {
	TotalRequested int                 `json:"totalRequested" example:"3"`
	TotalSuccess   int                 `json:"totalSuccess" example:"2"`
	TotalFailed    int                 `json:"totalFailed" example:"1"`
	Results        []ParticipantResult `json:"results"`
}

// ParticipantResponse represents the participant response
// @Description Participant information for a board
type ParticipantResponse struct {
	ID        uuid.UUID `json:"id" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"`
	BoardID   uuid.UUID `json:"boardId" example:"1275eac5-f0f9-4bee-8235-576a0042f42b"`
	UserID    uuid.UUID `json:"userId" example:"a1b2c3d4-e5f6-7890-abcd-ef1234567890"`
	CreatedAt time.Time `json:"createdAt" example:"2024-01-15T10:30:00Z"`
}
