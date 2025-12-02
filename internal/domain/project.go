package domain

import (
	"time"

	"github.com/google/uuid"
)

// Project represents a project entity within a workspace
type Project struct {
	BaseModel
	WorkspaceID  uuid.UUID            `gorm:"type:uuid;not null;index:idx_projects_workspace_id" json:"workspace_id"`
	OwnerID      uuid.UUID            `gorm:"type:uuid;not null;index:idx_projects_owner_id" json:"owner_id"`
	Name         string               `gorm:"type:varchar(255);not null" json:"name"`
	Description  string               `gorm:"type:text" json:"description"`
	StartDate    *time.Time           `gorm:"type:timestamp" json:"start_date,omitempty"`
	DueDate      *time.Time           `gorm:"type:timestamp" json:"due_date,omitempty"`
	IsDefault    bool                 `gorm:"default:false;index:idx_projects_is_default" json:"is_default"`
	IsPublic     bool                 `gorm:"default:false" json:"is_public"`
	Boards       []Board              `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE" json:"boards,omitempty"`
	Members      []ProjectMember      `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE" json:"members,omitempty"`
	JoinRequests []ProjectJoinRequest `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE" json:"join_requests,omitempty"`
	// ✅ 수정: Attachments는 다형성 관계이므로 FK 제거, Repository에서 별도 조회
	Attachments []Attachment `gorm:"-" json:"attachments,omitempty"`
}

// ProjectRole represents the role of a project member
type ProjectRole string

const (
	ProjectRoleOwner  ProjectRole = "OWNER"
	ProjectRoleAdmin  ProjectRole = "ADMIN"
	ProjectRoleMember ProjectRole = "MEMBER"
)

// ProjectMember represents a member of a project
type ProjectMember struct {
	ID        uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ProjectID uuid.UUID   `gorm:"type:uuid;not null;index:idx_project_members_project_id;uniqueIndex:uq_project_members_project_user" json:"project_id"`
	UserID    uuid.UUID   `gorm:"type:uuid;not null;index:idx_project_members_user_id;uniqueIndex:uq_project_members_project_user" json:"user_id"`
	RoleName  ProjectRole `gorm:"type:varchar(50);not null;index:idx_project_members_role" json:"role_name"`
	JoinedAt  time.Time   `gorm:"type:timestamp;not null;default:now()" json:"joined_at"`
	Project   Project     `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE" json:"project,omitempty"`
}

// ProjectJoinRequestStatus represents the status of a join request
type ProjectJoinRequestStatus string

const (
	JoinRequestPending  ProjectJoinRequestStatus = "PENDING"
	JoinRequestApproved ProjectJoinRequestStatus = "APPROVED"
	JoinRequestRejected ProjectJoinRequestStatus = "REJECTED"
)

// ProjectJoinRequest represents a request to join a project
type ProjectJoinRequest struct {
	ID          uuid.UUID                `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ProjectID   uuid.UUID                `gorm:"type:uuid;not null;index:idx_project_join_requests_project_id;index:idx_project_join_requests_project_status,priority:1" json:"project_id"`
	UserID      uuid.UUID                `gorm:"type:uuid;not null;index:idx_project_join_requests_user_id" json:"user_id"`
	Status      ProjectJoinRequestStatus `gorm:"type:varchar(50);not null;default:'PENDING';index:idx_project_join_requests_status;index:idx_project_join_requests_project_status,priority:2" json:"status"`
	RequestedAt time.Time                `gorm:"type:timestamp;not null;default:now()" json:"requested_at"`
	UpdatedAt   time.Time                `gorm:"type:timestamp;not null;default:now()" json:"updated_at"`
	Project     Project                  `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE" json:"project,omitempty"`
}

// TableName specifies the table name for Project
func (Project) TableName() string {
	return "projects"
}

// TableName specifies the table name for ProjectMember
func (ProjectMember) TableName() string {
	return "project_members"
}

// TableName specifies the table name for ProjectJoinRequest
func (ProjectJoinRequest) TableName() string {
	return "project_join_requests"
}
