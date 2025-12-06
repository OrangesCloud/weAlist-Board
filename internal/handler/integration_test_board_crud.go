package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
)

func TestIntegration_FullWorkflow(t *testing.T) {
	db := setupIntegrationTestDB(t)
	router := setupIntegrationRouter(db)

	// Step 1: Create project
	project := createTestProject(t, db)

	// Step 2: Create board directly in DB (to avoid auth issues)
	board := createTestBoard(t, db, project.ID)

	// Step 3: Verify board was created
	var dbBoard domain.Board
	err := db.First(&dbBoard, "id = ?", board.ID).Error
	require.NoError(t, err)
	assert.Equal(t, board.ID, dbBoard.ID)

	// Step 4: Add participants
	userID1 := uuid.New()
	userID2 := uuid.New()
	addParticipantsReq := dto.AddParticipantsRequest{
		BoardID: board.ID,
		UserIDs: []uuid.UUID{userID1, userID2},
	}
	body, _ := json.Marshal(addParticipantsReq)
	req := httptest.NewRequest(http.MethodPost, "/api/participants", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code, "Response body: %s", w.Body.String())

	// Step 5: Get boards by project and verify participants are included
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/boards/project/%s", project.ID), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Response body: %s", w.Body.String())

	var listResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &listResp)
	require.NoError(t, err)

	boards := listResp["data"].([]interface{})
	require.Len(t, boards, 1)

	boardResp := boards[0].(map[string]interface{})
	participantIDs := boardResp["participantIds"].([]interface{})
	assert.Len(t, participantIDs, 2, "Board should have 2 participants")

	// Verify participant IDs match
	participantIDStrings := make([]string, len(participantIDs))
	for i, id := range participantIDs {
		participantIDStrings[i] = id.(string)
	}
	assert.Contains(t, participantIDStrings, userID1.String())
	assert.Contains(t, participantIDStrings, userID2.String())
}

// TestIntegration_BoardStartDateAndAssigneeDefault tests board startDate field and assigneeId default value
// **Validates: Requirements 6.1, 6.2, 6.3, 6.4**

func TestIntegration_CreateBoardWithParticipants(t *testing.T) {
	db := setupIntegrationTestDB(t)
	router := setupIntegrationRouter(db)

	// Create test project
	project := createTestProject(t, db)
	authorID := uuid.New()

	// Generate participant IDs
	participant1 := uuid.New()
	participant2 := uuid.New()
	participant3 := uuid.New()

	// Create board with participants
	createReq := dto.CreateBoardRequest{
		ProjectID:    project.ID,
		Title:        "Board with Participants",
		Content:      "Testing participant creation",
		Participants: []uuid.UUID{participant1, participant2, participant3},
	}

	body, err := json.Marshal(createReq)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/boards", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", authorID.String()) // Set user ID in header for test

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify response status
	assert.Equal(t, http.StatusCreated, w.Code, "Response body: %s", w.Body.String())

	// Parse response
	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	// Verify response structure
	assert.NotNil(t, resp["data"], "Response should have data field")
	boardData := resp["data"].(map[string]interface{})

	// Verify participantIds in response
	participantIDs, ok := boardData["participantIds"].([]interface{})
	require.True(t, ok, "participantIds should be an array")
	assert.Len(t, participantIDs, 3, "Should have 3 participants in response")

	// Verify participant IDs match request
	responseIDStrings := make([]string, len(participantIDs))
	for i, id := range participantIDs {
		responseIDStrings[i] = id.(string)
	}
	assert.Contains(t, responseIDStrings, participant1.String())
	assert.Contains(t, responseIDStrings, participant2.String())
	assert.Contains(t, responseIDStrings, participant3.String())

	// Verify participants exist in database
	boardID, err := uuid.Parse(boardData["boardId"].(string))
	require.NoError(t, err)

	var dbParticipants []domain.Participant
	err = db.Where("board_id = ?", boardID).Find(&dbParticipants).Error
	require.NoError(t, err)
	assert.Len(t, dbParticipants, 3, "Should have 3 participants in database")

	// Verify participant user IDs
	dbUserIDs := make([]uuid.UUID, len(dbParticipants))
	for i, p := range dbParticipants {
		dbUserIDs[i] = p.UserID
	}
	assert.Contains(t, dbUserIDs, participant1)
	assert.Contains(t, dbUserIDs, participant2)
	assert.Contains(t, dbUserIDs, participant3)
}

// TestIntegration_CreateBoardWithoutParticipants tests board creation without participants
// **Validates: Requirements 1.3**
func TestIntegration_CreateBoardWithoutParticipants(t *testing.T) {
	db := setupIntegrationTestDB(t)
	router := setupIntegrationRouter(db)

	project := createTestProject(t, db)
	authorID := uuid.New()

	tests := []struct {
		name        string
		request     dto.CreateBoardRequest
		description string
	}{
		{
			name: "Empty participants array",
			request: dto.CreateBoardRequest{
				ProjectID:    project.ID,
				Title:        "Board with Empty Participants",
				Content:      "Testing empty array",
				Participants: []uuid.UUID{},
			},
			description: "Should create board successfully with empty participants array",
		},
		{
			name: "Omitted participants field",
			request: dto.CreateBoardRequest{
				ProjectID: project.ID,
				Title:     "Board without Participants Field",
				Content:   "Testing omitted field",
				// Participants field is not set (nil)
			},
			description: "Should create board successfully with omitted participants field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/boards", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-User-ID", authorID.String())

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Verify success
			assert.Equal(t, http.StatusCreated, w.Code, "%s - Response body: %s", tt.description, w.Body.String())

			// Parse response
			var resp map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			boardData := resp["data"].(map[string]interface{})

			// Verify participantIds is present and empty
			participantIDs, ok := boardData["participantIds"].([]interface{})
			require.True(t, ok, "%s - participantIds should be an array", tt.description)
			assert.Len(t, participantIDs, 0, "%s - participantIds should be empty", tt.description)

			// Verify no participants in database
			boardID, err := uuid.Parse(boardData["boardId"].(string))
			require.NoError(t, err)

			var dbParticipants []domain.Participant
			err = db.Where("board_id = ?", boardID).Find(&dbParticipants).Error
			require.NoError(t, err)
			assert.Len(t, dbParticipants, 0, "%s - Should have 0 participants in database", tt.description)
		})
	}
}

// TestIntegration_CreateBoardValidationErrors tests validation errors for participants
// **Validates: Requirements 3.1, 3.2**
func TestIntegration_CreateBoardValidationErrors(t *testing.T) {
	db := setupIntegrationTestDB(t)
	router := setupIntegrationRouter(db)

	project := createTestProject(t, db)
	authorID := uuid.New()

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		description    string
	}{
		{
			name: "Invalid UUID format in participants",
			requestBody: map[string]interface{}{
				"projectId":    project.ID.String(),
				"title":        "Test Board",
				"content":      "Test Content",
				"participants": []string{"invalid-uuid", "not-a-uuid"},
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Should return 400 for invalid UUID formats",
		},
		{
			name: "More than 50 participants",
			requestBody: func() map[string]interface{} {
				// Generate 51 valid UUIDs
				participants := make([]string, 51)
				for i := 0; i < 51; i++ {
					participants[i] = uuid.New().String()
				}
				return map[string]interface{}{
					"projectId":    project.ID.String(),
					"title":        "Test Board",
					"content":      "Test Content",
					"participants": participants,
				}
			}(),
			expectedStatus: http.StatusBadRequest,
			description:    "Should return 400 when more than 50 participants provided",
		},
		{
			name: "Mixed valid and invalid UUIDs",
			requestBody: map[string]interface{}{
				"projectId": project.ID.String(),
				"title":     "Test Board",
				"content":   "Test Content",
				"participants": []string{
					uuid.New().String(),
					"invalid-uuid",
					uuid.New().String(),
				},
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Should return 400 when any UUID is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/boards", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-User-ID", authorID.String())

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Verify error status
			assert.Equal(t, tt.expectedStatus, w.Code, "%s - Response body: %s", tt.description, w.Body.String())

			// Verify error response structure
			var resp map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			// Should have error field
			assert.NotNil(t, resp["error"], "%s - Response should have error field", tt.description)
			errorData, ok := resp["error"].(map[string]interface{})
			require.True(t, ok, "%s - Error should be a map", tt.description)
			assert.Contains(t, errorData, "message", "%s - Error should contain message", tt.description)
		})
	}
}

// TestIntegration_CreateBoardDuplicateParticipants tests duplicate participant handling
// **Validates: Requirements 1.4**
func TestIntegration_CreateBoardDuplicateParticipants(t *testing.T) {
	db := setupIntegrationTestDB(t)
	router := setupIntegrationRouter(db)

	project := createTestProject(t, db)
	authorID := uuid.New()

	// Generate participant IDs with duplicates
	participant1 := uuid.New()
	participant2 := uuid.New()
	participant3 := uuid.New()

	// Create board with duplicate participants
	createReq := dto.CreateBoardRequest{
		ProjectID: project.ID,
		Title:     "Board with Duplicate Participants",
		Content:   "Testing duplicate handling",
		Participants: []uuid.UUID{
			participant1,
			participant2,
			participant1, // duplicate
			participant3,
			participant2, // duplicate
			participant1, // duplicate
		},
	}

	body, err := json.Marshal(createReq)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/boards", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", authorID.String())

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify success
	assert.Equal(t, http.StatusCreated, w.Code, "Response body: %s", w.Body.String())

	// Parse response
	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	boardData := resp["data"].(map[string]interface{})

	// Verify participantIds contains only unique participants
	participantIDs, ok := boardData["participantIds"].([]interface{})
	require.True(t, ok, "participantIds should be an array")
	assert.Len(t, participantIDs, 3, "Should have only 3 unique participants in response")

	// Verify response contains deduplicated list
	responseIDStrings := make([]string, len(participantIDs))
	for i, id := range participantIDs {
		responseIDStrings[i] = id.(string)
	}
	assert.Contains(t, responseIDStrings, participant1.String())
	assert.Contains(t, responseIDStrings, participant2.String())
	assert.Contains(t, responseIDStrings, participant3.String())

	// Verify only unique participants exist in database
	boardID, err := uuid.Parse(boardData["boardId"].(string))
	require.NoError(t, err)

	var dbParticipants []domain.Participant
	err = db.Where("board_id = ?", boardID).Find(&dbParticipants).Error
	require.NoError(t, err)
	assert.Len(t, dbParticipants, 3, "Should have only 3 unique participants in database")

	// Verify database contains correct unique user IDs
	dbUserIDs := make(map[uuid.UUID]bool)
	for _, p := range dbParticipants {
		dbUserIDs[p.UserID] = true
	}
	assert.True(t, dbUserIDs[participant1], "Database should contain participant1")
	assert.True(t, dbUserIDs[participant2], "Database should contain participant2")
	assert.True(t, dbUserIDs[participant3], "Database should contain participant3")
	assert.Len(t, dbUserIDs, 3, "Database should have exactly 3 unique participants")
}
