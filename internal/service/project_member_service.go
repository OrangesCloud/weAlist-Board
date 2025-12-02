package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"project-board-api/internal/client"
	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
	"project-board-api/internal/repository"
	"project-board-api/internal/response"
)

// ProjectMemberService defines the interface for project member business logic
type ProjectMemberService interface {
	GetMembers(ctx context.Context, projectID, userID uuid.UUID, token string) ([]*dto.ProjectMemberResponse, error)
	RemoveMember(ctx context.Context, projectID, requesterID, memberID uuid.UUID) error
	UpdateMemberRole(ctx context.Context, projectID, requesterID, memberID uuid.UUID, role string) (*dto.ProjectMemberResponse, error)
}

// projectMemberServiceImpl is the implementation of ProjectMemberService
type projectMemberServiceImpl struct {
	projectRepo repository.ProjectRepository
	userClient  client.UserClient
}

// NewProjectMemberService creates a new instance of ProjectMemberService
func NewProjectMemberService(projectRepo repository.ProjectRepository, userClient client.UserClient) ProjectMemberService {
	return &projectMemberServiceImpl{
		projectRepo: projectRepo,
		userClient:  userClient,
	}
}

// GetMembers retrieves all members of a project with user profile information
func (s *projectMemberServiceImpl) GetMembers(ctx context.Context, projectID, userID uuid.UUID, token string) ([]*dto.ProjectMemberResponse, error) {
	// Check if requester is a project member
	isMember, err := s.projectRepo.IsProjectMember(ctx, projectID, userID)
	if err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to check membership", err.Error())
	}
	if !isMember {
		return nil, response.NewForbiddenError("You are not a member of this project", "")
	}

	// Fetch project to get workspace ID
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError("Project not found", "")
		}
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch project", err.Error())
	}

	// Fetch members from repository
	members, err := s.projectRepo.FindMembersByProjectID(ctx, projectID)
	if err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch members", err.Error())
	}

	// Convert to response DTOs with user profile information
	responses := make([]*dto.ProjectMemberResponse, len(members))
	for i, member := range members {
		responses[i] = &dto.ProjectMemberResponse{
			MemberID:  member.ID,
			ProjectID: member.ProjectID,
			UserID:    member.UserID,
			RoleName:  string(member.RoleName),
			JoinedAt:  member.JoinedAt,
		}

		// Fetch workspace profile for member
		profile, err := s.userClient.GetWorkspaceProfile(ctx, project.WorkspaceID, member.UserID, token)
		if err == nil && profile != nil {
			responses[i].UserEmail = profile.Email
			responses[i].UserName = profile.NickName
		}
		// Graceful degradation: if profile fetch fails, continue without user details
	}

	return responses, nil
}

// RemoveMember removes a member from a project with authorization checks
func (s *projectMemberServiceImpl) RemoveMember(ctx context.Context, projectID, requesterID, memberID uuid.UUID) error {
	// Check if requester is OWNER or ADMIN
	requesterMember, err := s.projectRepo.FindMemberByProjectAndUser(ctx, projectID, requesterID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewForbiddenError("You are not a member of this project", "")
		}
		return response.NewAppError(response.ErrCodeInternal, "Failed to check membership", err.Error())
	}
	if requesterMember.RoleName != domain.ProjectRoleOwner && requesterMember.RoleName != domain.ProjectRoleAdmin {
		return response.NewForbiddenError("Only project owner or admin can remove members", "")
	}

	// Fetch the member to be removed
	targetMember, err := s.projectRepo.FindMemberByProjectAndUser(ctx, projectID, memberID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError("Member not found", "")
		}
		return response.NewAppError(response.ErrCodeInternal, "Failed to fetch member", err.Error())
	}

	// Cannot remove OWNER
	if targetMember.RoleName == domain.ProjectRoleOwner {
		return response.NewValidationError("Cannot remove project owner", "")
	}

	// Cannot remove self
	if requesterID == memberID {
		return response.NewValidationError("Cannot remove yourself from the project", "")
	}

	// Remove member from repository
	if err := s.projectRepo.RemoveMember(ctx, targetMember.ID); err != nil {
		return response.NewAppError(response.ErrCodeInternal, "Failed to remove member", err.Error())
	}

	return nil
}

// UpdateMemberRole updates a member's role with authorization checks
func (s *projectMemberServiceImpl) UpdateMemberRole(ctx context.Context, projectID, requesterID, memberID uuid.UUID, role string) (*dto.ProjectMemberResponse, error) {
	// Check if requester is OWNER
	requesterMember, err := s.projectRepo.FindMemberByProjectAndUser(ctx, projectID, requesterID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewForbiddenError("You are not a member of this project", "")
		}
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to check membership", err.Error())
	}
	if requesterMember.RoleName != domain.ProjectRoleOwner {
		return nil, response.NewForbiddenError("Only project owner can change member roles", "")
	}

	// Validate role
	projectRole := domain.ProjectRole(role)
	if projectRole != domain.ProjectRoleOwner && projectRole != domain.ProjectRoleAdmin && projectRole != domain.ProjectRoleMember {
		return nil, response.NewValidationError("Invalid role", "")
	}

	// Fetch the member to be updated
	targetMember, err := s.projectRepo.FindMemberByProjectAndUser(ctx, projectID, memberID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError("Member not found", "")
		}
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch member", err.Error())
	}

	// Cannot change OWNER role
	if targetMember.RoleName == domain.ProjectRoleOwner {
		return nil, response.NewValidationError("Cannot change project owner role", "")
	}

	// Update member role in repository
	if err := s.projectRepo.UpdateMemberRole(ctx, targetMember.ID, projectRole); err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to update member role", err.Error())
	}

	// Fetch updated member
	updatedMember, err := s.projectRepo.FindMemberByProjectAndUser(ctx, projectID, memberID)
	if err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch updated member", err.Error())
	}

	// Convert to response DTO
	return &dto.ProjectMemberResponse{
		MemberID:  updatedMember.ID,
		ProjectID: updatedMember.ProjectID,
		UserID:    updatedMember.UserID,
		RoleName:  string(updatedMember.RoleName),
		JoinedAt:  updatedMember.JoinedAt,
	}, nil
}
