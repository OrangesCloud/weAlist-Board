package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"project-board-api/internal/domain"
	"project-board-api/internal/response"
)

func (h *AttachmentHandler) GetBoardAttachments(c *gin.Context) {
	boardIDStr := c.Param("boardId")
	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid board ID")
		return
	}

	attachments, err := h.attachmentRepo.FindByEntityID(c.Request.Context(), domain.EntityTypeBoard, boardID)
	if err != nil {
		response.SendError(c, http.StatusInternalServerError, response.ErrCodeInternal, "Failed to retrieve attachments")
		return
	}

	// Convert to response format
	resp := make([]AttachmentResponse, len(attachments))
	for i, attachment := range attachments {
		// Generate full URL from S3 key when retrieving
		fileURL := h.s3Client.GetFileURL(attachment.FileURL)

		resp[i] = AttachmentResponse{
			ID:          attachment.ID,
			EntityType:  string(attachment.EntityType),
			EntityID:    attachment.EntityID,
			Status:      string(attachment.Status),
			FileName:    attachment.FileName,
			FileURL:     fileURL, // Return full URL to client
			FileSize:    attachment.FileSize,
			ContentType: attachment.ContentType,
			UploadedBy:  attachment.UploadedBy,
			UploadedAt:  attachment.CreatedAt,
			ExpiresAt:   attachment.ExpiresAt,
		}
	}

	response.SendSuccess(c, http.StatusOK, resp)
}

// GetCommentAttachments godoc
// @Summary      Get comment attachments
// @Description  Retrieves all attachments associated with a specific comment
// @Description  Returns only confirmed attachments linked to the comment
// @Tags         attachments
// @Accept       json
// @Produce      json
// @Param        commentId path string true "Comment ID"
// @Success      200 {object} response.SuccessResponse{data=[]AttachmentResponse} "Attachments retrieved successfully"
// @Failure      400 {object} response.ErrorResponse "Invalid comment ID"
// @Failure      500 {object} response.ErrorResponse "Failed to retrieve attachments"
// @Router       /comments/{commentId}/attachments [get]
func (h *AttachmentHandler) GetCommentAttachments(c *gin.Context) {
	commentIDStr := c.Param("commentId")
	commentID, err := uuid.Parse(commentIDStr)
	if err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid comment ID")
		return
	}

	attachments, err := h.attachmentRepo.FindByEntityID(c.Request.Context(), domain.EntityTypeComment, commentID)
	if err != nil {
		response.SendError(c, http.StatusInternalServerError, response.ErrCodeInternal, "Failed to retrieve attachments")
		return
	}

	// Convert to response format
	resp := make([]AttachmentResponse, len(attachments))
	for i, attachment := range attachments {
		// Generate full URL from S3 key when retrieving
		fileURL := h.s3Client.GetFileURL(attachment.FileURL)

		resp[i] = AttachmentResponse{
			ID:          attachment.ID,
			EntityType:  string(attachment.EntityType),
			EntityID:    attachment.EntityID,
			Status:      string(attachment.Status),
			FileName:    attachment.FileName,
			FileURL:     fileURL, // Return full URL to client
			FileSize:    attachment.FileSize,
			ContentType: attachment.ContentType,
			UploadedBy:  attachment.UploadedBy,
			UploadedAt:  attachment.CreatedAt,
			ExpiresAt:   attachment.ExpiresAt,
		}
	}

	response.SendSuccess(c, http.StatusOK, resp)
}

// GetProjectAttachments godoc
// @Summary      Get project attachments
// @Description  Retrieves all attachments associated with a specific project
// @Description  Returns only confirmed attachments linked to the project
// @Tags         attachments
// @Accept       json
// @Produce      json
// @Param        projectId path string true "Project ID"
// @Success      200 {object} response.SuccessResponse{data=[]AttachmentResponse} "Attachments retrieved successfully"
// @Failure      400 {object} response.ErrorResponse "Invalid project ID"
// @Failure      500 {object} response.ErrorResponse "Failed to retrieve attachments"
// @Router       /projects/{projectId}/attachments [get]
func (h *AttachmentHandler) GetProjectAttachments(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid project ID")
		return
	}

	attachments, err := h.attachmentRepo.FindByEntityID(c.Request.Context(), domain.EntityTypeProject, projectID)
	if err != nil {
		response.SendError(c, http.StatusInternalServerError, response.ErrCodeInternal, "Failed to retrieve attachments")
		return
	}

	// Convert to response format
	resp := make([]AttachmentResponse, len(attachments))
	for i, attachment := range attachments {
		// Generate full URL from S3 key when retrieving
		fileURL := h.s3Client.GetFileURL(attachment.FileURL)

		resp[i] = AttachmentResponse{
			ID:          attachment.ID,
			EntityType:  string(attachment.EntityType),
			EntityID:    attachment.EntityID,
			Status:      string(attachment.Status),
			FileName:    attachment.FileName,
			FileURL:     fileURL, // Return full URL to client
			FileSize:    attachment.FileSize,
			ContentType: attachment.ContentType,
			UploadedBy:  attachment.UploadedBy,
			UploadedAt:  attachment.CreatedAt,
			ExpiresAt:   attachment.ExpiresAt,
		}
	}

	response.SendSuccess(c, http.StatusOK, resp)
}

// DeleteAttachment godoc
// @Summary      Delete attachment
// @Description  Deletes an attachment from both S3 and database
// @Description  Only the user who uploaded the attachment can delete it
// @Description  Performs soft delete on the database record
// @Tags         attachments
// @Accept       json
// @Produce      json
// @Param        attachmentId path string true "Attachment ID"
// @Success      200 {object} response.SuccessResponse{data=map[string]string} "Attachment deleted successfully"
// @Failure      400 {object} response.ErrorResponse "Invalid attachment ID"
// @Failure      401 {object} response.ErrorResponse "Unauthorized - user not authenticated"
// @Failure      403 {object} response.ErrorResponse "Forbidden - user does not have permission to delete this attachment"
// @Failure      404 {object} response.ErrorResponse "Attachment not found"
// @Failure      500 {object} response.ErrorResponse "Failed to delete attachment"
// @Router       /attachments/{attachmentId} [delete]
func (h *AttachmentHandler) DeleteAttachment(c *gin.Context) {
	// Parse attachment ID
	attachmentIDStr := c.Param("attachmentId")
	attachmentID, err := uuid.Parse(attachmentIDStr)
	if err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid attachment ID")
		return
	}

	// Get user ID from context (set by auth middleware)
	userIDValue, exists := c.Get("user_id")
	if !exists {
		response.SendError(c, http.StatusUnauthorized, response.ErrCodeUnauthorized, "User not authenticated")
		return
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		// Try parsing as string if it's not already a UUID
		userIDStr, ok := userIDValue.(string)
		if !ok {
			response.SendError(c, http.StatusUnauthorized, response.ErrCodeUnauthorized, "Invalid user ID format")
			return
		}
		userID, err = uuid.Parse(userIDStr)
		if err != nil {
			response.SendError(c, http.StatusUnauthorized, response.ErrCodeUnauthorized, "Invalid user ID format")
			return
		}
	}

	// Find attachment by ID
	attachment, err := h.attachmentRepo.FindByID(c.Request.Context(), attachmentID)
	if err != nil {
		response.SendError(c, http.StatusNotFound, response.ErrCodeNotFound, "Attachment not found")
		return
	}

	// Verify user has permission to delete (must be the uploader)
	if attachment.UploadedBy != userID {
		response.SendError(c, http.StatusForbidden, response.ErrCodeForbidden, "You do not have permission to delete this attachment")
		return
	}

	// FileURL is already the S3 key (not full URL)
	fileKey := attachment.FileURL

	// Delete file from S3
	if fileKey != "" {
		if err := h.s3Client.DeleteFile(c.Request.Context(), fileKey); err != nil {
			// Log error but continue with database deletion
			// This ensures we don't leave orphaned database records
			c.Error(err)
		}
	}

	// Delete attachment record from database (soft delete)
	if err := h.attachmentRepo.Delete(c.Request.Context(), attachmentID); err != nil {
		response.SendError(c, http.StatusInternalServerError, response.ErrCodeInternal, "Failed to delete attachment")
		return
	}

	response.SendSuccess(c, http.StatusOK, map[string]string{
		"message": "Attachment deleted successfully",
	})
}
