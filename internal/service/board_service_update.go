package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
	"project-board-api/internal/response"
)

// UpdateBoard updates an existing board
func (s *boardServiceImpl) UpdateBoard(ctx context.Context, boardID uuid.UUID, req *dto.UpdateBoardRequest) (*dto.BoardResponse, error) {
	// Fetch existing board
	board, err := s.boardRepo.FindByID(ctx, boardID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewAppError(response.ErrCodeNotFound, "Board not found", "")
		}
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch board", err.Error())
	}

	// Determine the effective start and due dates for validation
	effectiveStartDate := board.StartDate
	effectiveDueDate := board.DueDate

	if req.StartDate != nil {
		effectiveStartDate = req.StartDate
	}
	if req.DueDate != nil {
		effectiveDueDate = req.DueDate
	}

	// Validate date range
	if effectiveStartDate != nil && effectiveDueDate != nil {
		if effectiveStartDate.After(*effectiveDueDate) {
			return nil, response.NewAppError(response.ErrCodeValidation, "Start date cannot be after due date", "")
		}
	}

	// Update fields if provided
	if req.Title != nil {
		board.Title = *req.Title
	}
	if req.Description != nil {
		board.Description = *req.Description
	}
	if req.StartDate != nil {
		board.StartDate = req.StartDate
	}
	if req.DueDate != nil {
		board.DueDate = req.DueDate
	}

	// Handle custom fields update
	if req.CustomFields != nil {
		// Convert values to IDs for storage
		convertedFields, err := s.fieldOptionConverter.ConvertValuesToIDs(ctx, board.ProjectID, req.CustomFields)
		if err != nil {
			s.logger.Error("Failed to convert custom field values to IDs",
				zap.Error(err),
				zap.String("board_id", boardID.String()))
			return nil, response.NewAppError(response.ErrCodeInternal, "Failed to process custom fields", err.Error())
		}

		customFieldsJSON, err := json.Marshal(convertedFields)
		if err != nil {
			return nil, response.NewAppError(response.ErrCodeInternal, "Failed to marshal custom fields", err.Error())
		}
		board.CustomFields = datatypes.JSON(customFieldsJSON)
	}

	// Handle assignee updates
	if req.AssigneeIDs != nil {
		board.AssigneeIDs = req.AssigneeIDs
	}

	// Handle attachment updates
	if req.AttachmentIDs != nil {
		// Get current attachments
		currentAttachments, err := s.attachmentRepo.FindByEntity(ctx, domain.EntityTypeBoard, boardID)
		if err != nil {
			s.logger.Error("Failed to fetch current attachments",
				zap.Error(err),
				zap.String("board_id", boardID.String()))
			return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch current attachments", err.Error())
		}

		// Find attachments to delete (in current but not in new)
		currentIDs := make(map[uuid.UUID]bool)
		for _, att := range currentAttachments {
			currentIDs[att.ID] = true
		}

		newIDs := make(map[uuid.UUID]bool)
		for _, id := range req.AttachmentIDs {
			newIDs[id] = true
		}

		// Delete removed attachments
		var toDelete []*domain.Attachment
		for _, att := range currentAttachments {
			if !newIDs[att.ID] {
				toDelete = append(toDelete, att)
			}
		}

		if len(toDelete) > 0 {
			// Delete from S3 and database asynchronously
			go s.deleteAttachmentsWithS3(context.Background(), toDelete)
		}

		// Confirm new attachments
		var toConfirm []uuid.UUID
		for _, id := range req.AttachmentIDs {
			if !currentIDs[id] {
				toConfirm = append(toConfirm, id)
			}
		}

		if len(toConfirm) > 0 {
			if err := s.validateAndConfirmAttachments(ctx, toConfirm, domain.EntityTypeBoard, boardID); err != nil {
				return nil, err
			}
		}
	}

	// Handle participant updates
	if req.ParticipantIDs != nil {
		// Get current participants
		currentParticipants, err := s.participantRepo.FindByBoard(ctx, boardID)
		if err != nil {
			s.logger.Error("Failed to fetch current participants",
				zap.Error(err),
				zap.String("board_id", boardID.String()))
			return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch current participants", err.Error())
		}

		// Find participants to remove and add
		currentPIDs := make(map[uuid.UUID]bool)
		for _, p := range currentParticipants {
			currentPIDs[p.UserID] = true
		}

		newPIDs := make(map[uuid.UUID]bool)
		for _, id := range req.ParticipantIDs {
			newPIDs[id] = true
		}

		// Remove participants not in new list
		for _, p := range currentParticipants {
			if !newPIDs[p.UserID] {
				if err := s.participantRepo.Delete(ctx, p.ID); err != nil {
					s.logger.Warn("Failed to delete participant",
						zap.Error(err),
						zap.String("participant_id", p.ID.String()))
				}
			}
		}

		// Add new participants
		var toAdd []uuid.UUID
		for _, id := range req.ParticipantIDs {
			if !currentPIDs[id] {
				toAdd = append(toAdd, id)
			}
		}

		if len(toAdd) > 0 {
			if _, err := s.addParticipantsInternal(ctx, boardID, toAdd); err != nil {
				s.logger.Warn("Failed to add some participants",
					zap.Error(err),
					zap.String("board_id", boardID.String()))
			}
		}
	}

	// Save updates
	if err := s.boardRepo.Update(ctx, board); err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to update board", err.Error())
	}

	// Fetch updated board with associations
	updatedBoard, err := s.boardRepo.FindByID(ctx, boardID)
	if err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch updated board", err.Error())
	}

	// Convert custom fields to values for response
	if err := s.convertBoardCustomFieldsToValues(ctx, updatedBoard); err != nil {
		s.logger.Warn("Failed to convert custom fields to values",
			zap.Error(err),
			zap.String("board_id", boardID.String()))
	}

	return s.toBoardResponse(updatedBoard), nil
}
