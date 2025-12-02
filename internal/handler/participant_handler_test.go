package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"project-board-api/internal/dto"
	"project-board-api/internal/response"
)

// MockParticipantService is a mock implementation of ParticipantService
type MockParticipantService struct {
	AddParticipantsFunc         func(ctx context.Context, req *dto.AddParticipantsRequest) (*dto.AddParticipantsResponse, error)
	AddParticipantsInternalFunc func(ctx context.Context, boardID uuid.UUID, userIDs []uuid.UUID) (int, error)
	GetParticipantsFunc         func(ctx context.Context, boardID uuid.UUID) ([]*dto.ParticipantResponse, error)
	RemoveParticipantFunc       func(ctx context.Context, boardID, userID uuid.UUID) error
}

func (m *MockParticipantService) AddParticipants(ctx context.Context, req *dto.AddParticipantsRequest) (*dto.AddParticipantsResponse, error) {
	if m.AddParticipantsFunc != nil {
		return m.AddParticipantsFunc(ctx, req)
	}
	return &dto.AddParticipantsResponse{}, nil
}

func (m *MockParticipantService) AddParticipantsInternal(ctx context.Context, boardID uuid.UUID, userIDs []uuid.UUID) (int, error) {
	if m.AddParticipantsInternalFunc != nil {
		return m.AddParticipantsInternalFunc(ctx, boardID, userIDs)
	}
	return len(userIDs), nil
}

func (m *MockParticipantService) GetParticipants(ctx context.Context, boardID uuid.UUID) ([]*dto.ParticipantResponse, error) {
	if m.GetParticipantsFunc != nil {
		return m.GetParticipantsFunc(ctx, boardID)
	}
	return nil, nil
}

func (m *MockParticipantService) RemoveParticipant(ctx context.Context, boardID, userID uuid.UUID) error {
	if m.RemoveParticipantFunc != nil {
		return m.RemoveParticipantFunc(ctx, boardID, userID)
	}
	return nil
}

func TestParticipantHandler_AddParticipants(t *testing.T) {
	boardID := uuid.New()
	userID1 := uuid.New()
	userID2 := uuid.New()
	userID3 := uuid.New()

	tests := []struct {
		name           string
		requestBody    interface{}
		mockService    func(*MockParticipantService)
		expectedStatus int
		validateBody   func(*testing.T, []byte)
	}{
		{
			name: "성공: 단건 참여자 추가",
			requestBody: dto.AddParticipantsRequest{
				BoardID: boardID,
				UserIDs: []uuid.UUID{userID1},
			},
			mockService: func(m *MockParticipantService) {
				m.AddParticipantsFunc = func(ctx context.Context, req *dto.AddParticipantsRequest) (*dto.AddParticipantsResponse, error) {
					return &dto.AddParticipantsResponse{
						TotalRequested: 1,
						TotalSuccess:   1,
						TotalFailed:    0,
						Results: []dto.ParticipantResult{
							{UserID: userID1, Success: true},
						},
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
			validateBody: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				data := resp["data"].(map[string]interface{})
				if data["totalSuccess"].(float64) != 1 {
					t.Errorf("Expected totalSuccess=1, got %v", data["totalSuccess"])
				}
			},
		},
		{
			name: "성공: 다중 참여자 추가",
			requestBody: dto.AddParticipantsRequest{
				BoardID: boardID,
				UserIDs: []uuid.UUID{userID1, userID2, userID3},
			},
			mockService: func(m *MockParticipantService) {
				m.AddParticipantsFunc = func(ctx context.Context, req *dto.AddParticipantsRequest) (*dto.AddParticipantsResponse, error) {
					return &dto.AddParticipantsResponse{
						TotalRequested: 3,
						TotalSuccess:   3,
						TotalFailed:    0,
						Results: []dto.ParticipantResult{
							{UserID: userID1, Success: true},
							{UserID: userID2, Success: true},
							{UserID: userID3, Success: true},
						},
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
			validateBody: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				data := resp["data"].(map[string]interface{})
				if data["totalSuccess"].(float64) != 3 {
					t.Errorf("Expected totalSuccess=3, got %v", data["totalSuccess"])
				}
			},
		},
		{
			name: "부분 성공: 일부 참여자만 추가 성공 (207 Multi-Status)",
			requestBody: dto.AddParticipantsRequest{
				BoardID: boardID,
				UserIDs: []uuid.UUID{userID1, userID2, userID3},
			},
			mockService: func(m *MockParticipantService) {
				m.AddParticipantsFunc = func(ctx context.Context, req *dto.AddParticipantsRequest) (*dto.AddParticipantsResponse, error) {
					return &dto.AddParticipantsResponse{
						TotalRequested: 3,
						TotalSuccess:   2,
						TotalFailed:    1,
						Results: []dto.ParticipantResult{
							{UserID: userID1, Success: true},
							{UserID: userID2, Success: true},
							{UserID: userID3, Success: false, Error: "Participant already exists"},
						},
					}, nil
				}
			},
			expectedStatus: http.StatusMultiStatus,
			validateBody: func(t *testing.T, body []byte) {
				var resp dto.AddParticipantsResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if resp.TotalSuccess != 2 {
					t.Errorf("Expected totalSuccess=2, got %v", resp.TotalSuccess)
				}
				if resp.TotalFailed != 1 {
					t.Errorf("Expected totalFailed=1, got %v", resp.TotalFailed)
				}
			},
		},
		{
			name: "실패: 모든 참여자 추가 실패 (400 Bad Request)",
			requestBody: dto.AddParticipantsRequest{
				BoardID: boardID,
				UserIDs: []uuid.UUID{userID1, userID2},
			},
			mockService: func(m *MockParticipantService) {
				m.AddParticipantsFunc = func(ctx context.Context, req *dto.AddParticipantsRequest) (*dto.AddParticipantsResponse, error) {
					return &dto.AddParticipantsResponse{
						TotalRequested: 2,
						TotalSuccess:   0,
						TotalFailed:    2,
						Results: []dto.ParticipantResult{
							{UserID: userID1, Success: false, Error: "Participant already exists"},
							{UserID: userID2, Success: false, Error: "Participant already exists"},
						},
					}, nil
				}
			},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if success, ok := resp["success"].(bool); ok && success {
					t.Errorf("Expected success=false")
				}
			},
		},
		{
			name:           "실패: 잘못된 요청 본문",
			requestBody:    "invalid json",
			mockService:    func(m *MockParticipantService) {},
			expectedStatus: http.StatusBadRequest,
			validateBody:   nil,
		},
		{
			name: "실패: Board가 존재하지 않음",
			requestBody: dto.AddParticipantsRequest{
				BoardID: boardID,
				UserIDs: []uuid.UUID{userID1},
			},
			mockService: func(m *MockParticipantService) {
				m.AddParticipantsFunc = func(ctx context.Context, req *dto.AddParticipantsRequest) (*dto.AddParticipantsResponse, error) {
					return nil, response.NewAppError(response.ErrCodeNotFound, "Board not found", "")
				}
			},
			expectedStatus: http.StatusNotFound,
			validateBody:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockService := &MockParticipantService{}
			tt.mockService(mockService)
			handler := NewParticipantHandler(mockService)

			router := setupTestRouter()
			router.POST("/api/participants", handler.AddParticipants)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/participants", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("AddParticipants() status = %v, want %v, body: %s", w.Code, tt.expectedStatus, w.Body.String())
			}

			if tt.validateBody != nil {
				tt.validateBody(t, w.Body.Bytes())
			}
		})
	}
}

func TestParticipantHandler_GetParticipants(t *testing.T) {
	boardID := uuid.New()

	tests := []struct {
		name           string
		boardID        string
		mockService    func(*MockParticipantService)
		expectedStatus int
	}{
		{
			name:    "성공: 참여자 목록 조회",
			boardID: boardID.String(),
			mockService: func(m *MockParticipantService) {
				m.GetParticipantsFunc = func(ctx context.Context, id uuid.UUID) ([]*dto.ParticipantResponse, error) {
					return []*dto.ParticipantResponse{
						{
							ID:      uuid.New(),
							BoardID: id,
							UserID:  uuid.New(),
						},
						{
							ID:      uuid.New(),
							BoardID: id,
							UserID:  uuid.New(),
						},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "실패: 잘못된 UUID",
			boardID:        "invalid-uuid",
			mockService:    func(m *MockParticipantService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:    "실패: Board가 존재하지 않음",
			boardID: boardID.String(),
			mockService: func(m *MockParticipantService) {
				m.GetParticipantsFunc = func(ctx context.Context, id uuid.UUID) ([]*dto.ParticipantResponse, error) {
					return nil, response.NewAppError(response.ErrCodeNotFound, "Board not found", "")
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockService := &MockParticipantService{}
			tt.mockService(mockService)
			handler := NewParticipantHandler(mockService)

			router := setupTestRouter()
			router.GET("/api/participants/board/:boardId", handler.GetParticipants)

			req := httptest.NewRequest(http.MethodGet, "/api/participants/board/"+tt.boardID, nil)
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("GetParticipants() status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestParticipantHandler_RemoveParticipant(t *testing.T) {
	boardID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name           string
		boardID        string
		userID         string
		mockService    func(*MockParticipantService)
		expectedStatus int
	}{
		{
			name:    "성공: 참여자 제거",
			boardID: boardID.String(),
			userID:  userID.String(),
			mockService: func(m *MockParticipantService) {
				m.RemoveParticipantFunc = func(ctx context.Context, bID, uID uuid.UUID) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "실패: 잘못된 Board UUID",
			boardID:        "invalid-uuid",
			userID:         userID.String(),
			mockService:    func(m *MockParticipantService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "실패: 잘못된 User UUID",
			boardID:        boardID.String(),
			userID:         "invalid-uuid",
			mockService:    func(m *MockParticipantService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:    "실패: 참여자가 존재하지 않음",
			boardID: boardID.String(),
			userID:  userID.String(),
			mockService: func(m *MockParticipantService) {
				m.RemoveParticipantFunc = func(ctx context.Context, bID, uID uuid.UUID) error {
					return response.NewAppError(response.ErrCodeNotFound, "Participant not found", "")
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockService := &MockParticipantService{}
			tt.mockService(mockService)
			handler := NewParticipantHandler(mockService)

			router := setupTestRouter()
			router.DELETE("/api/participants/board/:boardId/user/:userId", handler.RemoveParticipant)

			req := httptest.NewRequest(http.MethodDelete, "/api/participants/board/"+tt.boardID+"/user/"+tt.userID, nil)
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("RemoveParticipant() status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}
