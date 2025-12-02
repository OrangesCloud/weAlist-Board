package job

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"project-board-api/internal/domain"
)

// MockAttachmentRepository is a mock implementation of AttachmentRepository
type MockAttachmentRepository struct {
	mock.Mock
}

func (m *MockAttachmentRepository) Create(ctx context.Context, attachment *domain.Attachment) error {
	args := m.Called(ctx, attachment)
	return args.Error(0)
}

func (m *MockAttachmentRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Attachment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Attachment), args.Error(1)
}

func (m *MockAttachmentRepository) FindByEntityID(ctx context.Context, entityType domain.EntityType, entityID uuid.UUID) ([]*domain.Attachment, error) {
	args := m.Called(ctx, entityType, entityID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Attachment), args.Error(1)
}

func (m *MockAttachmentRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.Attachment, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Attachment), args.Error(1)
}

func (m *MockAttachmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAttachmentRepository) FindExpiredTempAttachments(ctx context.Context) ([]*domain.Attachment, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Attachment), args.Error(1)
}

func (m *MockAttachmentRepository) ConfirmAttachments(ctx context.Context, attachmentIDs []uuid.UUID, entityID uuid.UUID) error {
	args := m.Called(ctx, attachmentIDs, entityID)
	return args.Error(0)
}

func (m *MockAttachmentRepository) DeleteBatch(ctx context.Context, attachmentIDs []uuid.UUID) error {
	args := m.Called(ctx, attachmentIDs)
	return args.Error(0)
}

// MockS3Client is a mock implementation of S3ClientInterface
type MockS3Client struct {
	mock.Mock
}

func (m *MockS3Client) DeleteFile(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockS3Client) GenerateFileKey(entityType, workspaceID, fileExt string) (string, error) {
	args := m.Called(entityType, workspaceID, fileExt)
	return args.String(0), args.Error(1)
}

func (m *MockS3Client) GeneratePresignedURL(ctx context.Context, entityType, workspaceID, fileName, contentType string) (string, string, error) {
	args := m.Called(ctx, entityType, workspaceID, fileName, contentType)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockS3Client) UploadFile(ctx context.Context, key string, file io.Reader, contentType string) (string, error) {
	args := m.Called(ctx, key, file, contentType)
	return args.String(0), args.Error(1)
}

func (m *MockS3Client) GetFileURL(key string) string {
	args := m.Called(key)
	return args.String(0)
}

func TestCleanupJob_Run_ExpiredFilesDeleted(t *testing.T) {
	// Setup
	mockRepo := new(MockAttachmentRepository)
	mockS3 := new(MockS3Client)
	logger := zap.NewNop()
	
	job := NewCleanupJob(mockRepo, mockS3, logger)
	
	// Create expired attachments
	now := time.Now()
	expiredTime := now.Add(-2 * time.Hour)
	
	attachment1 := &domain.Attachment{
		BaseModel: domain.BaseModel{
			ID: uuid.New(),
		},
		EntityType:  domain.EntityTypeBoard,
		Status:      domain.AttachmentStatusTemp,
		FileName:    "test1.jpg",
		FileURL:     "https://bucket.s3.region.amazonaws.com/board/boards/workspace1/2024/01/file1.jpg",
		FileSize:    1024,
		ContentType: "image/jpeg",
		UploadedBy:  uuid.New(),
		ExpiresAt:   &expiredTime,
	}
	
	attachment2 := &domain.Attachment{
		BaseModel: domain.BaseModel{
			ID: uuid.New(),
		},
		EntityType:  domain.EntityTypeComment,
		Status:      domain.AttachmentStatusTemp,
		FileName:    "test2.pdf",
		FileURL:     "https://bucket.s3.region.amazonaws.com/board/comments/workspace1/2024/01/file2.pdf",
		FileSize:    2048,
		ContentType: "application/pdf",
		UploadedBy:  uuid.New(),
		ExpiresAt:   &expiredTime,
	}
	
	expiredAttachments := []*domain.Attachment{attachment1, attachment2}
	
	// Mock expectations
	mockRepo.On("FindExpiredTempAttachments", mock.Anything).Return(expiredAttachments, nil)
	mockS3.On("DeleteFile", mock.Anything, "board/boards/workspace1/2024/01/file1.jpg").Return(nil)
	mockS3.On("DeleteFile", mock.Anything, "board/comments/workspace1/2024/01/file2.pdf").Return(nil)
	mockRepo.On("DeleteBatch", mock.Anything, []uuid.UUID{attachment1.ID, attachment2.ID}).Return(nil)
	
	// Execute
	job.Run()
	
	// Assert
	mockRepo.AssertExpectations(t)
	mockS3.AssertExpectations(t)
}

func TestCleanupJob_Run_NoExpiredFiles(t *testing.T) {
	// Setup
	mockRepo := new(MockAttachmentRepository)
	mockS3 := new(MockS3Client)
	logger := zap.NewNop()
	
	job := NewCleanupJob(mockRepo, mockS3, logger)
	
	// Mock expectations - no expired attachments
	mockRepo.On("FindExpiredTempAttachments", mock.Anything).Return([]*domain.Attachment{}, nil)
	
	// Execute
	job.Run()
	
	// Assert
	mockRepo.AssertExpectations(t)
	mockS3.AssertNotCalled(t, "DeleteFile")
	mockRepo.AssertNotCalled(t, "DeleteBatch")
}

func TestCleanupJob_Run_NonExpiredFilesNotDeleted(t *testing.T) {
	// Setup
	mockRepo := new(MockAttachmentRepository)
	mockS3 := new(MockS3Client)
	logger := zap.NewNop()
	
	job := NewCleanupJob(mockRepo, mockS3, logger)
	
	// Mock expectations - repository should not return non-expired attachments
	// This simulates the repository correctly filtering by expiration
	mockRepo.On("FindExpiredTempAttachments", mock.Anything).Return([]*domain.Attachment{}, nil)
	
	// Execute
	job.Run()
	
	// Assert - S3 delete and batch delete should not be called
	mockRepo.AssertExpectations(t)
	mockS3.AssertNotCalled(t, "DeleteFile")
	mockRepo.AssertNotCalled(t, "DeleteBatch")
}

func TestCleanupJob_Run_S3DeleteFailure(t *testing.T) {
	// Setup
	mockRepo := new(MockAttachmentRepository)
	mockS3 := new(MockS3Client)
	logger := zap.NewNop()
	
	job := NewCleanupJob(mockRepo, mockS3, logger)
	
	// Create expired attachments
	now := time.Now()
	expiredTime := now.Add(-2 * time.Hour)
	
	attachment1 := &domain.Attachment{
		BaseModel: domain.BaseModel{
			ID: uuid.New(),
		},
		EntityType:  domain.EntityTypeBoard,
		Status:      domain.AttachmentStatusTemp,
		FileName:    "test1.jpg",
		FileURL:     "https://bucket.s3.region.amazonaws.com/board/boards/workspace1/2024/01/file1.jpg",
		FileSize:    1024,
		ContentType: "image/jpeg",
		UploadedBy:  uuid.New(),
		ExpiresAt:   &expiredTime,
	}
	
	attachment2 := &domain.Attachment{
		BaseModel: domain.BaseModel{
			ID: uuid.New(),
		},
		EntityType:  domain.EntityTypeComment,
		Status:      domain.AttachmentStatusTemp,
		FileName:    "test2.pdf",
		FileURL:     "https://bucket.s3.region.amazonaws.com/board/comments/workspace1/2024/01/file2.pdf",
		FileSize:    2048,
		ContentType: "application/pdf",
		UploadedBy:  uuid.New(),
		ExpiresAt:   &expiredTime,
	}
	
	expiredAttachments := []*domain.Attachment{attachment1, attachment2}
	
	// Mock expectations - first S3 delete fails, second succeeds
	mockRepo.On("FindExpiredTempAttachments", mock.Anything).Return(expiredAttachments, nil)
	mockS3.On("DeleteFile", mock.Anything, "board/boards/workspace1/2024/01/file1.jpg").Return(errors.New("S3 error"))
	mockS3.On("DeleteFile", mock.Anything, "board/comments/workspace1/2024/01/file2.pdf").Return(nil)
	
	// Only the second attachment should be deleted from DB
	mockRepo.On("DeleteBatch", mock.Anything, []uuid.UUID{attachment2.ID}).Return(nil)
	
	// Execute
	job.Run()
	
	// Assert
	mockRepo.AssertExpectations(t)
	mockS3.AssertExpectations(t)
}

func TestCleanupJob_Run_RepositoryFindError(t *testing.T) {
	// Setup
	mockRepo := new(MockAttachmentRepository)
	mockS3 := new(MockS3Client)
	logger := zap.NewNop()
	
	job := NewCleanupJob(mockRepo, mockS3, logger)
	
	// Mock expectations - repository returns error
	mockRepo.On("FindExpiredTempAttachments", mock.Anything).Return(nil, errors.New("database error"))
	
	// Execute
	job.Run()
	
	// Assert - should handle error gracefully
	mockRepo.AssertExpectations(t)
	mockS3.AssertNotCalled(t, "DeleteFile")
	mockRepo.AssertNotCalled(t, "DeleteBatch")
}

func TestCleanupJob_Run_DatabaseDeleteError(t *testing.T) {
	// Setup
	mockRepo := new(MockAttachmentRepository)
	mockS3 := new(MockS3Client)
	logger := zap.NewNop()
	
	job := NewCleanupJob(mockRepo, mockS3, logger)
	
	// Create expired attachment
	now := time.Now()
	expiredTime := now.Add(-2 * time.Hour)
	
	attachment := &domain.Attachment{
		BaseModel: domain.BaseModel{
			ID: uuid.New(),
		},
		EntityType:  domain.EntityTypeBoard,
		Status:      domain.AttachmentStatusTemp,
		FileName:    "test.jpg",
		FileURL:     "https://bucket.s3.region.amazonaws.com/board/boards/workspace1/2024/01/file.jpg",
		FileSize:    1024,
		ContentType: "image/jpeg",
		UploadedBy:  uuid.New(),
		ExpiresAt:   &expiredTime,
	}
	
	expiredAttachments := []*domain.Attachment{attachment}
	
	// Mock expectations - S3 delete succeeds but DB delete fails
	mockRepo.On("FindExpiredTempAttachments", mock.Anything).Return(expiredAttachments, nil)
	mockS3.On("DeleteFile", mock.Anything, "board/boards/workspace1/2024/01/file.jpg").Return(nil)
	mockRepo.On("DeleteBatch", mock.Anything, []uuid.UUID{attachment.ID}).Return(errors.New("database error"))
	
	// Execute
	job.Run()
	
	// Assert - should handle error gracefully
	mockRepo.AssertExpectations(t)
	mockS3.AssertExpectations(t)
}

func TestCleanupJob_ExtractFileKeyFromURL(t *testing.T) {
	// Setup
	mockRepo := new(MockAttachmentRepository)
	mockS3 := new(MockS3Client)
	logger := zap.NewNop()
	
	job := NewCleanupJob(mockRepo, mockS3, logger)
	
	tests := []struct {
		name     string
		fileURL  string
		expected string
	}{
		{
			name:     "Standard S3 URL",
			fileURL:  "https://bucket.s3.region.amazonaws.com/board/boards/workspace1/2024/01/file.jpg",
			expected: "board/boards/workspace1/2024/01/file.jpg",
		},
		{
			name:     "S3 URL with subdirectories",
			fileURL:  "https://bucket.s3.region.amazonaws.com/board/comments/workspace2/2024/12/test.pdf",
			expected: "board/comments/workspace2/2024/12/test.pdf",
		},
		{
			name:     "Invalid URL format",
			fileURL:  "invalid-url",
			expected: "",
		},
		{
			name:     "URL without key",
			fileURL:  "https://bucket.s3.region.amazonaws.com/",
			expected: "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := job.extractFileKeyFromURL(tt.fileURL)
			assert.Equal(t, tt.expected, result)
		})
	}
}
