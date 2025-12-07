package handler

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"project-board-api/internal/domain"
)

func TestIntegrationAttachmentsRetrieval(t *testing.T) {
	db := setupIntegrationTestDB(t)

	project := createTestProject(t, db)
	board := createTestBoard(t, db, project.ID)
	uploaderID := uuid.New()

	tests := []struct {
		name         string
		entityType   domain.EntityType
		entityID     uuid.UUID
		attachments  []domain.Attachment
		validateFunc func(*testing.T, []domain.Attachment)
	}{
		{
			name:       "Board with attachments",
			entityType: domain.EntityTypeBoard,
			entityID:   board.ID,
			attachments: []domain.Attachment{
				{
					BaseModel: domain.BaseModel{
						ID:        uuid.New(),
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					EntityType:  domain.EntityTypeBoard,
					EntityID:    &board.ID,
					Status:      domain.AttachmentStatusConfirmed,
					FileName:    "document.pdf",
					FileURL:     "https://s3.amazonaws.com/bucket/doc.pdf",
					FileSize:    1024000,
					ContentType: "application/pdf",
					UploadedBy:  uploaderID,
					ExpiresAt:   nil,
				},
				{
					BaseModel: domain.BaseModel{
						ID:        uuid.New(),
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					EntityType:  domain.EntityTypeBoard,
					EntityID:    &board.ID,
					Status:      domain.AttachmentStatusConfirmed,
					FileName:    "image.png",
					FileURL:     "https://s3.amazonaws.com/bucket/img.png",
					FileSize:    512000,
					ContentType: "image/png",
					UploadedBy:  uploaderID,
					ExpiresAt:   nil,
				},
			},
			validateFunc: func(t *testing.T, attachments []domain.Attachment) {
				assert.Len(t, attachments, 2, "Should have 2 attachments")
				assert.Equal(t, "document.pdf", attachments[0].FileName)
				assert.Equal(t, "image.png", attachments[1].FileName)
			},
		},
		{
			name:       "Project with attachments",
			entityType: domain.EntityTypeProject,
			entityID:   project.ID,
			attachments: []domain.Attachment{
				{
					BaseModel: domain.BaseModel{
						ID:        uuid.New(),
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					EntityType:  domain.EntityTypeProject,
					EntityID:    &project.ID,
					Status:      domain.AttachmentStatusConfirmed,
					FileName:    "spec.docx",
					FileURL:     "https://s3.amazonaws.com/bucket/spec.docx",
					FileSize:    2048000,
					ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
					UploadedBy:  uploaderID,
					ExpiresAt:   nil,
				},
			},
			validateFunc: func(t *testing.T, attachments []domain.Attachment) {
				assert.Len(t, attachments, 1, "Should have 1 attachment")
				assert.Equal(t, "spec.docx", attachments[0].FileName)
			},
		},
		{
			name:        "Board without attachments",
			entityType:  domain.EntityTypeBoard,
			entityID:    board.ID,
			attachments: []domain.Attachment{},
			validateFunc: func(t *testing.T, attachments []domain.Attachment) {
				assert.Len(t, attachments, 0, "Should have 0 attachments")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up attachments from previous tests
			db.Exec("DELETE FROM attachments WHERE entity_id = ?", tt.entityID)

			// Create attachments
			for _, attachment := range tt.attachments {
				err := db.Create(&attachment).Error
				require.NoError(t, err, "Failed to create attachment")
			}

			// Retrieve attachments
			var retrieved []domain.Attachment
			err := db.Where("entity_type = ? AND entity_id = ?", tt.entityType, tt.entityID).Find(&retrieved).Error
			require.NoError(t, err, "Failed to retrieve attachments")

			tt.validateFunc(t, retrieved)
		})
	}
}

// TestIntegration_DateValidation tests date validation for boards and projects
// **Validates: Requirements 6.5, 7.4**
