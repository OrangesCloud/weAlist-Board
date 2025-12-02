package handler

import (
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"project-board-api/internal/client"
	"project-board-api/internal/domain"
	"project-board-api/internal/repository"
	"project-board-api/internal/response"
)

// AttachmentHandler handles attachment-related requests
type AttachmentHandler struct {
	s3Client       *client.S3Client
	attachmentRepo repository.AttachmentRepository
}

// NewAttachmentHandler creates a new AttachmentHandler
func NewAttachmentHandler(s3Client *client.S3Client, attachmentRepo repository.AttachmentRepository) *AttachmentHandler {
	return &AttachmentHandler{
		s3Client:       s3Client,
		attachmentRepo: attachmentRepo,
	}
}

// File size limit: 50MB
const MaxFileSize = 50 * 1024 * 1024

var (
	AllowedImageTypes = map[string]bool{
		// 기본 이미지
		"image/jpeg":    true,
		"image/jpg":     true,
		"image/png":     true,
		"image/gif":     true,
		"image/webp":    true,
		"image/svg+xml": true,
		"image/heic":    true, // iPhone
	}

	AllowedDocTypes = map[string]bool{
		// 문서
		"application/pdf": true,
		"text/plain":      true,
		"text/markdown":   true,
		"text/csv":        true,

		// MS Office
		"application/msword":            true, // .doc
		"application/vnd.ms-excel":      true, // .xls
		"application/vnd.ms-powerpoint": true, // .ppt
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   true, // .docx
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true, // .xlsx
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": true, // .pptx

		// 압축
		"application/zip":              true,
		"application/x-zip-compressed": true,

		// 데이터
		"application/json": true,
	}

	AllowedImageExtensions = map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
		".svg":  true,
		".heic": true,
	}

	AllowedDocExtensions = map[string]bool{
		".pdf":  true,
		".txt":  true,
		".md":   true,
		".csv":  true,
		".doc":  true,
		".docx": true,
		".xls":  true,
		".xlsx": true,
		".ppt":  true,
		".pptx": true,
		".zip":  true,
		".json": true,
	}
)

// PresignedURLRequest represents the request to generate a presigned URL
type PresignedURLRequest struct {
	EntityType  string `json:"entityType" binding:"required"`
	WorkspaceID string `json:"workspaceId" binding:"required"`
	FileName    string `json:"fileName" binding:"required"`
	FileSize    int64  `json:"fileSize" binding:"required"`
	ContentType string `json:"contentType" binding:"required"`
}

// PresignedURLResponse represents the response containing the presigned URL
type PresignedURLResponse struct {
	AttachmentID uuid.UUID `json:"attachmentId"`
	UploadURL    string    `json:"uploadUrl"`
	FileKey      string    `json:"fileKey"`
	ExpiresIn    int       `json:"expiresIn"` // seconds
}

// GeneratePresignedURL godoc
// @Summary      Generate presigned URL for file upload
// @Description  Generates a presigned URL for uploading a file directly to S3
// @Description  Creates a temporary attachment record and returns its ID along with the presigned URL
// @Description  Validates file metadata (size, type, name) before generating URL
// @Description  Supported entity types: BOARD, COMMENT, PROJECT
// @Description  Supported file types: images (jpg, jpeg, png, gif, webp, svg, heic) and documents (pdf, txt, doc, docx, xls, xlsx, ppt, pptx, zip, json, md, csv)
// @Description  Maximum file size: 50MB
// @Description  URL expires in 5 minutes (300 seconds)
// @Tags         attachments
// @Accept       json
// @Produce      json
// @Param        request body PresignedURLRequest true "Presigned URL request"
// @Success      200 {object} response.SuccessResponse{data=PresignedURLResponse} "Presigned URL generated successfully"
// @Failure      400 {object} response.ErrorResponse "Invalid request or file validation failed"
// @Failure      401 {object} response.ErrorResponse "Unauthorized - user not authenticated"
// @Failure      500 {object} response.ErrorResponse "Failed to generate presigned URL"
// @Router       /attachments/presigned-url [post]
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
