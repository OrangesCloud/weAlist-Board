package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"project-board-api/internal/domain"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// Create tables manually for SQLite compatibility
	db.Exec(`CREATE TABLE projects (
		id TEXT PRIMARY KEY,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		deleted_at DATETIME,
		workspace_id TEXT NOT NULL,
		owner_id TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		start_date DATETIME,
		due_date DATETIME,
		is_default INTEGER DEFAULT 0,
		is_public INTEGER DEFAULT 0
	)`)

	db.Exec(`CREATE TABLE project_members (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		role_name TEXT NOT NULL,
		joined_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(project_id, user_id)
	)`)

	db.Exec(`CREATE TABLE project_join_requests (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'PENDING',
		requested_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	db.Exec(`CREATE TABLE attachments (
		id TEXT PRIMARY KEY,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		deleted_at DATETIME,
		entity_type TEXT NOT NULL,
		entity_id TEXT NOT NULL,
		file_name TEXT NOT NULL,
		file_url TEXT NOT NULL,
		file_size INTEGER NOT NULL,
		content_type TEXT NOT NULL,
		uploaded_by TEXT NOT NULL
	)`)

	return db
}

func TestProjectMemberRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectMemberRepository(db)
	ctx := context.Background()

	// Create a project first
	project := &domain.Project{
		BaseModel:   domain.BaseModel{ID: uuid.New()},
		WorkspaceID: uuid.New(),
		OwnerID:     uuid.New(),
		Name:        "Test Project",
	}
	if err := db.Create(project).Error; err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	member := &domain.ProjectMember{
		ProjectID: project.ID,
		UserID:    uuid.New(),
		RoleName:  domain.ProjectRoleOwner,
	}

	err := repo.Create(ctx, member)
	if err != nil {
		t.Errorf("Create() error = %v", err)
	}

	// Verify member was created
	var count int64
	db.Model(&domain.ProjectMember{}).Where("project_id = ?", project.ID).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 member, got %d", count)
	}
}

func TestProjectMemberRepository_FindByProjectID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectMemberRepository(db)
	ctx := context.Background()

	project := &domain.Project{
		BaseModel:   domain.BaseModel{ID: uuid.New()},
		WorkspaceID: uuid.New(),
		OwnerID:     uuid.New(),
		Name:        "Test Project",
	}
	db.Create(project)

	// Create multiple members
	member1 := &domain.ProjectMember{
		ProjectID: project.ID,
		UserID:    uuid.New(),
		RoleName:  domain.ProjectRoleOwner,
	}
	member2 := &domain.ProjectMember{
		ProjectID: project.ID,
		UserID:    uuid.New(),
		RoleName:  domain.ProjectRoleMember,
	}
	repo.Create(ctx, member1)
	repo.Create(ctx, member2)

	members, err := repo.FindByProjectID(ctx, project.ID)
	if err != nil {
		t.Errorf("FindByProjectID() error = %v", err)
	}
	if len(members) != 2 {
		t.Errorf("expected 2 members, got %d", len(members))
	}
}

func TestProjectMemberRepository_FindByProjectAndUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectMemberRepository(db)
	ctx := context.Background()

	project := &domain.Project{
		BaseModel:   domain.BaseModel{ID: uuid.New()},
		WorkspaceID: uuid.New(),
		OwnerID:     uuid.New(),
		Name:        "Test Project",
	}
	db.Create(project)

	userID := uuid.New()
	member := &domain.ProjectMember{
		ProjectID: project.ID,
		UserID:    userID,
		RoleName:  domain.ProjectRoleAdmin,
	}
	repo.Create(ctx, member)

	found, err := repo.FindByProjectAndUser(ctx, project.ID, userID)
	if err != nil {
		t.Errorf("FindByProjectAndUser() error = %v", err)
	}
	if found.RoleName != domain.ProjectRoleAdmin {
		t.Errorf("expected role ADMIN, got %s", found.RoleName)
	}
}

func TestProjectMemberRepository_UpdateRole(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectMemberRepository(db)
	ctx := context.Background()

	project := &domain.Project{
		BaseModel:   domain.BaseModel{ID: uuid.New()},
		WorkspaceID: uuid.New(),
		OwnerID:     uuid.New(),
		Name:        "Test Project",
	}
	db.Create(project)

	userID := uuid.New()
	member := &domain.ProjectMember{
		ProjectID: project.ID,
		UserID:    userID,
		RoleName:  domain.ProjectRoleMember,
	}
	repo.Create(ctx, member)

	err := repo.UpdateRole(ctx, project.ID, userID, domain.ProjectRoleAdmin)
	if err != nil {
		t.Errorf("UpdateRole() error = %v", err)
	}

	// Verify role was updated
	updated, _ := repo.FindByProjectAndUser(ctx, project.ID, userID)
	if updated.RoleName != domain.ProjectRoleAdmin {
		t.Errorf("expected role ADMIN, got %s", updated.RoleName)
	}
}

func TestProjectMemberRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectMemberRepository(db)
	ctx := context.Background()

	project := &domain.Project{
		BaseModel:   domain.BaseModel{ID: uuid.New()},
		WorkspaceID: uuid.New(),
		OwnerID:     uuid.New(),
		Name:        "Test Project",
	}
	db.Create(project)

	userID := uuid.New()
	member := &domain.ProjectMember{
		ProjectID: project.ID,
		UserID:    userID,
		RoleName:  domain.ProjectRoleMember,
	}
	repo.Create(ctx, member)

	err := repo.Delete(ctx, project.ID, userID)
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	// Verify member was deleted
	_, err = repo.FindByProjectAndUser(ctx, project.ID, userID)
	if err == nil {
		t.Error("expected error when finding deleted member")
	}
}

func TestProjectMemberRepository_IsProjectMember(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectMemberRepository(db)
	ctx := context.Background()

	project := &domain.Project{
		BaseModel:   domain.BaseModel{ID: uuid.New()},
		WorkspaceID: uuid.New(),
		OwnerID:     uuid.New(),
		Name:        "Test Project",
	}
	db.Create(project)

	userID := uuid.New()
	member := &domain.ProjectMember{
		ProjectID: project.ID,
		UserID:    userID,
		RoleName:  domain.ProjectRoleMember,
	}
	repo.Create(ctx, member)

	isMember, err := repo.IsProjectMember(ctx, project.ID, userID)
	if err != nil {
		t.Errorf("IsProjectMember() error = %v", err)
	}
	if !isMember {
		t.Error("expected user to be a member")
	}

	// Test non-member
	nonMemberID := uuid.New()
	isMember, err = repo.IsProjectMember(ctx, project.ID, nonMemberID)
	if err != nil {
		t.Errorf("IsProjectMember() error = %v", err)
	}
	if isMember {
		t.Error("expected user to not be a member")
	}
}
