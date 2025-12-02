package service

import (
	"context"

	"github.com/google/uuid"

	"project-board-api/internal/client"
	"project-board-api/internal/domain"
)

// MockFieldOptionRepository is a mock implementation of FieldOptionRepository
type MockFieldOptionRepository struct {
	CreateFunc                            func(ctx context.Context, fieldOption *domain.FieldOption) error
	FindByIDFunc                          func(ctx context.Context, id uuid.UUID) (*domain.FieldOption, error)
	FindByFieldTypeFunc                   func(ctx context.Context, fieldType domain.FieldType) ([]*domain.FieldOption, error)
	FindByProjectAndFieldTypeFunc         func(ctx context.Context, projectID uuid.UUID, fieldType domain.FieldType) ([]*domain.FieldOption, error)
	FindByFieldTypeAndValueFunc           func(ctx context.Context, fieldType, value string) (*domain.FieldOption, error)
	FindByProjectAndFieldTypeAndValueFunc func(ctx context.Context, projectID uuid.UUID, fieldType domain.FieldType, value string) (*domain.FieldOption, error)
	FindByIDsFunc                         func(ctx context.Context, ids []uuid.UUID) ([]*domain.FieldOption, error)
	FindSystemDefaultsFunc                func(ctx context.Context) ([]*domain.FieldOption, error)
	CreateBatchFunc                       func(ctx context.Context, fieldOptions []*domain.FieldOption) error
	UpdateFunc                            func(ctx context.Context, fieldOption *domain.FieldOption) error
	DeleteFunc                            func(ctx context.Context, id uuid.UUID) error
}

func (m *MockFieldOptionRepository) Create(ctx context.Context, fieldOption *domain.FieldOption) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, fieldOption)
	}
	return nil
}

func (m *MockFieldOptionRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.FieldOption, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockFieldOptionRepository) FindByFieldType(ctx context.Context, fieldType domain.FieldType) ([]*domain.FieldOption, error) {
	if m.FindByFieldTypeFunc != nil {
		return m.FindByFieldTypeFunc(ctx, fieldType)
	}
	return nil, nil
}

func (m *MockFieldOptionRepository) FindByFieldTypeAndValue(ctx context.Context, fieldType, value string) (*domain.FieldOption, error) {
	if m.FindByFieldTypeAndValueFunc != nil {
		return m.FindByFieldTypeAndValueFunc(ctx, fieldType, value)
	}
	return nil, nil
}

func (m *MockFieldOptionRepository) Update(ctx context.Context, fieldOption *domain.FieldOption) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, fieldOption)
	}
	return nil
}

func (m *MockFieldOptionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *MockFieldOptionRepository) FindByProjectAndFieldType(ctx context.Context, projectID uuid.UUID, fieldType domain.FieldType) ([]*domain.FieldOption, error) {
	if m.FindByProjectAndFieldTypeFunc != nil {
		return m.FindByProjectAndFieldTypeFunc(ctx, projectID, fieldType)
	}
	return nil, nil
}

func (m *MockFieldOptionRepository) FindSystemDefaults(ctx context.Context) ([]*domain.FieldOption, error) {
	if m.FindSystemDefaultsFunc != nil {
		return m.FindSystemDefaultsFunc(ctx)
	}
	return nil, nil
}

func (m *MockFieldOptionRepository) CreateBatch(ctx context.Context, fieldOptions []*domain.FieldOption) error {
	if m.CreateBatchFunc != nil {
		return m.CreateBatchFunc(ctx, fieldOptions)
	}
	return nil
}

func (m *MockFieldOptionRepository) FindByProjectAndFieldTypeAndValue(ctx context.Context, projectID uuid.UUID, fieldType domain.FieldType, value string) (*domain.FieldOption, error) {
	if m.FindByProjectAndFieldTypeAndValueFunc != nil {
		return m.FindByProjectAndFieldTypeAndValueFunc(ctx, projectID, fieldType, value)
	}
	return nil, nil
}

func (m *MockFieldOptionRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.FieldOption, error) {
	if m.FindByIDsFunc != nil {
		return m.FindByIDsFunc(ctx, ids)
	}
	return nil, nil
}

// MockFieldOptionConverter is a mock implementation of FieldOptionConverter
type MockFieldOptionConverter struct {
	ConvertValuesToIDsFunc     func(ctx context.Context, projectID uuid.UUID, customFields map[string]interface{}) (map[string]interface{}, error)
	ConvertIDsToValuesFunc     func(ctx context.Context, customFields map[string]interface{}) (map[string]interface{}, error)
	ConvertIDsToValuesBatchFunc func(ctx context.Context, boards []*domain.Board) error
}

func (m *MockFieldOptionConverter) ConvertValuesToIDs(ctx context.Context, projectID uuid.UUID, customFields map[string]interface{}) (map[string]interface{}, error) {
	if m.ConvertValuesToIDsFunc != nil {
		return m.ConvertValuesToIDsFunc(ctx, projectID, customFields)
	}
	// Default: return as-is (no conversion)
	return customFields, nil
}

func (m *MockFieldOptionConverter) ConvertIDsToValues(ctx context.Context, customFields map[string]interface{}) (map[string]interface{}, error) {
	if m.ConvertIDsToValuesFunc != nil {
		return m.ConvertIDsToValuesFunc(ctx, customFields)
	}
	// Default: return as-is (no conversion)
	return customFields, nil
}

func (m *MockFieldOptionConverter) ConvertIDsToValuesBatch(ctx context.Context, boards []*domain.Board) error {
	if m.ConvertIDsToValuesBatchFunc != nil {
		return m.ConvertIDsToValuesBatchFunc(ctx, boards)
	}
	// Default: no-op
	return nil
}

// MockUserClient is a mock implementation of UserClient
type MockUserClient struct {
	ValidateWorkspaceMemberFunc func(ctx context.Context, workspaceID, userID uuid.UUID, token string) (bool, error)
	GetUserProfileFunc          func(ctx context.Context, userID uuid.UUID, token string) (*client.UserProfile, error)
	GetWorkspaceProfileFunc     func(ctx context.Context, workspaceID, userID uuid.UUID, token string) (*client.WorkspaceProfile, error)
	GetWorkspaceFunc            func(ctx context.Context, workspaceID uuid.UUID, token string) (*client.Workspace, error)
	ValidateTokenFunc           func(ctx context.Context, token string) (uuid.UUID, error)
}

func (m *MockUserClient) ValidateWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID, token string) (bool, error) {
	if m.ValidateWorkspaceMemberFunc != nil {
		return m.ValidateWorkspaceMemberFunc(ctx, workspaceID, userID, token)
	}
	return true, nil
}

func (m *MockUserClient) GetUserProfile(ctx context.Context, userID uuid.UUID, token string) (*client.UserProfile, error) {
	if m.GetUserProfileFunc != nil {
		return m.GetUserProfileFunc(ctx, userID, token)
	}
	return &client.UserProfile{UserID: userID, Email: "test@example.com"}, nil
}

func (m *MockUserClient) GetWorkspaceProfile(ctx context.Context, workspaceID, userID uuid.UUID, token string) (*client.WorkspaceProfile, error) {
	if m.GetWorkspaceProfileFunc != nil {
		return m.GetWorkspaceProfileFunc(ctx, workspaceID, userID, token)
	}
	return &client.WorkspaceProfile{
		WorkspaceID: workspaceID,
		UserID:      userID,
		NickName:    "Test User",
		Email:       "test@example.com",
	}, nil
}

func (m *MockUserClient) GetWorkspace(ctx context.Context, workspaceID uuid.UUID, token string) (*client.Workspace, error) {
	if m.GetWorkspaceFunc != nil {
		return m.GetWorkspaceFunc(ctx, workspaceID, token)
	}
	return &client.Workspace{
		ID:         workspaceID,
		Name:       "Test Workspace",
		OwnerEmail: "workspace@example.com",
	}, nil
}

func (m *MockUserClient) ValidateToken(ctx context.Context, token string) (uuid.UUID, error) {
	if m.ValidateTokenFunc != nil {
		return m.ValidateTokenFunc(ctx, token)
	}
	return uuid.New(), nil
}

// MockAttachmentRepository is a mock implementation of AttachmentRepository
type MockAttachmentRepository struct {
	CreateFunc                     func(ctx context.Context, attachment *domain.Attachment) error
	FindByIDFunc                   func(ctx context.Context, id uuid.UUID) (*domain.Attachment, error)
	FindByEntityIDFunc             func(ctx context.Context, entityType domain.EntityType, entityID uuid.UUID) ([]*domain.Attachment, error)
	FindByIDsFunc                  func(ctx context.Context, ids []uuid.UUID) ([]*domain.Attachment, error)
	DeleteFunc                     func(ctx context.Context, id uuid.UUID) error
	FindExpiredTempAttachmentsFunc func(ctx context.Context) ([]*domain.Attachment, error)
	ConfirmAttachmentsFunc         func(ctx context.Context, attachmentIDs []uuid.UUID, entityID uuid.UUID) error
	DeleteBatchFunc                func(ctx context.Context, attachmentIDs []uuid.UUID) error
}

func (m *MockAttachmentRepository) Create(ctx context.Context, attachment *domain.Attachment) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, attachment)
	}
	return nil
}

func (m *MockAttachmentRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Attachment, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockAttachmentRepository) FindByEntityID(ctx context.Context, entityType domain.EntityType, entityID uuid.UUID) ([]*domain.Attachment, error) {
	if m.FindByEntityIDFunc != nil {
		return m.FindByEntityIDFunc(ctx, entityType, entityID)
	}
	return nil, nil
}

func (m *MockAttachmentRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.Attachment, error) {
	if m.FindByIDsFunc != nil {
		return m.FindByIDsFunc(ctx, ids)
	}
	return nil, nil
}

func (m *MockAttachmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *MockAttachmentRepository) FindExpiredTempAttachments(ctx context.Context) ([]*domain.Attachment, error) {
	if m.FindExpiredTempAttachmentsFunc != nil {
		return m.FindExpiredTempAttachmentsFunc(ctx)
	}
	return nil, nil
}

func (m *MockAttachmentRepository) ConfirmAttachments(ctx context.Context, attachmentIDs []uuid.UUID, entityID uuid.UUID) error {
	if m.ConfirmAttachmentsFunc != nil {
		return m.ConfirmAttachmentsFunc(ctx, attachmentIDs, entityID)
	}
	return nil
}

func (m *MockAttachmentRepository) DeleteBatch(ctx context.Context, attachmentIDs []uuid.UUID) error {
	if m.DeleteBatchFunc != nil {
		return m.DeleteBatchFunc(ctx, attachmentIDs)
	}
	return nil
}
