package service

import (
	"context"
	"errors"
	"project-board-api/internal/client"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
	"project-board-api/internal/response"
)

func (s *projectServiceImpl) GetProjectInitSettings(ctx context.Context, projectID, userID uuid.UUID, token string) (*dto.ProjectInitSettingsResponse, error) {
	// Fetch project from repository
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError("Project not found", "")
		}
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch project", err.Error())
	}

	// Check if user is a project member
	isMember, err := s.projectRepo.IsProjectMember(ctx, projectID, userID)
	if err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to check membership", err.Error())
	}

	// If not a project member, check workspace membership
	// TODO: 향후 프로젝트별 권한 관리 기능 구현 시 수정 필요
	if !isMember {
		s.logger.Debug("User is not a project member, checking workspace membership for init settings",
			zap.String("project_id", projectID.String()),
			zap.String("workspace_id", project.WorkspaceID.String()),
			zap.String("user_id", userID.String()),
		)

		isWorkspaceMember, err := s.userClient.ValidateWorkspaceMember(ctx, project.WorkspaceID, userID, token)
		if err != nil {
			s.logger.Error("Failed to validate workspace membership for init settings",
				zap.Error(err),
				zap.String("project_id", projectID.String()),
				zap.String("workspace_id", project.WorkspaceID.String()),
				zap.String("user_id", userID.String()),
			)
			return nil, response.NewForbiddenError("You are not a member of this project or workspace", "")
		}

		if !isWorkspaceMember {
			s.logger.Warn("Access denied to init settings: user is neither project member nor workspace member",
				zap.String("project_id", projectID.String()),
				zap.String("workspace_id", project.WorkspaceID.String()),
				zap.String("user_id", userID.String()),
			)
			return nil, response.NewForbiddenError("You are not a member of this project or workspace", "")
		}

		// Workspace member access granted - log for future audit and permission management
		// Note: This allows workspace members to access all projects in their workspace
		// until project-level permission management is implemented
		s.logger.Info("Access granted to init settings via workspace membership",
			zap.String("access_type", "workspace_member"),
			zap.String("project_id", projectID.String()),
			zap.String("workspace_id", project.WorkspaceID.String()),
			zap.String("user_id", userID.String()),
			zap.String("project_name", project.Name),
			zap.String("note", "Project-level permissions not yet implemented"),
		)
	} else {
		s.logger.Debug("Access granted to init settings via project membership",
			zap.String("access_type", "project_member"),
			zap.String("project_id", projectID.String()),
			zap.String("user_id", userID.String()),
		)
	}

	// Fetch workspace information
	workspace, err := s.userClient.GetWorkspace(ctx, project.WorkspaceID, token)
	if err != nil {
		// Log error but continue with graceful degradation
		workspace = &client.Workspace{
			ID:   project.WorkspaceID,
			Name: "",
		}
	}

	// Fetch owner profile information
	ownerProfile, err := s.userClient.GetWorkspaceProfile(ctx, project.WorkspaceID, project.OwnerID, token)
	if err != nil {
		// Log error but continue with graceful degradation
		s.logger.Warn("Failed to fetch owner profile for project init settings",
			zap.Error(err),
			zap.String("project_id", projectID.String()),
			zap.String("owner_id", project.OwnerID.String()),
		)
	}

	// Build project basic info with workspace and owner details
	projectInfo := dto.ProjectBasicInfo{
		ProjectID:      project.ID,
		WorkspaceID:    project.WorkspaceID,
		WorkspaceName:  workspace.Name,
		WorkspaceEmail: workspace.OwnerEmail,
		Name:           project.Name,
		Description:    project.Description,
		OwnerID:        project.OwnerID,
		IsPublic:       project.IsPublic,
		StartDate:      project.StartDate,
		DueDate:        project.DueDate,
		CreatedAt:      project.CreatedAt,
		UpdatedAt:      project.UpdatedAt,
	}

	// Add owner profile information if available
	if ownerProfile != nil {
		projectInfo.OwnerEmail = ownerProfile.Email
		projectInfo.OwnerName = ownerProfile.NickName
	}

	// Fetch project-specific field options from database
	stageOptions, err := s.fieldOptionRepo.FindByProjectAndFieldType(ctx, projectID, domain.FieldTypeStage)
	if err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch stage options", err.Error())
	}

	roleOptions, err := s.fieldOptionRepo.FindByProjectAndFieldType(ctx, projectID, domain.FieldTypeRole)
	if err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch role options", err.Error())
	}

	importanceOptions, err := s.fieldOptionRepo.FindByProjectAndFieldType(ctx, projectID, domain.FieldTypeImportance)
	if err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch importance options", err.Error())
	}

	// Convert field options to DTO format
	stageFieldOptions := make([]dto.FieldOption, len(stageOptions))
	for i, opt := range stageOptions {
		stageFieldOptions[i] = dto.FieldOption{
			OptionID:     opt.ID.String(),
			OptionLabel:  opt.Label,
			OptionValue:  opt.Value,
			Color:        opt.Color,
			DisplayOrder: opt.DisplayOrder,
			FieldID:      "stage",
		}
	}

	roleFieldOptions := make([]dto.FieldOption, len(roleOptions))
	for i, opt := range roleOptions {
		roleFieldOptions[i] = dto.FieldOption{
			OptionID:     opt.ID.String(),
			OptionLabel:  opt.Label,
			OptionValue:  opt.Value,
			Color:        opt.Color,
			DisplayOrder: opt.DisplayOrder,
			FieldID:      "role",
		}
	}

	importanceFieldOptions := make([]dto.FieldOption, len(importanceOptions))
	for i, opt := range importanceOptions {
		importanceFieldOptions[i] = dto.FieldOption{
			OptionID:     opt.ID.String(),
			OptionLabel:  opt.Label,
			OptionValue:  opt.Value,
			Color:        opt.Color,
			DisplayOrder: opt.DisplayOrder,
			FieldID:      "importance",
		}
	}

	// Define field definitions with options from database
	fields := []dto.FieldWithOptionsResponse{
		{
			FieldID:     "stage",
			FieldName:   "Stage",
			FieldType:   "select",
			IsRequired:  true,
			Description: "Current stage of the board",
			Options:     stageFieldOptions,
		},
		{
			FieldID:     "importance",
			FieldName:   "Importance",
			FieldType:   "select",
			IsRequired:  true,
			Description: "Priority level of the board",
			Options:     importanceFieldOptions,
		},
		{
			FieldID:     "role",
			FieldName:   "Role",
			FieldType:   "select",
			IsRequired:  true,
			Description: "Role responsible for the board",
			Options:     roleFieldOptions,
		},
	}

	// Define field types
	fieldTypes := []dto.FieldTypeInfo{
		{
			TypeID:      "select",
			TypeName:    "Select",
			Description: "Single selection from predefined options",
		},
		{
			TypeID:      "text",
			TypeName:    "Text",
			Description: "Free text input",
		},
		{
			TypeID:      "date",
			TypeName:    "Date",
			Description: "Date selection",
		},
		{
			TypeID:      "user",
			TypeName:    "User",
			Description: "User selection",
		},
	}

	return &dto.ProjectInitSettingsResponse{
		Project:       projectInfo,
		Fields:        fields,
		FieldTypes:    fieldTypes,
		DefaultViewID: nil, // Can be extended later to support custom views
	}, nil
}

// validateProjectDateRange validates that startDate is not after dueDate
