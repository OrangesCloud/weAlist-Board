package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"project-board-api/internal/domain"
)

func setupBoardTestDB(t *testing.T) *gorm.DB {
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

	db.Exec(`CREATE TABLE boards (
		id TEXT PRIMARY KEY,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		deleted_at DATETIME,
		project_id TEXT NOT NULL,
		author_id TEXT NOT NULL,
		assignee_id TEXT,
		title TEXT NOT NULL,
		content TEXT,
		custom_fields TEXT,
		start_date DATETIME,
		due_date DATETIME
	)`)

	db.Exec(`CREATE TABLE participants (
		id TEXT PRIMARY KEY,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		deleted_at DATETIME,
		board_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		UNIQUE(board_id, user_id)
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

func TestBoardRepository_FindByProjectID_WithParticipantsPreload(t *testing.T) {
	db := setupBoardTestDB(t)
	repo := NewBoardRepository(db)
	ctx := context.Background()

	projectID := uuid.New()

	// Create a project
	project := &domain.Project{
		BaseModel:   domain.BaseModel{ID: projectID},
		WorkspaceID: uuid.New(),
		OwnerID:     uuid.New(),
		Name:        "Test Project",
	}
	db.Create(project)

	// Create boards
	board1 := &domain.Board{
		BaseModel: domain.BaseModel{ID: uuid.New()},
		ProjectID: projectID,
		AuthorID:  uuid.New(),
		Title:     "Board 1",
		Content:   "Content 1",
	}
	board2 := &domain.Board{
		BaseModel: domain.BaseModel{ID: uuid.New()},
		ProjectID: projectID,
		AuthorID:  uuid.New(),
		Title:     "Board 2",
		Content:   "Content 2",
	}
	db.Create(board1)
	db.Create(board2)

	// Create participants for board1
	participant1 := &domain.Participant{
		BaseModel: domain.BaseModel{ID: uuid.New()},
		BoardID:   board1.ID,
		UserID:    uuid.New(),
	}
	participant2 := &domain.Participant{
		BaseModel: domain.BaseModel{ID: uuid.New()},
		BoardID:   board1.ID,
		UserID:    uuid.New(),
	}
	db.Create(participant1)
	db.Create(participant2)

	// Create participant for board2
	participant3 := &domain.Participant{
		BaseModel: domain.BaseModel{ID: uuid.New()},
		BoardID:   board2.ID,
		UserID:    uuid.New(),
	}
	db.Create(participant3)

	// Test: FindByProjectID should preload participants
	boards, err := repo.FindByProjectID(ctx, projectID, nil)
	if err != nil {
		t.Fatalf("FindByProjectID() error = %v", err)
	}

	// Verify we got 2 boards
	if len(boards) != 2 {
		t.Errorf("expected 2 boards, got %d", len(boards))
	}

	// Verify participants are preloaded for board1
	var foundBoard1 *domain.Board
	for _, b := range boards {
		if b.ID == board1.ID {
			foundBoard1 = b
			break
		}
	}

	if foundBoard1 == nil {
		t.Fatal("board1 not found in results")
	}

	if len(foundBoard1.Participants) != 2 {
		t.Errorf("expected board1 to have 2 participants, got %d", len(foundBoard1.Participants))
	}

	// Verify participants are preloaded for board2
	var foundBoard2 *domain.Board
	for _, b := range boards {
		if b.ID == board2.ID {
			foundBoard2 = b
			break
		}
	}

	if foundBoard2 == nil {
		t.Fatal("board2 not found in results")
	}

	if len(foundBoard2.Participants) != 1 {
		t.Errorf("expected board2 to have 1 participant, got %d", len(foundBoard2.Participants))
	}
}

func TestBoardRepository_FindByProjectID_WithNoParticipants(t *testing.T) {
	db := setupBoardTestDB(t)
	repo := NewBoardRepository(db)
	ctx := context.Background()

	projectID := uuid.New()

	// Create a project
	project := &domain.Project{
		BaseModel:   domain.BaseModel{ID: projectID},
		WorkspaceID: uuid.New(),
		OwnerID:     uuid.New(),
		Name:        "Test Project",
	}
	db.Create(project)

	// Create a board without participants
	board := &domain.Board{
		BaseModel: domain.BaseModel{ID: uuid.New()},
		ProjectID: projectID,
		AuthorID:  uuid.New(),
		Title:     "Board Without Participants",
		Content:   "Content",
	}
	db.Create(board)

	// Test: FindByProjectID should handle boards with no participants
	boards, err := repo.FindByProjectID(ctx, projectID, nil)
	if err != nil {
		t.Fatalf("FindByProjectID() error = %v", err)
	}

	// Verify we got 1 board
	if len(boards) != 1 {
		t.Errorf("expected 1 board, got %d", len(boards))
	}

	// Verify participants array is empty (not nil)
	if boards[0].Participants == nil {
		t.Error("expected Participants to be an empty slice, got nil")
	}

	if len(boards[0].Participants) != 0 {
		t.Errorf("expected 0 participants, got %d", len(boards[0].Participants))
	}
}

func TestBoardRepository_FindByProjectID_WithFilters(t *testing.T) {
	db := setupBoardTestDB(t)
	repo := NewBoardRepository(db)
	ctx := context.Background()

	projectID := uuid.New()

	// Create a project
	project := &domain.Project{
		BaseModel:   domain.BaseModel{ID: projectID},
		WorkspaceID: uuid.New(),
		OwnerID:     uuid.New(),
		Name:        "Test Project",
	}
	db.Create(project)

	// Create boards with custom fields
	board1 := &domain.Board{
		BaseModel: domain.BaseModel{ID: uuid.New()},
		ProjectID: projectID,
		AuthorID:  uuid.New(),
		Title:     "Board 1",
		Content:   "Content 1",
	}
	// Set custom_fields as JSON string for SQLite
	db.Create(board1)
	db.Exec("UPDATE boards SET custom_fields = ? WHERE id = ?", `{"stage":"in_progress"}`, board1.ID.String())

	board2 := &domain.Board{
		BaseModel: domain.BaseModel{ID: uuid.New()},
		ProjectID: projectID,
		AuthorID:  uuid.New(),
		Title:     "Board 2",
		Content:   "Content 2",
	}
	db.Create(board2)
	db.Exec("UPDATE boards SET custom_fields = ? WHERE id = ?", `{"stage":"done"}`, board2.ID.String())

	// Create participants for both boards
	participant1 := &domain.Participant{
		BaseModel: domain.BaseModel{ID: uuid.New()},
		BoardID:   board1.ID,
		UserID:    uuid.New(),
	}
	participant2 := &domain.Participant{
		BaseModel: domain.BaseModel{ID: uuid.New()},
		BoardID:   board2.ID,
		UserID:    uuid.New(),
	}
	db.Create(participant1)
	db.Create(participant2)

	// Test: FindByProjectID with filters should still preload participants
	filters := map[string]interface{}{
		"stage": "in_progress",
	}
	boards, err := repo.FindByProjectID(ctx, projectID, filters)
	if err != nil {
		t.Fatalf("FindByProjectID() error = %v", err)
	}

	// Verify we got 1 board (filtered)
	if len(boards) != 1 {
		t.Errorf("expected 1 board, got %d", len(boards))
	}

	// Verify the correct board was returned
	if boards[0].ID != board1.ID {
		t.Error("expected board1 to be returned")
	}

	// Verify participants are preloaded even with filters
	if len(boards[0].Participants) != 1 {
		t.Errorf("expected 1 participant, got %d", len(boards[0].Participants))
	}
}
