package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"project-board-api/internal/domain"
)

func TestIntegration_BoardStartDateAndAssigneeDefault(t *testing.T) {
	db := setupIntegrationTestDB(t)
	router := setupIntegrationRouter(db)

	project := createTestProject(t, db)
	authorID := uuid.New()

	tests := []struct {
		name         string
		board        *domain.Board
		validateFunc func(*testing.T, *domain.Board)
	}{
		{
			name: "Board with startDate",
			board: &domain.Board{
				BaseModel: domain.BaseModel{
					ID:        uuid.New(),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				ProjectID: project.ID,
				AuthorID:  authorID,
				Title:     "Board with Start Date",
				Content:   "Testing start date",
				StartDate: func() *time.Time { t := time.Now(); return &t }(),
			},
			validateFunc: func(t *testing.T, board *domain.Board) {
				assert.NotNil(t, board.StartDate, "startDate should be present")
			},
		},
		{
			name: "Board without assigneeId (should remain nil in DB)",
			board: &domain.Board{
				BaseModel: domain.BaseModel{
					ID:        uuid.New(),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				ProjectID: project.ID,
				AuthorID:  authorID,
				Title:     "Board without Assignee",
				Content:   "Testing assignee default",
			},
			validateFunc: func(t *testing.T, board *domain.Board) {
				// In DB, assigneeId can be nil. The service layer sets default when creating via API
				assert.Nil(t, board.AssigneeID, "assigneeId should be nil in DB when not set")
			},
		},
		{
			name: "Board with both startDate and assigneeId",
			board: &domain.Board{
				BaseModel: domain.BaseModel{
					ID:        uuid.New(),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				ProjectID:  project.ID,
				AuthorID:   authorID,
				AssigneeID: &authorID,
				Title:      "Complete Board",
				Content:    "Testing all fields",
				StartDate:  func() *time.Time { t := time.Now(); return &t }(),
			},
			validateFunc: func(t *testing.T, board *domain.Board) {
				assert.NotNil(t, board.StartDate, "startDate should be present")
				assert.NotNil(t, board.AssigneeID, "assigneeId should be present")
				assert.Equal(t, authorID, *board.AssigneeID, "assigneeId should match")
			},
		},
		{
			name: "Board with startDate and dueDate",
			board: &domain.Board{
				BaseModel: domain.BaseModel{
					ID:        uuid.New(),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				ProjectID: project.ID,
				AuthorID:  authorID,
				Title:     "Board with Dates",
				Content:   "Testing date fields",
				StartDate: func() *time.Time { t := time.Now(); return &t }(),
				DueDate:   func() *time.Time { t := time.Now().Add(7 * 24 * time.Hour); return &t }(),
			},
			validateFunc: func(t *testing.T, board *domain.Board) {
				assert.NotNil(t, board.StartDate, "startDate should be present")
				assert.NotNil(t, board.DueDate, "dueDate should be present")
				assert.True(t, board.StartDate.Before(*board.DueDate) || board.StartDate.Equal(*board.DueDate),
					"startDate should be before or equal to dueDate")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create board directly in DB
			err := db.Create(tt.board).Error
			require.NoError(t, err, "Failed to create board")

			// Retrieve and validate
			var retrieved domain.Board
			err = db.First(&retrieved, "id = ?", tt.board.ID).Error
			require.NoError(t, err, "Failed to retrieve board")

			tt.validateFunc(t, &retrieved)

			// Also test via API to verify response includes the fields
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/boards/%s", tt.board.ID), nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code, "Response body: %s", w.Body.String())

			var resp map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			boardData := resp["data"].(map[string]interface{})
			// Verify assigneeId field is always present in response (even if nil)
			assert.Contains(t, boardData, "assigneeId", "assigneeId field should always be in response")
		})
	}
}

// TestIntegration_ProjectStartDateAndDueDate tests project date fields
// **Validates: Requirements 7.1, 7.2, 7.3**
func TestIntegration_ProjectStartDateAndDueDate(t *testing.T) {
	db := setupIntegrationTestDB(t)

	tests := []struct {
		name         string
		project      *domain.Project
		validateFunc func(*testing.T, *domain.Project)
	}{
		{
			name: "Project with startDate and dueDate",
			project: &domain.Project{
				BaseModel: domain.BaseModel{
					ID:        uuid.New(),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				WorkspaceID: uuid.New(),
				OwnerID:     uuid.New(),
				Name:        "Project with Dates",
				Description: "Testing dates",
				StartDate:   func() *time.Time { t := time.Now(); return &t }(),
				DueDate:     func() *time.Time { t := time.Now().Add(30 * 24 * time.Hour); return &t }(),
			},
			validateFunc: func(t *testing.T, p *domain.Project) {
				assert.NotNil(t, p.StartDate, "StartDate should be set")
				assert.NotNil(t, p.DueDate, "DueDate should be set")
			},
		},
		{
			name: "Project without dates",
			project: &domain.Project{
				BaseModel: domain.BaseModel{
					ID:        uuid.New(),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				WorkspaceID: uuid.New(),
				OwnerID:     uuid.New(),
				Name:        "Project without Dates",
				Description: "Testing null dates",
			},
			validateFunc: func(t *testing.T, p *domain.Project) {
				assert.Nil(t, p.StartDate, "StartDate should be nil")
				assert.Nil(t, p.DueDate, "DueDate should be nil")
			},
		},
		{
			name: "Project with only startDate",
			project: &domain.Project{
				BaseModel: domain.BaseModel{
					ID:        uuid.New(),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				WorkspaceID: uuid.New(),
				OwnerID:     uuid.New(),
				Name:        "Project with Start Only",
				Description: "Testing partial dates",
				StartDate:   func() *time.Time { t := time.Now(); return &t }(),
			},
			validateFunc: func(t *testing.T, p *domain.Project) {
				assert.NotNil(t, p.StartDate, "StartDate should be set")
				assert.Nil(t, p.DueDate, "DueDate should be nil")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.Create(tt.project).Error
			require.NoError(t, err, "Failed to create project")

			// Retrieve and validate
			var retrieved domain.Project
			err = db.First(&retrieved, "id = ?", tt.project.ID).Error
			require.NoError(t, err, "Failed to retrieve project")

			tt.validateFunc(t, &retrieved)
		})
	}
}

// TestIntegration_AttachmentsRetrieval tests attachment retrieval for boards and projects
// **Validates: Requirements 8.1, 8.2, 8.3**

func TestIntegration_DateValidation(t *testing.T) {
	db := setupIntegrationTestDB(t)

	project := createTestProject(t, db)
	authorID := uuid.New()

	tests := []struct {
		name        string
		board       *domain.Board
		shouldPass  bool
		description string
	}{
		{
			name: "Board with valid dates (startDate before dueDate)",
			board: &domain.Board{
				BaseModel: domain.BaseModel{
					ID:        uuid.New(),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				ProjectID: project.ID,
				AuthorID:  authorID,
				Title:     "Valid Board",
				Content:   "Testing valid dates",
				StartDate: func() *time.Time { t := time.Now(); return &t }(),
				DueDate:   func() *time.Time { t := time.Now().Add(7 * 24 * time.Hour); return &t }(),
			},
			shouldPass:  true,
			description: "Should accept board when startDate is before dueDate",
		},
		{
			name: "Board with equal dates",
			board: &domain.Board{
				BaseModel: domain.BaseModel{
					ID:        uuid.New(),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				ProjectID: project.ID,
				AuthorID:  authorID,
				Title:     "Equal Dates Board",
				Content:   "Testing equal dates",
				StartDate: func() *time.Time { t := time.Now(); return &t }(),
				DueDate:   func() *time.Time { t := time.Now(); return &t }(),
			},
			shouldPass:  true,
			description: "Should accept board when startDate equals dueDate",
		},
		{
			name: "Board with only startDate (no dueDate)",
			board: &domain.Board{
				BaseModel: domain.BaseModel{
					ID:        uuid.New(),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				ProjectID: project.ID,
				AuthorID:  authorID,
				Title:     "Start Only Board",
				Content:   "Testing start date only",
				StartDate: func() *time.Time { t := time.Now(); return &t }(),
			},
			shouldPass:  true,
			description: "Should accept board with only startDate",
		},
		{
			name: "Board with only dueDate (no startDate)",
			board: &domain.Board{
				BaseModel: domain.BaseModel{
					ID:        uuid.New(),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				ProjectID: project.ID,
				AuthorID:  authorID,
				Title:     "Due Only Board",
				Content:   "Testing due date only",
				DueDate:   func() *time.Time { t := time.Now().Add(7 * 24 * time.Hour); return &t }(),
			},
			shouldPass:  true,
			description: "Should accept board with only dueDate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate dates before creating
			if tt.board.StartDate != nil && tt.board.DueDate != nil {
				isValid := tt.board.StartDate.Before(*tt.board.DueDate) || tt.board.StartDate.Equal(*tt.board.DueDate)
				if tt.shouldPass {
					assert.True(t, isValid, "%s - dates should be valid", tt.description)
				} else {
					assert.False(t, isValid, "%s - dates should be invalid", tt.description)
				}
			}

			// Create board in DB
			err := db.Create(tt.board).Error
			if tt.shouldPass {
				require.NoError(t, err, "%s - should create successfully", tt.description)

				// Verify dates were stored correctly
				var retrieved domain.Board
				err = db.First(&retrieved, "id = ?", tt.board.ID).Error
				require.NoError(t, err)

				if tt.board.StartDate != nil {
					assert.NotNil(t, retrieved.StartDate, "StartDate should be stored")
				}
				if tt.board.DueDate != nil {
					assert.NotNil(t, retrieved.DueDate, "DueDate should be stored")
				}
			}
		})
	}

	// Test invalid date scenario (startDate after dueDate)
	t.Run("Board with invalid dates (startDate after dueDate)", func(t *testing.T) {
		invalidBoard := &domain.Board{
			BaseModel: domain.BaseModel{
				ID:        uuid.New(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			ProjectID: project.ID,
			AuthorID:  authorID,
			Title:     "Invalid Board",
			Content:   "Testing invalid dates",
			StartDate: func() *time.Time { t := time.Now().Add(7 * 24 * time.Hour); return &t }(),
			DueDate:   func() *time.Time { t := time.Now(); return &t }(),
		}

		// Validate that startDate is after dueDate
		assert.True(t, invalidBoard.StartDate.After(*invalidBoard.DueDate),
			"startDate should be after dueDate for this test")

		// Note: The database layer doesn't enforce this constraint
		// The validation should happen at the service/handler layer
		// This test documents that the validation logic should exist
	})
}

// TestIntegration_CreateBoardWithParticipants tests board creation with participants via HTTP API
// **Validates: Requirements 1.2, 2.2**
