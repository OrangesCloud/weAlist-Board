package service

import (
	"context"
	"io"

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
	ConvertValuesToIDsFunc      func(ctx context.Context, projectID uuid.UUID, customFields map[string]interface{}) (map[string]interface{}, error)
	ConvertIDsToValuesFunc      func(ctx context.Context, customFields map[string]interface{}) (map[string]interface{}, error)
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

// MockS3Client is a mock implementation of S3Client
type MockS3Client struct {
	GenerateFileKeyFunc      func(entityType, workspaceID, fileExt string) (string, error)
	GeneratePresignedURLFunc func(ctx context.Context, entityType, workspaceID, fileName, contentType string) (string, string, error)
	UploadFileFunc           func(ctx context.Context, key string, file io.Reader, contentType string) (string, error)
	DeleteFileFunc           func(ctx context.Context, key string) error
	GetFileURLFunc           func(key string) string
}

func (m *MockS3Client) GenerateFileKey(entityType, workspaceID, fileExt string) (string, error) {
	if m.GenerateFileKeyFunc != nil {
		return m.GenerateFileKeyFunc(entityType, workspaceID, fileExt)
	}
	return "mock-file-key" + fileExt, nil
}

func (m *MockS3Client) GeneratePresignedURL(ctx context.Context, entityType, workspaceID, fileName, contentType string) (string, string, error) {
	if m.GeneratePresignedURLFunc != nil {
		return m.GeneratePresignedURLFunc(ctx, entityType, workspaceID, fileName, contentType)
	}
	return "https://mock-presigned-url.com", "mock-file-key", nil
}

func (m *MockS3Client) UploadFile(ctx context.Context, key string, file io.Reader, contentType string) (string, error) {
	if m.UploadFileFunc != nil {
		return m.UploadFileFunc(ctx, key, file, contentType)
	}
	return "https://mock-s3-url.com/" + key, nil
}

func (m *MockS3Client) DeleteFile(ctx context.Context, key string) error {
	if m.DeleteFileFunc != nil {
		return m.DeleteFileFunc(ctx, key)
	}
	return nil
}

func (m *MockS3Client) GetFileURL(key string) string {
	if m.GetFileURLFunc != nil {
		return m.GetFileURLFunc(key)
	}
	return "https://mock-s3-url.com/" + key
}

// MockBoardRepository is a mock implementation of BoardRepository
type MockBoardRepository struct {
	CreateFunc          func(ctx context.Context, board *domain.Board) error
	FindByIDFunc        func(ctx context.Context, id uuid.UUID) (*domain.Board, error)
	FindByProjectIDFunc func(ctx context.Context, projectID uuid.UUID, filters interface{}) ([]*domain.Board, error)
	UpdateFunc          func(ctx context.Context, board *domain.Board) error
	DeleteFunc          func(ctx context.Context, id uuid.UUID) error
}

func (m *MockBoardRepository) Create(ctx context.Context, board *domain.Board) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, board)
	}
	return nil
}

func (m *MockBoardRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockBoardRepository) FindByProjectID(ctx context.Context, projectID uuid.UUID, filters interface{}) ([]*domain.Board, error) {
	if m.FindByProjectIDFunc != nil {
		return m.FindByProjectIDFunc(ctx, projectID, filters)
	}
	return nil, nil
}

func (m *MockBoardRepository) Update(ctx context.Context, board *domain.Board) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, board)
	}
	return nil
}

func (m *MockBoardRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

// MockProjectRepository is a mock implementation of ProjectRepository
type MockProjectRepository struct {
	CreateFunc                      func(ctx context.Context, project *domain.Project) error
	FindByIDFunc                    func(ctx context.Context, id uuid.UUID) (*domain.Project, error)
	FindByWorkspaceIDFunc           func(ctx context.Context, workspaceID uuid.UUID) ([]*domain.Project, error)
	FindDefaultByWorkspaceIDFunc    func(ctx context.Context, workspaceID uuid.UUID) (*domain.Project, error)
	UpdateFunc                      func(ctx context.Context, project *domain.Project) error
	DeleteFunc                      func(ctx context.Context, id uuid.UUID) error
	SearchFunc                      func(ctx context.Context, workspaceID uuid.UUID, query string, page, limit int) ([]*domain.Project, int64, error)
	AddMemberFunc                   func(ctx context.Context, member *domain.ProjectMember) error
	FindMemberByProjectAndUserFunc  func(ctx context.Context, projectID, userID uuid.UUID) (*domain.ProjectMember, error)
	RemoveMemberFunc                func(ctx context.Context, memberID uuid.UUID) error
	UpdateMemberRoleFunc            func(ctx context.Context, memberID uuid.UUID, role domain.ProjectRole) error
	IsProjectMemberFunc             func(ctx context.Context, projectID, userID uuid.UUID) (bool, error)
	FindMembersByProjectIDFunc      func(ctx context.Context, projectID uuid.UUID) ([]*domain.ProjectMember, error)
	CreateJoinRequestFunc           func(ctx context.Context, request *domain.ProjectJoinRequest) error
	FindJoinRequestByIDFunc         func(ctx context.Context, id uuid.UUID) (*domain.ProjectJoinRequest, error)
	FindJoinRequestsByProjectIDFunc func(ctx context.Context, projectID uuid.UUID, status *domain.ProjectJoinRequestStatus) ([]*domain.ProjectJoinRequest, error)
	FindPendingByProjectAndUserFunc func(ctx context.Context, projectID, userID uuid.UUID) (*domain.ProjectJoinRequest, error)
	UpdateJoinRequestStatusFunc     func(ctx context.Context, id uuid.UUID, status domain.ProjectJoinRequestStatus) error
}

func (m *MockProjectRepository) Create(ctx context.Context, project *domain.Project) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, project)
	}
	return nil
}

func (m *MockProjectRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockProjectRepository) FindByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) ([]*domain.Project, error) {
	if m.FindByWorkspaceIDFunc != nil {
		return m.FindByWorkspaceIDFunc(ctx, workspaceID)
	}
	return nil, nil
}

func (m *MockProjectRepository) FindDefaultByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) (*domain.Project, error) {
	if m.FindDefaultByWorkspaceIDFunc != nil {
		return m.FindDefaultByWorkspaceIDFunc(ctx, workspaceID)
	}
	return nil, nil
}

func (m *MockProjectRepository) Update(ctx context.Context, project *domain.Project) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, project)
	}
	return nil
}

func (m *MockProjectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *MockProjectRepository) Search(ctx context.Context, workspaceID uuid.UUID, query string, page, limit int) ([]*domain.Project, int64, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, workspaceID, query, page, limit)
	}
	return nil, 0, nil
}

func (m *MockProjectRepository) AddMember(ctx context.Context, member *domain.ProjectMember) error {
	if m.AddMemberFunc != nil {
		return m.AddMemberFunc(ctx, member)
	}
	return nil
}

func (m *MockProjectRepository) FindMembersByProjectID(ctx context.Context, projectID uuid.UUID) ([]*domain.ProjectMember, error) {
	if m.FindMembersByProjectIDFunc != nil {
		return m.FindMembersByProjectIDFunc(ctx, projectID)
	}
	return nil, nil
}

func (m *MockProjectRepository) FindMemberByProjectAndUser(ctx context.Context, projectID, userID uuid.UUID) (*domain.ProjectMember, error) {
	if m.FindMemberByProjectAndUserFunc != nil {
		return m.FindMemberByProjectAndUserFunc(ctx, projectID, userID)
	}
	return nil, nil
}

func (m *MockProjectRepository) RemoveMember(ctx context.Context, memberID uuid.UUID) error {
	if m.RemoveMemberFunc != nil {
		return m.RemoveMemberFunc(ctx, memberID)
	}
	return nil
}

func (m *MockProjectRepository) UpdateMemberRole(ctx context.Context, memberID uuid.UUID, role domain.ProjectRole) error {
	if m.UpdateMemberRoleFunc != nil {
		return m.UpdateMemberRoleFunc(ctx, memberID, role)
	}
	return nil
}

func (m *MockProjectRepository) IsProjectMember(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	if m.IsProjectMemberFunc != nil {
		return m.IsProjectMemberFunc(ctx, projectID, userID)
	}
	return false, nil
}

func (m *MockProjectRepository) CreateJoinRequest(ctx context.Context, request *domain.ProjectJoinRequest) error {
	if m.CreateJoinRequestFunc != nil {
		return m.CreateJoinRequestFunc(ctx, request)
	}
	return nil
}

func (m *MockProjectRepository) FindJoinRequestByID(ctx context.Context, id uuid.UUID) (*domain.ProjectJoinRequest, error) {
	if m.FindJoinRequestByIDFunc != nil {
		return m.FindJoinRequestByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockProjectRepository) FindJoinRequestsByProjectID(ctx context.Context, projectID uuid.UUID, status *domain.ProjectJoinRequestStatus) ([]*domain.ProjectJoinRequest, error) {
	if m.FindJoinRequestsByProjectIDFunc != nil {
		return m.FindJoinRequestsByProjectIDFunc(ctx, projectID, status)
	}
	return nil, nil
}

func (m *MockProjectRepository) FindPendingByProjectAndUser(ctx context.Context, projectID, userID uuid.UUID) (*domain.ProjectJoinRequest, error) {
	if m.FindPendingByProjectAndUserFunc != nil {
		return m.FindPendingByProjectAndUserFunc(ctx, projectID, userID)
	}
	return nil, nil
}

func (m *MockProjectRepository) UpdateJoinRequestStatus(ctx context.Context, id uuid.UUID, status domain.ProjectJoinRequestStatus) error {
	if m.UpdateJoinRequestStatusFunc != nil {
		return m.UpdateJoinRequestStatusFunc(ctx, id, status)
	}
	return nil
}

// MockParticipantRepository is a mock implementation of ParticipantRepository
type MockParticipantRepository struct {
	CreateFunc             func(ctx context.Context, participant *domain.Participant) error
	FindByBoardIDFunc      func(ctx context.Context, boardID uuid.UUID) ([]*domain.Participant, error)
	FindByBoardAndUserFunc func(ctx context.Context, boardID, userID uuid.UUID) (*domain.Participant, error)
	DeleteFunc             func(ctx context.Context, boardID, userID uuid.UUID) error
}

func (m *MockParticipantRepository) Create(ctx context.Context, participant *domain.Participant) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, participant)
	}
	return nil
}

func (m *MockParticipantRepository) FindByBoardID(ctx context.Context, boardID uuid.UUID) ([]*domain.Participant, error) {
	if m.FindByBoardIDFunc != nil {
		return m.FindByBoardIDFunc(ctx, boardID)
	}
	return nil, nil
}

func (m *MockParticipantRepository) FindByBoardAndUser(ctx context.Context, boardID, userID uuid.UUID) (*domain.Participant, error) {
	if m.FindByBoardAndUserFunc != nil {
		return m.FindByBoardAndUserFunc(ctx, boardID, userID)
	}
	return nil, nil
}

func (m *MockParticipantRepository) Delete(ctx context.Context, boardID, userID uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, boardID, userID)
	}
	return nil
}

// MockCommentRepository is a mock implementation of CommentRepository
type MockCommentRepository struct {
	CreateFunc        func(ctx context.Context, comment *domain.Comment) error
	FindByIDFunc      func(ctx context.Context, id uuid.UUID) (*domain.Comment, error)
	FindByBoardIDFunc func(ctx context.Context, boardID uuid.UUID) ([]*domain.Comment, error)
	UpdateFunc        func(ctx context.Context, comment *domain.Comment) error
	DeleteFunc        func(ctx context.Context, id uuid.UUID) error
}

func (m *MockCommentRepository) Create(ctx context.Context, comment *domain.Comment) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, comment)
	}
	return nil
}

func (m *MockCommentRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Comment, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockCommentRepository) FindByBoardID(ctx context.Context, boardID uuid.UUID) ([]*domain.Comment, error) {
	if m.FindByBoardIDFunc != nil {
		return m.FindByBoardIDFunc(ctx, boardID)
	}
	return nil, nil
}

func (m *MockCommentRepository) Update(ctx context.Context, comment *domain.Comment) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, comment)
	}
	return nil
}

func (m *MockCommentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}
