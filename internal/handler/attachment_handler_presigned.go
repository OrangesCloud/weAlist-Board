package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"project-board-api/internal/domain"
	"project-board-api/internal/response"
)

func (h *AttachmentHandler) GeneratePresignedURL(c *gin.Context) {
	var req PresignedURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid request body")
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
		var err error
		userID, err = uuid.Parse(userIDStr)
		if err != nil {
			response.SendError(c, http.StatusUnauthorized, response.ErrCodeUnauthorized, "Invalid user ID format")
			return
		}
	}

	// Validate file size
	if req.FileSize <= 0 {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "File size must be greater than 0")
		return
	}
	if req.FileSize > MaxFileSize {
		response.SendError(c, http.StatusBadRequest, "FILE_TOO_LARGE", "File size exceeds 50MB limit")
		return
	}

	// Validate entity type
	entityType, err := validateEntityType(req.EntityType)
	if err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, err.Error())
		return
	}

	// Validate file type and extension
	if err := validateFileType(req.FileName, req.ContentType); err != nil {
		response.SendError(c, http.StatusBadRequest, "INVALID_FILE_TYPE", err.Error())
		return
	}

	// Validate workspace ID
	if _, err := uuid.Parse(req.WorkspaceID); err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid workspace ID")
		return
	}

	// Convert entity type to lowercase for S3 key generation
	entityTypeKey := strings.ToLower(string(entityType)) + "s"

	// Generate presigned URL
	uploadURL, fileKey, err := h.s3Client.GeneratePresignedURL(
		c.Request.Context(),
		entityTypeKey,
		req.WorkspaceID,
		req.FileName,
		req.ContentType,
	)
	if err != nil {
		response.SendError(c, http.StatusInternalServerError, response.ErrCodeInternal, "Failed to generate presigned URL")
		return
	}

	// Create attachment record with temporary status
	now := time.Now()
	expiresAt := now.Add(1 * time.Hour) // Expires in 1 hour

	attachment := &domain.Attachment{
		BaseModel: domain.BaseModel{
			ID: uuid.New(), // Generate UUID in Go code
		},
		EntityType:  entityType,
		EntityID:    nil, // Will be set when entity is created
		Status:      domain.AttachmentStatusTemp,
		FileName:    req.FileName,
		FileURL:     fileKey, // S3 key only (not full URL)
		FileSize:    req.FileSize,
		ContentType: req.ContentType,
		UploadedBy:  userID,
		ExpiresAt:   &expiresAt,
	}

	// Save to database
	if err := h.attachmentRepo.Create(c.Request.Context(), attachment); err != nil {
		response.SendError(c, http.StatusInternalServerError, response.ErrCodeInternal, "Failed to create attachment record")
		return
	}

	// Return presigned URL response with attachment ID
	resp := PresignedURLResponse{
		AttachmentID: attachment.ID,
		UploadURL:    uploadURL,
		FileKey:      fileKey,
		ExpiresIn:    300, // 5 minutes
	}

	response.SendSuccess(c, http.StatusOK, resp)
}

// validateEntityType validates and converts entity type string to domain.EntityType
