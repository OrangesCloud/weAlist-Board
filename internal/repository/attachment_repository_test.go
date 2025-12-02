package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"project-board-api/internal/domain"
)

func setupAttachmentTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// Create attachments table for SQLite compatibility
	db.Exec(`CREATE TABLE attachments (
		id TEXT PRIMARY KEY,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		deleted_at DATETIME,
		entity_type TEXT NOT NULL,
		entity_id TEXT,
		status TEXT NOT NULL DEFAULT 'TEMP',
		file_name TEXT NOT NULL,
		file_url TEXT NOT NULL,
		file_size INTEGER NOT NULL,
		content_type TEXT NOT NULL,
		uploaded_by TEXT NOT NULL,
		expires_at DATETIME
	)`)

	return db
}

func TestAttachmentRepository_FindExpiredTempAttachments(t *testing.T) {
	db := setupAttachmentTestDB(t)
	repo := NewAttachmentRepository(db)
	ctx := context.Background()

	now := time.Now()
	pastTime := now.Add(-2 * time.Hour)
	futureTime := now.Add(2 * time.Hour)

	// Create expired temp attachment
	expiredAttachment := &domain.Attachment{
		BaseModel:   domain.BaseModel{ID: uuid.New()},
		EntityType:  domain.EntityTypeBoard,
		Status:      domain.AttachmentStatusTemp,
		FileName:    "expired.jpg",
		FileURL:     "https://s3.amazonaws.com/bucket/expired.jpg",
		FileSize:    1024,
		ContentType: "image/jpeg",
		UploadedBy:  uuid.New(),
		ExpiresAt:   &pastTime,
	}
	db.Create(expiredAttachment)

	// Create non-expired temp attachment
	validTempAttachment := &domain.Attachment{
		BaseModel:   domain.BaseModel{ID: uuid.New()},
		EntityType:  domain.EntityTypeBoard,
		Status:      domain.AttachmentStatusTemp,
		FileName:    "valid.jpg",
		FileURL:     "https://s3.amazonaws.com/bucket/valid.jpg",
		FileSize:    2048,
		ContentType: "image/jpeg",
		UploadedBy:  uuid.New(),
		ExpiresAt:   &futureTime,
	}
	db.Create(validTempAttachment)

	// Create confirmed attachment (should not be returned even if expired)
	entityID := uuid.New()
	confirmedAttachment := &domain.Attachment{
		BaseModel:   domain.BaseModel{ID: uuid.New()},
		EntityType:  domain.EntityTypeBoard,
		EntityID:    &entityID,
		Status:      domain.AttachmentStatusConfirmed,
		FileName:    "confirmed.jpg",
		FileURL:     "https://s3.amazonaws.com/bucket/confirmed.jpg",
		FileSize:    3072,
		ContentType: "image/jpeg",
		UploadedBy:  uuid.New(),
		ExpiresAt:   &pastTime,
	}
	db.Create(confirmedAttachment)

	// Test: FindExpiredTempAttachments should return only expired temp attachments
	expired, err := repo.FindExpiredTempAttachments(ctx)
	if err != nil {
		t.Fatalf("FindExpiredTempAttachments() error = %v", err)
	}

	// Verify we got exactly 1 expired temp attachment
	if len(expired) != 1 {
		t.Errorf("expected 1 expired temp attachment, got %d", len(expired))
	}

	// Verify it's the correct attachment
	if len(expired) > 0 && expired[0].ID != expiredAttachment.ID {
		t.Errorf("expected expired attachment ID %v, got %v", expiredAttachment.ID, expired[0].ID)
	}
}

func TestAttachmentRepository_ConfirmAttachments(t *testing.T) {
	db := setupAttachmentTestDB(t)
	repo := NewAttachmentRepository(db)
	ctx := context.Background()

	futureTime := time.Now().Add(1 * time.Hour)

	// Create temp attachments
	attachment1 := &domain.Attachment{
		BaseModel:   domain.BaseModel{ID: uuid.New()},
		EntityType:  domain.EntityTypeBoard,
		Status:      domain.AttachmentStatusTemp,
		FileName:    "file1.jpg",
		FileURL:     "https://s3.amazonaws.com/bucket/file1.jpg",
		FileSize:    1024,
		ContentType: "image/jpeg",
		UploadedBy:  uuid.New(),
		ExpiresAt:   &futureTime,
	}
	attachment2 := &domain.Attachment{
		BaseModel:   domain.BaseModel{ID: uuid.New()},
		EntityType:  domain.EntityTypeBoard,
		Status:      domain.AttachmentStatusTemp,
		FileName:    "file2.jpg",
		FileURL:     "https://s3.amazonaws.com/bucket/file2.jpg",
		FileSize:    2048,
		ContentType: "image/jpeg",
		UploadedBy:  uuid.New(),
		ExpiresAt:   &futureTime,
	}
	db.Create(attachment1)
	db.Create(attachment2)

	// Create entity ID to link attachments to
	entityID := uuid.New()

	// Test: ConfirmAttachments should update status and set entity_id
	attachmentIDs := []uuid.UUID{attachment1.ID, attachment2.ID}
	err := repo.ConfirmAttachments(ctx, attachmentIDs, entityID)
	if err != nil {
		t.Fatalf("ConfirmAttachments() error = %v", err)
	}

	// Verify attachment1 was updated
	var updatedAttachment1 domain.Attachment
	db.First(&updatedAttachment1, attachment1.ID)
	if updatedAttachment1.Status != domain.AttachmentStatusConfirmed {
		t.Errorf("expected status CONFIRMED, got %v", updatedAttachment1.Status)
	}
	if updatedAttachment1.EntityID == nil || *updatedAttachment1.EntityID != entityID {
		t.Errorf("expected entity_id %v, got %v", entityID, updatedAttachment1.EntityID)
	}

	// Verify attachment2 was updated
	var updatedAttachment2 domain.Attachment
	db.First(&updatedAttachment2, attachment2.ID)
	if updatedAttachment2.Status != domain.AttachmentStatusConfirmed {
		t.Errorf("expected status CONFIRMED, got %v", updatedAttachment2.Status)
	}
	if updatedAttachment2.EntityID == nil || *updatedAttachment2.EntityID != entityID {
		t.Errorf("expected entity_id %v, got %v", entityID, updatedAttachment2.EntityID)
	}
}

func TestAttachmentRepository_ConfirmAttachments_EmptyList(t *testing.T) {
	db := setupAttachmentTestDB(t)
	repo := NewAttachmentRepository(db)
	ctx := context.Background()

	entityID := uuid.New()

	// Test: ConfirmAttachments with empty list should not error
	err := repo.ConfirmAttachments(ctx, []uuid.UUID{}, entityID)
	if err != nil {
		t.Fatalf("ConfirmAttachments() with empty list error = %v", err)
	}
}

func TestAttachmentRepository_DeleteBatch(t *testing.T) {
	db := setupAttachmentTestDB(t)
	repo := NewAttachmentRepository(db)
	ctx := context.Background()

	// Create attachments
	attachment1 := &domain.Attachment{
		BaseModel:   domain.BaseModel{ID: uuid.New()},
		EntityType:  domain.EntityTypeBoard,
		Status:      domain.AttachmentStatusTemp,
		FileName:    "file1.jpg",
		FileURL:     "https://s3.amazonaws.com/bucket/file1.jpg",
		FileSize:    1024,
		ContentType: "image/jpeg",
		UploadedBy:  uuid.New(),
	}
	attachment2 := &domain.Attachment{
		BaseModel:   domain.BaseModel{ID: uuid.New()},
		EntityType:  domain.EntityTypeBoard,
		Status:      domain.AttachmentStatusTemp,
		FileName:    "file2.jpg",
		FileURL:     "https://s3.amazonaws.com/bucket/file2.jpg",
		FileSize:    2048,
		ContentType: "image/jpeg",
		UploadedBy:  uuid.New(),
	}
	attachment3 := &domain.Attachment{
		BaseModel:   domain.BaseModel{ID: uuid.New()},
		EntityType:  domain.EntityTypeBoard,
		Status:      domain.AttachmentStatusTemp,
		FileName:    "file3.jpg",
		FileURL:     "https://s3.amazonaws.com/bucket/file3.jpg",
		FileSize:    3072,
		ContentType: "image/jpeg",
		UploadedBy:  uuid.New(),
	}
	db.Create(attachment1)
	db.Create(attachment2)
	db.Create(attachment3)

	// Test: DeleteBatch should delete specified attachments
	attachmentIDs := []uuid.UUID{attachment1.ID, attachment2.ID}
	err := repo.DeleteBatch(ctx, attachmentIDs)
	if err != nil {
		t.Fatalf("DeleteBatch() error = %v", err)
	}

	// Verify attachment1 was deleted (should not be found in normal query)
	var deletedAttachment1 domain.Attachment
	result := db.First(&deletedAttachment1, attachment1.ID)
	if result.Error == nil {
		t.Error("expected attachment1 to be deleted, but it was found")
	}

	// Verify attachment2 was deleted (should not be found in normal query)
	var deletedAttachment2 domain.Attachment
	result = db.First(&deletedAttachment2, attachment2.ID)
	if result.Error == nil {
		t.Error("expected attachment2 to be deleted, but it was found")
	}

	// Verify attachment3 was NOT deleted
	var existingAttachment3 domain.Attachment
	result = db.First(&existingAttachment3, attachment3.ID)
	if result.Error != nil {
		t.Fatalf("failed to query attachment3: %v", result.Error)
	}
	if existingAttachment3.ID != attachment3.ID {
		t.Error("expected attachment3 to still exist")
	}
}

func TestAttachmentRepository_DeleteBatch_EmptyList(t *testing.T) {
	db := setupAttachmentTestDB(t)
	repo := NewAttachmentRepository(db)
	ctx := context.Background()

	// Test: DeleteBatch with empty list should not error
	err := repo.DeleteBatch(ctx, []uuid.UUID{})
	if err != nil {
		t.Fatalf("DeleteBatch() with empty list error = %v", err)
	}
}

func TestAttachmentRepository_FindByID(t *testing.T) {
	db := setupAttachmentTestDB(t)
	repo := NewAttachmentRepository(db)
	ctx := context.Background()

	// Create test attachment
	attachment := &domain.Attachment{
		BaseModel:   domain.BaseModel{ID: uuid.New()},
		EntityType:  domain.EntityTypeBoard,
		Status:      domain.AttachmentStatusConfirmed,
		FileName:    "test.jpg",
		FileURL:     "https://s3.amazonaws.com/bucket/test.jpg",
		FileSize:    1024,
		ContentType: "image/jpeg",
		UploadedBy:  uuid.New(),
	}
	db.Create(attachment)

	// Test: FindByID should return the attachment
	found, err := repo.FindByID(ctx, attachment.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.ID != attachment.ID {
		t.Errorf("FindByID() ID = %v, want %v", found.ID, attachment.ID)
	}
	if found.FileName != attachment.FileName {
		t.Errorf("FindByID() FileName = %v, want %v", found.FileName, attachment.FileName)
	}
	if found.FileURL != attachment.FileURL {
		t.Errorf("FindByID() FileURL = %v, want %v", found.FileURL, attachment.FileURL)
	}
}

func TestAttachmentRepository_FindByID_NotFound(t *testing.T) {
	db := setupAttachmentTestDB(t)
	repo := NewAttachmentRepository(db)
	ctx := context.Background()

	// Test: FindByID with non-existent ID should return error
	nonExistentID := uuid.New()
	_, err := repo.FindByID(ctx, nonExistentID)
	if err == nil {
		t.Error("FindByID() expected error for non-existent ID, got nil")
	}
}
