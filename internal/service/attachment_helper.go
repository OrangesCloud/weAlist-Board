// Package service provides business logic implementations.
package service

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"project-board-api/internal/domain"
	"project-board-api/internal/response"
)

// toDomainAttachments converts []*domain.Attachment (pointer slice) to []domain.Attachment (value slice)
func toDomainAttachments(attachments []*domain.Attachment) []domain.Attachment {
	if attachments == nil {
		return nil
	}
	result := make([]domain.Attachment, len(attachments))
	for i, att := range attachments {
		// 포인터 역참조 (*)를 통해 값 복사
		if att != nil {
			result[i] = *att
		}
		// nil 포인터는 해당 위치를 빈(zero value) domain.Attachment로 남겨둠
	}
	return result
}

// removeDuplicateUUIDs removes duplicate UUIDs from a slice
func removeDuplicateUUIDs(uuids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]bool)
	result := make([]uuid.UUID, 0, len(uuids))

	for _, id := range uuids {
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}

	return result
}

// validateDateRange validates that startDate is not after dueDate
func validateDateRange(startDate, dueDate *time.Time) error {
	if startDate != nil && dueDate != nil {
		if startDate.After(*dueDate) {
			return response.NewAppError(response.ErrCodeValidation, "Start date cannot be after due date", "")
		}
	}
	return nil
}

// extractS3KeyFromURL extracts the S3 key from a full S3 URL
// Example: https://bucket.s3.region.amazonaws.com/board/boards/workspace/2024/01/file.jpg -> board/boards/workspace/2024/01/file.jpg
func extractS3KeyFromURL(fileURL string) string {
	// Find the position after the domain
	// Format: https://{bucket}.s3.{region}.amazonaws.com/{key}
	start := strings.Index(fileURL, ".amazonaws.com/")
	if start == -1 {
		// Try alternative format for MinIO or custom endpoints
		// Format: http://localhost:9000/{bucket}/{key}
		parts := strings.SplitN(fileURL, "/", 5)
		if len(parts) >= 5 {
			// Skip protocol, domain, and bucket name, return key only
			return strings.Join(parts[4:], "/")
		}
		return ""
	}

	// Extract key after .amazonaws.com/
	key := fileURL[start+len(".amazonaws.com/"):]
	return key
}
