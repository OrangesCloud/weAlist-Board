package job

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"project-board-api/internal/client"
	"project-board-api/internal/repository"
)

// CleanupJob handles cleanup of expired temporary attachments
type CleanupJob struct {
	attachmentRepo repository.AttachmentRepository
	s3Client       client.S3ClientInterface
	logger         *zap.Logger
}

// NewCleanupJob creates a new CleanupJob instance
func NewCleanupJob(
	attachmentRepo repository.AttachmentRepository,
	s3Client client.S3ClientInterface,
	logger *zap.Logger,
) *CleanupJob {
	return &CleanupJob{
		attachmentRepo: attachmentRepo,
		s3Client:       s3Client,
		logger:         logger,
	}
}

// Run executes the cleanup job
// It finds all expired temporary attachments and deletes them from both S3 and the database
func (j *CleanupJob) Run() {
	ctx := context.Background()
	
	j.logger.Info("Starting cleanup job for expired temporary attachments")
	
	// Find expired temporary attachments
	expiredAttachments, err := j.attachmentRepo.FindExpiredTempAttachments(ctx)
	if err != nil {
		j.logger.Error("Failed to find expired temporary attachments",
			zap.Error(err),
		)
		return
	}
	
	if len(expiredAttachments) == 0 {
		j.logger.Info("No expired temporary attachments found")
		return
	}
	
	j.logger.Info("Found expired temporary attachments",
		zap.Int("count", len(expiredAttachments)),
	)
	
	// Delete files from S3 and collect IDs for batch deletion
	var successfulDeletionIDs []uuid.UUID
	successCount := 0
	failCount := 0
	
	for _, attachment := range expiredAttachments {
		// Extract file key from file URL
		fileKey := j.extractFileKeyFromURL(attachment.FileURL)
		if fileKey == "" {
			j.logger.Warn("Failed to extract file key from URL",
				zap.String("attachment_id", attachment.ID.String()),
				zap.String("file_url", attachment.FileURL),
			)
			failCount++
			continue
		}
		
		// Delete from S3
		if err := j.s3Client.DeleteFile(ctx, fileKey); err != nil {
			j.logger.Error("Failed to delete file from S3",
				zap.String("attachment_id", attachment.ID.String()),
				zap.String("file_key", fileKey),
				zap.Error(err),
			)
			failCount++
			continue
		}
		
		successfulDeletionIDs = append(successfulDeletionIDs, attachment.ID)
		successCount++
		
		j.logger.Debug("Deleted file from S3",
			zap.String("attachment_id", attachment.ID.String()),
			zap.String("file_key", fileKey),
		)
	}
	
	// Delete from database in batch
	if len(successfulDeletionIDs) > 0 {
		if err := j.attachmentRepo.DeleteBatch(ctx, successfulDeletionIDs); err != nil {
			j.logger.Error("Failed to delete attachments from database",
				zap.Int("count", len(successfulDeletionIDs)),
				zap.Error(err),
			)
		} else {
			j.logger.Info("Successfully deleted attachments from database",
				zap.Int("count", len(successfulDeletionIDs)),
			)
		}
	}
	
	j.logger.Info("Cleanup job completed",
		zap.Int("total_expired", len(expiredAttachments)),
		zap.Int("success", successCount),
		zap.Int("failed", failCount),
	)
}

// extractFileKeyFromURL extracts the S3 file key from a full S3 URL
// Example: https://bucket.s3.region.amazonaws.com/board/boards/workspace/2024/01/file.jpg -> board/boards/workspace/2024/01/file.jpg
func (j *CleanupJob) extractFileKeyFromURL(fileURL string) string {
	// Handle S3 URL format: https://bucket.s3.region.amazonaws.com/key
	// or https://s3.region.amazonaws.com/bucket/key
	
	// Find the position after the domain
	parts := strings.SplitN(fileURL, "/", 4)
	if len(parts) < 4 {
		return ""
	}
	
	// The key is everything after the third slash
	return parts[3]
}
