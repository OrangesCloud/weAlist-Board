package dto

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestAddParticipantsRequest_Structure(t *testing.T) {
	tests := []struct {
		name        string
		request     AddParticipantsRequest
		description string
	}{
		{
			name: "Valid request with single user",
			request: AddParticipantsRequest{
				BoardID: uuid.New(),
				UserIDs: []uuid.UUID{uuid.New()},
			},
			description: "Should support single participant addition",
		},
		{
			name: "Valid request with multiple users",
			request: AddParticipantsRequest{
				BoardID: uuid.New(),
				UserIDs: []uuid.UUID{uuid.New(), uuid.New(), uuid.New()},
			},
			description: "Should support multiple participants addition",
		},
		{
			name: "Valid request at max limit (50 users)",
			request: AddParticipantsRequest{
				BoardID: uuid.New(),
				UserIDs: generateUUIDs(50),
			},
			description: "Should support up to 50 participants",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify structure can be created
			if tt.request.BoardID == uuid.Nil {
				t.Error("BoardID should not be nil")
			}
			if len(tt.request.UserIDs) == 0 {
				t.Error("UserIDs should not be empty")
			}
		})
	}
}

func TestAddParticipantsRequest_RequiredFields(t *testing.T) {
	tests := []struct {
		name        string
		jsonStr     string
		expectError bool
	}{
		{
			name:        "Missing boardId field",
			jsonStr:     `{"userIds": ["550e8400-e29b-41d4-a716-446655440000"]}`,
			expectError: false, // JSON unmarshaling won't fail, but validation will
		},
		{
			name:        "Missing userIds field",
			jsonStr:     `{"boardId": "550e8400-e29b-41d4-a716-446655440000"}`,
			expectError: false, // JSON unmarshaling won't fail, but validation will
		},
		{
			name:        "Empty userIds array",
			jsonStr:     `{"boardId": "550e8400-e29b-41d4-a716-446655440000", "userIds": []}`,
			expectError: false, // JSON unmarshaling succeeds, validation should catch this
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req AddParticipantsRequest
			err := json.Unmarshal([]byte(tt.jsonStr), &req)
			if (err != nil) != tt.expectError {
				t.Errorf("Unmarshal error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestAddParticipantsRequest_JSONMarshaling(t *testing.T) {
	boardID := uuid.New()
	userID1 := uuid.New()
	userID2 := uuid.New()

	tests := []struct {
		name    string
		request AddParticipantsRequest
	}{
		{
			name: "Single user",
			request: AddParticipantsRequest{
				BoardID: boardID,
				UserIDs: []uuid.UUID{userID1},
			},
		},
		{
			name: "Multiple users",
			request: AddParticipantsRequest{
				BoardID: boardID,
				UserIDs: []uuid.UUID{userID1, userID2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			// Unmarshal back
			var unmarshaled AddParticipantsRequest
			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Verify fields
			if unmarshaled.BoardID != tt.request.BoardID {
				t.Errorf("BoardID mismatch: got %v, want %v", unmarshaled.BoardID, tt.request.BoardID)
			}

			if len(unmarshaled.UserIDs) != len(tt.request.UserIDs) {
				t.Errorf("UserIDs length mismatch: got %d, want %d", len(unmarshaled.UserIDs), len(tt.request.UserIDs))
			}

			for i, userID := range tt.request.UserIDs {
				if unmarshaled.UserIDs[i] != userID {
					t.Errorf("UserID[%d] mismatch: got %v, want %v", i, unmarshaled.UserIDs[i], userID)
				}
			}
		})
	}
}

func TestParticipantResult_Structure(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name   string
		result ParticipantResult
	}{
		{
			name: "Success result",
			result: ParticipantResult{
				UserID:  userID,
				Success: true,
				Error:   "",
			},
		},
		{
			name: "Failure result with error",
			result: ParticipantResult{
				UserID:  userID,
				Success: false,
				Error:   "Participant already exists",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.result)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			// Unmarshal back
			var unmarshaled ParticipantResult
			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Verify fields
			if unmarshaled.UserID != tt.result.UserID {
				t.Errorf("UserID mismatch: got %v, want %v", unmarshaled.UserID, tt.result.UserID)
			}
			if unmarshaled.Success != tt.result.Success {
				t.Errorf("Success mismatch: got %v, want %v", unmarshaled.Success, tt.result.Success)
			}
			if unmarshaled.Error != tt.result.Error {
				t.Errorf("Error mismatch: got %v, want %v", unmarshaled.Error, tt.result.Error)
			}
		})
	}
}

func TestAddParticipantsResponse_Structure(t *testing.T) {
	userID1 := uuid.New()
	userID2 := uuid.New()

	tests := []struct {
		name     string
		response AddParticipantsResponse
	}{
		{
			name: "All success",
			response: AddParticipantsResponse{
				TotalRequested: 2,
				TotalSuccess:   2,
				TotalFailed:    0,
				Results: []ParticipantResult{
					{UserID: userID1, Success: true},
					{UserID: userID2, Success: true},
				},
			},
		},
		{
			name: "Partial success",
			response: AddParticipantsResponse{
				TotalRequested: 2,
				TotalSuccess:   1,
				TotalFailed:    1,
				Results: []ParticipantResult{
					{UserID: userID1, Success: true},
					{UserID: userID2, Success: false, Error: "Already exists"},
				},
			},
		},
		{
			name: "All failed",
			response: AddParticipantsResponse{
				TotalRequested: 2,
				TotalSuccess:   0,
				TotalFailed:    2,
				Results: []ParticipantResult{
					{UserID: userID1, Success: false, Error: "Invalid user"},
					{UserID: userID2, Success: false, Error: "Invalid user"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.response)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			// Unmarshal back
			var unmarshaled AddParticipantsResponse
			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Verify fields
			if unmarshaled.TotalRequested != tt.response.TotalRequested {
				t.Errorf("TotalRequested mismatch: got %d, want %d", unmarshaled.TotalRequested, tt.response.TotalRequested)
			}
			if unmarshaled.TotalSuccess != tt.response.TotalSuccess {
				t.Errorf("TotalSuccess mismatch: got %d, want %d", unmarshaled.TotalSuccess, tt.response.TotalSuccess)
			}
			if unmarshaled.TotalFailed != tt.response.TotalFailed {
				t.Errorf("TotalFailed mismatch: got %d, want %d", unmarshaled.TotalFailed, tt.response.TotalFailed)
			}
			if len(unmarshaled.Results) != len(tt.response.Results) {
				t.Errorf("Results length mismatch: got %d, want %d", len(unmarshaled.Results), len(tt.response.Results))
			}
		})
	}
}

// Helper function to generate a slice of UUIDs
func generateUUIDs(count int) []uuid.UUID {
	uuids := make([]uuid.UUID, count)
	for i := 0; i < count; i++ {
		uuids[i] = uuid.New()
	}
	return uuids
}
