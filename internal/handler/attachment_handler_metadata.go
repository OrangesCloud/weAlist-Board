package handler

import (
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"project-board-api/internal/domain"
	"project-board-api/internal/response"
)

func validateEntityType(entityTypeStr string) (domain.EntityType, error) {
	entityType := domain.EntityType(strings.ToUpper(entityTypeStr))

	switch entityType {
	case domain.EntityTypeBoard, domain.EntityTypeComment, domain.EntityTypeProject:
		return entityType, nil
	default:
		return "", response.NewValidationError("Invalid entity type", "Entity type must be BOARD, COMMENT, or PROJECT")
	}
}

// validateFileType validates file type and extension
func validateFileType(fileName, contentType string) error {
	// Extract file extension using filepath.Ext
	fileExt := strings.ToLower(filepath.Ext(fileName))

	if fileExt == "" {
		return response.NewValidationError("Invalid file name", "File must have an extension")
	}

	// Check if it's an allowed image or document type
	isAllowedImage := AllowedImageTypes[contentType] && AllowedImageExtensions[fileExt]
	isAllowedDoc := AllowedDocTypes[contentType] && AllowedDocExtensions[fileExt]

	if !isAllowedImage && !isAllowedDoc {
		return response.NewValidationError(
			"Unsupported file type",
			"Supported types: images (jpg, jpeg, png, gif, webp, svg, heic) and documents (pdf, txt, doc, docx, xls, xlsx, ppt, pptx, zip, json, md, csv)",
		)
	}

	return nil
}

// SaveAttachmentMetadataRequest represents the request to save attachment metadata
type SaveAttachmentMetadataRequest struct {
	EntityType  string `json:"entityType" binding:"required"`
	FileKey     string `json:"fileKey" binding:"required"`
	FileName    string `json:"fileName" binding:"required"`
	FileSize    int64  `json:"fileSize" binding:"required"`
	ContentType string `json:"contentType" binding:"required"`
}

// AttachmentResponse represents the attachment metadata response
type AttachmentResponse struct {
	ID          uuid.UUID  `json:"id"`
	EntityType  string     `json:"entityType"`
	EntityID    *uuid.UUID `json:"entityId"`
	Status      string     `json:"status"`
	FileName    string     `json:"fileName"`
	FileURL     string     `json:"fileUrl"`
	FileSize    int64      `json:"fileSize"`
	ContentType string     `json:"contentType"`
	UploadedBy  uuid.UUID  `json:"uploadedBy"`
	UploadedAt  time.Time  `json:"uploadedAt"`
	ExpiresAt   *time.Time `json:"expiresAt"`
}

// SaveAttachmentMetadata godoc
// @Summary      Save attachment metadata
// @Description  Saves attachment metadata to the database after successful S3 upload
// @Description  Creates a temporary attachment record with 1-hour expiration
// @Description  The attachment will be linked to an entity (board/comment/project) when that entity is created
// @Description  Supported entity types: BOARD, COMMENT, PROJECT
// @Tags         attachments
// @Accept       json
// @Produce      json
// @Param        request body SaveAttachmentMetadataRequest true "Attachment metadata"
// @Success      201 {object} response.SuccessResponse{data=AttachmentResponse} "Attachment metadata saved successfully"
// @Failure      400 {object} response.ErrorResponse "Invalid request or file validation failed"
// @Failure      401 {object} response.ErrorResponse "Unauthorized - user not authenticated"
// @Failure      500 {object} response.ErrorResponse "Failed to save attachment metadata"
// @Router       /attachments [post]
func (h *AttachmentHandler) SaveAttachmentMetadata(c *gin.Context) {
	var req SaveAttachmentMetadataRequest
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

	// Validate entity type
	entityType, err := validateEntityType(req.EntityType)
	if err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, err.Error())
		return
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

	// Validate file type
	if err := validateFileType(req.FileName, req.ContentType); err != nil {
		response.SendError(c, http.StatusBadRequest, "INVALID_FILE_TYPE", err.Error())
		return
	}

	// Validate fileKey format (should not be empty and should contain expected structure)
	if req.FileKey == "" {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "File key is required")
		return
	}

	// Validate fileKey starts with expected prefix
	expectedPrefix := "board/"
	if !strings.HasPrefix(req.FileKey, expectedPrefix) {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid file key format")
		return
	}

	// Create attachment record with temporary status
	// Note: EntityID is nil to indicate temporary attachment
	// This will be updated when the entity (board/comment/project) is created
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
		FileURL:     req.FileKey, // S3 key only (not full URL)
		FileSize:    req.FileSize,
		ContentType: req.ContentType,
		UploadedBy:  userID,
		ExpiresAt:   &expiresAt,
	}

	// Save to database
	if err := h.attachmentRepo.Create(c.Request.Context(), attachment); err != nil {
		response.SendError(c, http.StatusInternalServerError, response.ErrCodeInternal, "Failed to save attachment metadata")
		return
	}

	// Prepare response
	resp := AttachmentResponse{
		ID:          attachment.ID,
		EntityType:  string(attachment.EntityType),
		EntityID:    attachment.EntityID,
		Status:      string(attachment.Status),
		FileName:    attachment.FileName,
		FileURL:     attachment.FileURL,
		FileSize:    attachment.FileSize,
		ContentType: attachment.ContentType,
		UploadedBy:  attachment.UploadedBy,
		UploadedAt:  attachment.CreatedAt,
		ExpiresAt:   attachment.ExpiresAt,
	}

	response.SendSuccess(c, http.StatusCreated, resp)
}

// GetBoardAttachments godoc
// @Summary      Get board attachments
// @Description  Retrieves all attachments associated with a specific board
// @Description  Returns only confirmed attachments linked to the board
// @Tags         attachments
// @Accept       json
// @Produce      json
// @Param        boardId path string true "Board ID"
// @Success      200 {object} response.SuccessResponse{data=[]AttachmentResponse} "Attachments retrieved successfully"
// @Failure      400 {object} response.ErrorResponse "Invalid board ID"
// @Failure      500 {object} response.ErrorResponse "Failed to retrieve attachments"
// @Router       /boards/{boardId}/attachments [get]
