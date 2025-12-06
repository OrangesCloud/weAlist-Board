package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
	"project-board-api/internal/response"
)

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

	// Validate date range with effective dates
	if err := validateDateRange(effectiveStartDate, effectiveDueDate); err != nil {
		return nil, err
	}

	// Validate and confirm attachments if provided
	if len(req.AttachmentIDs) > 0 {
		if err := s.validateAndConfirmAttachments(ctx, req.AttachmentIDs, domain.EntityTypeBoard, uuid.Nil); err != nil {
			return nil, err
		}
	}

	// Update fields if provided
	if req.Title != nil {
		board.Title = *req.Title
	}
	if req.Content != nil {
		board.Content = *req.Content
	}
	if req.CustomFields != nil {
		// Convert values to IDs
		convertedFields, err := s.fieldOptionConverter.ConvertValuesToIDs(ctx, board.ProjectID, *req.CustomFields)
		if err != nil {
			return nil, response.NewAppError(response.ErrCodeValidation, "Invalid custom field values", err.Error())
		}

		// Convert CustomFields to datatypes.JSON
		jsonBytes, err := json.Marshal(convertedFields)
		if err != nil {
			return nil, response.NewAppError(response.ErrCodeInternal, "Failed to marshal custom fields", err.Error())
		}
		board.CustomFields = jsonBytes
	}
	if req.AssigneeID != nil {
		if *req.AssigneeID == uuid.Nil {
			board.AssigneeID = nil
		} else {
			board.AssigneeID = req.AssigneeID
		}
	}
	if req.StartDate != nil {
		board.StartDate = req.StartDate
	}
	if req.DueDate != nil {
		board.DueDate = req.DueDate
	}

	// Update board first
	if err := s.boardRepo.Update(ctx, board); err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to update board", err.Error())
	}

	// Attachments 처리 로직 개선 및 Confirm
	if len(req.AttachmentIDs) > 0 {
		// 에러 발생 시 업데이트 실패 처리
		if err := s.attachmentRepo.ConfirmAttachments(ctx, req.AttachmentIDs, board.ID); err != nil {
			s.logger.Error("Failed to confirm attachments during board update",
				zap.String("board_id", board.ID.String()),
				zap.Strings("attachment_ids", func() []string {
					ids := make([]string, len(req.AttachmentIDs))
					for i, id := range req.AttachmentIDs {
						ids[i] = id.String()
					}
					return ids
				}()),
				zap.Error(err))

			return nil, response.NewAppError(response.ErrCodeInternal,
				"Failed to confirm attachments: "+err.Error(),
				"Please ensure all attachment IDs are valid and not already used")
		}
	}

	// ✅ [수정] Participants 업데이트 로직 - board 업데이트 후 처리
	if req.Participants != nil {
		// 1. 기존 참여자 모두 조회
		existingParticipants, err := s.participantRepo.FindByBoardID(ctx, boardID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Warn("Failed to fetch existing participants for update",
				zap.String("board_id", boardID.String()),
				zap.Error(err))
		}

		// 2. 기존 참여자 모두 삭제
		if len(existingParticipants) > 0 {
			s.logger.Info("Deleting existing participants",
				zap.String("board_id", boardID.String()),
				zap.Int("count", len(existingParticipants)))

			for _, p := range existingParticipants {
				if err := s.participantRepo.Delete(ctx, boardID, p.UserID); err != nil {
					s.logger.Warn("Failed to delete existing participant",
						zap.String("board_id", boardID.String()),
						zap.String("user_id", p.UserID.String()),
						zap.Error(err))
				}
			}
		}

		// 3. 새로운 참여자 추가
		if len(req.Participants) > 0 {
			s.logger.Info("Adding new participants",
				zap.String("board_id", boardID.String()),
				zap.Int("count", len(req.Participants)))

			uniqueUserIDs := removeDuplicateUUIDs(req.Participants)
			for _, userID := range uniqueUserIDs {
				participant := &domain.Participant{
					BoardID: boardID,
					UserID:  userID,
				}
				if err := s.participantRepo.Create(ctx, participant); err != nil {
					s.logger.Warn("Failed to add new participant",
						zap.String("board_id", boardID.String()),
						zap.String("user_id", userID.String()),
						zap.Error(err))
				}
			}
		}
	}

	// board와 연결된 모든 Attachments를 다시 조회합니다. (타입 변환 적용)
	allAttachments, err := s.attachmentRepo.FindByEntityID(ctx, domain.EntityTypeBoard, board.ID)
	if err != nil {
		s.logger.Warn("Failed to fetch all confirmed attachments after update", zap.Error(err))
	} else {
		board.Attachments = toDomainAttachments(allAttachments)
	}

	// ✅ [수정] 업데이트된 participants를 다시 로드
	reloadedBoard, err := s.boardRepo.FindByID(ctx, board.ID)
	if err != nil {
		s.logger.Warn("Failed to reload board with participants after update",
			zap.String("board_id", board.ID.String()),
			zap.Error(err))
	} else {
		board = reloadedBoard
		// Attachments는 위에서 이미 로드했으므로 다시 할당
		board.Attachments = toDomainAttachments(allAttachments)
	}

	// Convert to response DTO
	return s.toBoardResponse(board), nil
}

// DeleteBoard soft deletes a board and its associated attachments
