package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"project-board-api/internal/client"
	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
	"project-board-api/internal/repository"
	"project-board-api/internal/response"
)

// ProjectJoinRequestService defines the interface for project join request business logic
type ProjectJoinRequestService interface {
	CreateJoinRequest(ctx context.Context, projectID, userID uuid.UUID, token string) (*dto.ProjectJoinRequestResponse, error)
	GetJoinRequests(ctx context.Context, projectID, userID uuid.UUID, status *string, token string) ([]*dto.ProjectJoinRequestResponse, error)
	UpdateJoinRequest(ctx context.Context, requestID, userID uuid.UUID, status string, token string) (*dto.ProjectJoinRequestResponse, error)
}

// projectJoinRequestServiceImpl is the implementation of ProjectJoinRequestService
type projectJoinRequestServiceImpl struct {
	projectRepo repository.ProjectRepository
	userClient  client.UserClient
}

// NewProjectJoinRequestService creates a new instance of ProjectJoinRequestService
func NewProjectJoinRequestService(projectRepo repository.ProjectRepository, userClient client.UserClient) ProjectJoinRequestService {
	return &projectJoinRequestServiceImpl{
		projectRepo: projectRepo,
		userClient:  userClient,
	}
}

// CreateJoinRequest creates a new join request with duplicate validation
func (s *projectJoinRequestServiceImpl) CreateJoinRequest(ctx context.Context, projectID, userID uuid.UUID, token string) (*dto.ProjectJoinRequestResponse, error) {
	// Fetch project to get workspace ID
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError("Project not found", "")
		}
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch project", err.Error())
	}

	// Validate workspace membership
	isValid, err := s.userClient.ValidateWorkspaceMember(ctx, project.WorkspaceID, userID, token)
	if err != nil {
		return nil, response.NewAppError(response.ErrCodeForbidden, "You are not a member of this workspace", "")
	}
	if !isValid {
		return nil, response.NewAppError(response.ErrCodeForbidden, "You are not a member of this workspace", "")
	}

	// Check if user is already a project member
	isMember, err := s.projectRepo.IsProjectMember(ctx, projectID, userID)
	if err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to check membership", err.Error())
	}
	if isMember {
		return nil, response.NewAppError("ALREADY_MEMBER", "User is already a member of this project", "")
	}

	// Check if user already has a pending join request
	pendingRequest, err := s.projectRepo.FindPendingByProjectAndUser(ctx, projectID, userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to check pending requests", err.Error())
	}
	if pendingRequest != nil {
		return nil, response.NewAppError("PENDING_REQUEST_EXISTS", "User already has a pending join request for this project", "")
	}

	// Create join request
	joinRequest := &domain.ProjectJoinRequest{
		ProjectID:   projectID,
		UserID:      userID,
		Status:      domain.JoinRequestPending,
		RequestedAt: time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.projectRepo.CreateJoinRequest(ctx, joinRequest); err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to create join request", err.Error())
	}

	// Convert to response DTO
	return &dto.ProjectJoinRequestResponse{
		RequestID:   joinRequest.ID,
		ProjectID:   joinRequest.ProjectID,
		UserID:      joinRequest.UserID,
		Status:      string(joinRequest.Status),
		RequestedAt: joinRequest.RequestedAt,
		UpdatedAt:   joinRequest.UpdatedAt,
	}, nil
}

// GetJoinRequests retrieves join requests for a project with authorization checks
func (s *projectJoinRequestServiceImpl) GetJoinRequests(ctx context.Context, projectID, userID uuid.UUID, status *string, token string) ([]*dto.ProjectJoinRequestResponse, error) {
	// Check if requester is OWNER or ADMIN
	requesterMember, err := s.projectRepo.FindMemberByProjectAndUser(ctx, projectID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewForbiddenError("You are not a member of this project", "")
		}
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to check membership", err.Error())
	}
	if requesterMember.RoleName != domain.ProjectRoleOwner && requesterMember.RoleName != domain.ProjectRoleAdmin {
		return nil, response.NewForbiddenError("Only project owner or admin can view join requests", "")
	}

	// Fetch project to get workspace ID
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError("Project not found", "")
		}
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch project", err.Error())
	}

	// Convert status string to domain type if provided
	var statusFilter *domain.ProjectJoinRequestStatus
	if status != nil {
		s := domain.ProjectJoinRequestStatus(*status)
		statusFilter = &s
	}

	// Fetch join requests from repository
	requests, err := s.projectRepo.FindJoinRequestsByProjectID(ctx, projectID, statusFilter)
	if err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch join requests", err.Error())
	}

	// Convert to response DTOs with user profile information
	responses := make([]*dto.ProjectJoinRequestResponse, len(requests))
	for i, request := range requests {
		responses[i] = &dto.ProjectJoinRequestResponse{
			RequestID:   request.ID,
			ProjectID:   request.ProjectID,
			UserID:      request.UserID,
			Status:      string(request.Status),
			RequestedAt: request.RequestedAt,
			UpdatedAt:   request.UpdatedAt,
		}

		// Fetch workspace profile for requester
		profile, err := s.userClient.GetWorkspaceProfile(ctx, project.WorkspaceID, request.UserID, token)
		if err == nil && profile != nil {
			responses[i].UserEmail = profile.Email
			responses[i].UserName = profile.NickName
		}
		// Graceful degradation: if profile fetch fails, continue without user details
	}

	return responses, nil
}

// UpdateJoinRequest updates a join request status with automatic member addition on approval
func (s *projectJoinRequestServiceImpl) UpdateJoinRequest(ctx context.Context, requestID, userID uuid.UUID, status string, token string) (*dto.ProjectJoinRequestResponse, error) {
	// Validate status
	requestStatus := domain.ProjectJoinRequestStatus(status)
	if requestStatus != domain.JoinRequestApproved && requestStatus != domain.JoinRequestRejected {
		return nil, response.NewValidationError("Invalid status", "")
	}

	// Fetch join request
	joinRequest, err := s.projectRepo.FindJoinRequestByID(ctx, requestID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError("Join request not found", "")
		}
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch join request", err.Error())
	}

	// Check if request is already processed
	if joinRequest.Status != domain.JoinRequestPending {
		return nil, response.NewValidationError("Join request has already been processed", "")
	}

	// Check if requester is OWNER or ADMIN
	requesterMember, err := s.projectRepo.FindMemberByProjectAndUser(ctx, joinRequest.ProjectID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewForbiddenError("You are not a member of this project", "")
		}
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to check membership", err.Error())
	}
	if requesterMember.RoleName != domain.ProjectRoleOwner && requesterMember.RoleName != domain.ProjectRoleAdmin {
		return nil, response.NewForbiddenError("Only project owner or admin can update join requests", "")
	}

	// Update join request status
	if err := s.projectRepo.UpdateJoinRequestStatus(ctx, requestID, requestStatus); err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to update join request", err.Error())
	}

	// If approved, add user as project member
	if requestStatus == domain.JoinRequestApproved {
		member := &domain.ProjectMember{
			ProjectID: joinRequest.ProjectID,
			UserID:    joinRequest.UserID,
			RoleName:  domain.ProjectRoleMember,
			JoinedAt:  time.Now(),
		}
		if err := s.projectRepo.AddMember(ctx, member); err != nil {
			return nil, response.NewAppError(response.ErrCodeInternal, "Failed to add member", err.Error())
		}
	}

	// Fetch updated join request
	updatedRequest, err := s.projectRepo.FindJoinRequestByID(ctx, requestID)
	if err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch updated join request", err.Error())
	}

	// Convert to response DTO
	return &dto.ProjectJoinRequestResponse{
		RequestID:   updatedRequest.ID,
		ProjectID:   updatedRequest.ProjectID,
		UserID:      updatedRequest.UserID,
		Status:      string(updatedRequest.Status),
		RequestedAt: updatedRequest.RequestedAt,
		UpdatedAt:   updatedRequest.UpdatedAt,
	}, nil
}
