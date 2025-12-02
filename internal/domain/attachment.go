package domain

import (
	"time"

	"github.com/google/uuid"
)

// EntityType represents the type of entity an attachment is associated with
type EntityType string

const (
	EntityTypeBoard   EntityType = "BOARD"
	EntityTypeProject EntityType = "PROJECT"
	EntityTypeComment EntityType = "COMMENT"
)

// AttachmentStatus represents the status of an attachment
type AttachmentStatus string

const (
	AttachmentStatusTemp      AttachmentStatus = "TEMP"      // Temporary status
	AttachmentStatusConfirmed AttachmentStatus = "CONFIRMED" // Confirmed status
)

// Attachment represents a file attachment associated with a board or project
// This is a polymorphic relationship - EntityID can reference Board, Project, or Comment
// ⚠️ IMPORTANT: Do not add foreign key constraints on EntityID as it references multiple tables
type Attachment struct {
	BaseModel
	EntityType  EntityType       `gorm:"type:varchar(50);not null;index:idx_attachments_entity,priority:1" json:"entity_type"`
	EntityID    *uuid.UUID       `gorm:"type:uuid;index:idx_attachments_entity,priority:2" json:"entity_id"` // ✅ FK 제거, 다형성 관계
	Status      AttachmentStatus `gorm:"type:varchar(20);not null;default:'TEMP';index:idx_attachments_status" json:"status"`
	FileName    string           `gorm:"type:varchar(255);not null" json:"file_name"`
	FileURL     string           `gorm:"type:text;not null" json:"file_url"` // ✅ S3 key만 저장 (full URL 아님)
	FileSize    int64            `gorm:"not null" json:"file_size"`
	ContentType string           `gorm:"type:varchar(100);not null" json:"content_type"`
	UploadedBy  uuid.UUID        `gorm:"type:uuid;not null;index:idx_attachments_uploaded_by" json:"uploaded_by"`
	ExpiresAt   *time.Time       `gorm:"type:timestamp;index:idx_attachments_expires_at" json:"expires_at"`
}

// TableName specifies the table name for Attachment
func (Attachment) TableName() string {
	return "attachments"
}
