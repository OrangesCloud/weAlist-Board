package handler

import (
	"github.com/google/uuid"

	"project-board-api/internal/client"
	"project-board-api/internal/repository"
)

// AttachmentHandler handles attachment-related requests
type AttachmentHandler struct {
	s3Client       client.S3ClientInterface
	attachmentRepo repository.AttachmentRepository
}

// NewAttachmentHandler creates a new AttachmentHandler
func NewAttachmentHandler(s3Client client.S3ClientInterface, attachmentRepo repository.AttachmentRepository) *AttachmentHandler {
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
