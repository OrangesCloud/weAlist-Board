package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
	"project-board-api/internal/response"
)

// MockParticipantRepository is a mock implementation of ParticipantRepository
type MockParticipantRepository struct {
	CreateFunc             func(ctx context.Context, participant *domain.Participant) error
	FindByBoardIDFunc      func(ctx context.Context, boardID uuid.UUID) ([]*domain.Participant, error)
	FindByBoardAndUserFunc func(ctx context.Context, boardID, userID uuid.UUID) (*domain.Participant, error)
	DeleteFunc             func(ctx context.Context, boardID, userID uuid.UUID) error
}

func (m *MockParticipantRepository) Create(ctx context.Context, participant *domain.Participant) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, participant)
	}
	return nil
}

func (m *MockParticipantRepository) FindByBoardID(ctx context.Context, boardID uuid.UUID) ([]*domain.Participant, error) {
	if m.FindByBoardIDFunc != nil {
		return m.FindByBoardIDFunc(ctx, boardID)
	}
	return nil, nil
}

func (m *MockParticipantRepository) FindByBoardAndUser(ctx context.Context, boardID, userID uuid.UUID) (*domain.Participant, error) {
	if m.FindByBoardAndUserFunc != nil {
		return m.FindByBoardAndUserFunc(ctx, boardID, userID)
	}
	return nil, nil
}

func (m *MockParticipantRepository) Delete(ctx context.Context, boardID, userID uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, boardID, userID)
	}
	return nil
}

func TestParticipantService_AddParticipants(t *testing.T) {
	boardID := uuid.New()
	userID1 := uuid.New()
	userID2 := uuid.New()
	userID3 := uuid.New()

	tests := []struct {
		name            string
		req             *dto.AddParticipantsRequest
		mockBoard       func(*MockBoardRepository)
		mockParticipant func(*MockParticipantRepository)
		wantErr         bool
		wantErrCode     string
		wantSuccess     int
		wantFailed      int
	}{
		{
			name: "성공: 단건 Participant 추가 (배열 1개 요소)",
			req: &dto.AddParticipantsRequest{
				BoardID: boardID,
				UserIDs: []uuid.UUID{userID1},
			},
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return &domain.Board{}, nil
				}
			},
			mockParticipant: func(m *MockParticipantRepository) {
				m.FindByBoardAndUserFunc = func(ctx context.Context, bID, uID uuid.UUID) (*domain.Participant, error) {
					return nil, gorm.ErrRecordNotFound
				}
				m.CreateFunc = func(ctx context.Context, participant *domain.Participant) error {
					participant.ID = uuid.New()
					participant.CreatedAt = time.Now()
					participant.UpdatedAt = time.Now()
					return nil
				}
			},
			wantErr:     false,
			wantSuccess: 1,
			wantFailed:  0,
		},
		{
			name: "성공: 다중 Participant 추가",
			req: &dto.AddParticipantsRequest{
				BoardID: boardID,
				UserIDs: []uuid.UUID{userID1, userID2, userID3},
			},
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return &domain.Board{}, nil
				}
			},
			mockParticipant: func(m *MockParticipantRepository) {
				m.FindByBoardAndUserFunc = func(ctx context.Context, bID, uID uuid.UUID) (*domain.Participant, error) {
					return nil, gorm.ErrRecordNotFound
				}
				m.CreateFunc = func(ctx context.Context, participant *domain.Participant) error {
					participant.ID = uuid.New()
					participant.CreatedAt = time.Now()
					participant.UpdatedAt = time.Now()
					return nil
				}
			},
			wantErr:     false,
			wantSuccess: 3,
			wantFailed:  0,
		},
		{
			name: "성공: 중복 UserID 제거 처리",
			req: &dto.AddParticipantsRequest{
				BoardID: boardID,
				UserIDs: []uuid.UUID{userID1, userID1, userID2},
			},
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return &domain.Board{}, nil
				}
			},
			mockParticipant: func(m *MockParticipantRepository) {
				m.FindByBoardAndUserFunc = func(ctx context.Context, bID, uID uuid.UUID) (*domain.Participant, error) {
					return nil, gorm.ErrRecordNotFound
				}
				m.CreateFunc = func(ctx context.Context, participant *domain.Participant) error {
					participant.ID = uuid.New()
					participant.CreatedAt = time.Now()
					participant.UpdatedAt = time.Now()
					return nil
				}
			},
			wantErr:     false,
			wantSuccess: 2,
			wantFailed:  0,
		},
		{
			name: "부분 성공: 일부 Participant는 이미 존재",
			req: &dto.AddParticipantsRequest{
				BoardID: boardID,
				UserIDs: []uuid.UUID{userID1, userID2, userID3},
			},
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return &domain.Board{}, nil
				}
			},
			mockParticipant: func(m *MockParticipantRepository) {
				m.FindByBoardAndUserFunc = func(ctx context.Context, bID, uID uuid.UUID) (*domain.Participant, error) {
					// userID2는 이미 존재
					if uID == userID2 {
						return &domain.Participant{
							BaseModel: domain.BaseModel{ID: uuid.New()},
							BoardID:   boardID,
							UserID:    userID2,
						}, nil
					}
					return nil, gorm.ErrRecordNotFound
				}
				m.CreateFunc = func(ctx context.Context, participant *domain.Participant) error {
					participant.ID = uuid.New()
					participant.CreatedAt = time.Now()
					participant.UpdatedAt = time.Now()
					return nil
				}
			},
			wantErr:     false,
			wantSuccess: 2,
			wantFailed:  1,
		},
		{
			name: "모두 실패: 모든 Participant가 이미 존재",
			req: &dto.AddParticipantsRequest{
				BoardID: boardID,
				UserIDs: []uuid.UUID{userID1, userID2},
			},
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return &domain.Board{}, nil
				}
			},
			mockParticipant: func(m *MockParticipantRepository) {
				m.FindByBoardAndUserFunc = func(ctx context.Context, bID, uID uuid.UUID) (*domain.Participant, error) {
					return &domain.Participant{
						BaseModel: domain.BaseModel{ID: uuid.New()},
						BoardID:   boardID,
						UserID:    uID,
					}, nil
				}
			},
			wantErr:     false,
			wantSuccess: 0,
			wantFailed:  2,
		},
		{
			name: "실패: Board가 존재하지 않음",
			req: &dto.AddParticipantsRequest{
				BoardID: boardID,
				UserIDs: []uuid.UUID{userID1},
			},
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return nil, gorm.ErrRecordNotFound
				}
			},
			mockParticipant: func(m *MockParticipantRepository) {},
			wantErr:         true,
			wantErrCode:     response.ErrCodeNotFound,
		},
		{
			name: "부분 성공: 일부 Participant 생성 실패 (DB 에러)",
			req: &dto.AddParticipantsRequest{
				BoardID: boardID,
				UserIDs: []uuid.UUID{userID1, userID2},
			},
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return &domain.Board{}, nil
				}
			},
			mockParticipant: func(m *MockParticipantRepository) {
				m.FindByBoardAndUserFunc = func(ctx context.Context, bID, uID uuid.UUID) (*domain.Participant, error) {
					return nil, gorm.ErrRecordNotFound
				}
				m.CreateFunc = func(ctx context.Context, participant *domain.Participant) error {
					// userID2 생성 시 에러
					if participant.UserID == userID2 {
						return errors.New("database error")
					}
					participant.ID = uuid.New()
					participant.CreatedAt = time.Now()
					participant.UpdatedAt = time.Now()
					return nil
				}
			},
			wantErr:     false,
			wantSuccess: 1,
			wantFailed:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockBoardRepo := &MockBoardRepository{}
			mockParticipantRepo := &MockParticipantRepository{}
			tt.mockBoard(mockBoardRepo)
			tt.mockParticipant(mockParticipantRepo)

			service := NewParticipantService(mockParticipantRepo, mockBoardRepo)

			// When
			result, err := service.AddParticipants(context.Background(), tt.req)

			// Then
			if tt.wantErr {
				if err == nil {
					t.Errorf("AddParticipants() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if appErr, ok := err.(*response.AppError); ok {
					if appErr.Code != tt.wantErrCode {
						t.Errorf("AddParticipants() error code = %v, want %v", appErr.Code, tt.wantErrCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("AddParticipants() unexpected error = %v", err)
					return
				}
				if result == nil {
					t.Error("AddParticipants() returned nil result")
					return
				}
				if result.TotalSuccess != tt.wantSuccess {
					t.Errorf("AddParticipants() TotalSuccess = %v, want %v", result.TotalSuccess, tt.wantSuccess)
				}
				if result.TotalFailed != tt.wantFailed {
					t.Errorf("AddParticipants() TotalFailed = %v, want %v", result.TotalFailed, tt.wantFailed)
				}
				if result.TotalRequested != tt.wantSuccess+tt.wantFailed {
					t.Errorf("AddParticipants() TotalRequested = %v, want %v", result.TotalRequested, tt.wantSuccess+tt.wantFailed)
				}
				if len(result.Results) != tt.wantSuccess+tt.wantFailed {
					t.Errorf("AddParticipants() Results length = %v, want %v", len(result.Results), tt.wantSuccess+tt.wantFailed)
				}
			}
		})
	}
}

func TestParticipantService_GetParticipants(t *testing.T) {
	boardID := uuid.New()

	tests := []struct {
		name            string
		boardID         uuid.UUID
		mockBoard       func(*MockBoardRepository)
		mockParticipant func(*MockParticipantRepository)
		wantErr         bool
		wantErrCode     string
		wantCount       int
	}{
		{
			name:    "성공: Participant 목록 조회",
			boardID: boardID,
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return &domain.Board{}, nil
				}
			},
			mockParticipant: func(m *MockParticipantRepository) {
				m.FindByBoardIDFunc = func(ctx context.Context, bID uuid.UUID) ([]*domain.Participant, error) {
					return []*domain.Participant{
						{
							BaseModel: domain.BaseModel{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
							BoardID:   boardID,
							UserID:    uuid.New(),
						},
						{
							BaseModel: domain.BaseModel{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
							BoardID:   boardID,
							UserID:    uuid.New(),
						},
					}, nil
				}
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:    "성공: 빈 Participant 목록",
			boardID: boardID,
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return &domain.Board{}, nil
				}
			},
			mockParticipant: func(m *MockParticipantRepository) {
				m.FindByBoardIDFunc = func(ctx context.Context, bID uuid.UUID) ([]*domain.Participant, error) {
					return []*domain.Participant{}, nil
				}
			},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:    "실패: Board가 존재하지 않음",
			boardID: boardID,
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return nil, gorm.ErrRecordNotFound
				}
			},
			mockParticipant: func(m *MockParticipantRepository) {},
			wantErr:         true,
			wantErrCode:     response.ErrCodeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockBoardRepo := &MockBoardRepository{}
			mockParticipantRepo := &MockParticipantRepository{}
			tt.mockBoard(mockBoardRepo)
			tt.mockParticipant(mockParticipantRepo)

			service := NewParticipantService(mockParticipantRepo, mockBoardRepo)

			// When
			got, err := service.GetParticipants(context.Background(), tt.boardID)

			// Then
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetParticipants() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if appErr, ok := err.(*response.AppError); ok {
					if appErr.Code != tt.wantErrCode {
						t.Errorf("GetParticipants() error code = %v, want %v", appErr.Code, tt.wantErrCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("GetParticipants() unexpected error = %v", err)
					return
				}
				if got == nil {
					t.Error("GetParticipants() returned nil response")
					return
				}
				if len(got) != tt.wantCount {
					t.Errorf("GetParticipants() count = %v, want %v", len(got), tt.wantCount)
				}
			}
		})
	}
}

func TestParticipantService_RemoveParticipant(t *testing.T) {
	boardID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name            string
		boardID         uuid.UUID
		userID          uuid.UUID
		mockBoard       func(*MockBoardRepository)
		mockParticipant func(*MockParticipantRepository)
		wantErr         bool
		wantErrCode     string
	}{
		{
			name:    "성공: Participant 제거",
			boardID: boardID,
			userID:  userID,
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return &domain.Board{}, nil
				}
			},
			mockParticipant: func(m *MockParticipantRepository) {
				m.FindByBoardAndUserFunc = func(ctx context.Context, bID, uID uuid.UUID) (*domain.Participant, error) {
					return &domain.Participant{
						BaseModel: domain.BaseModel{ID: uuid.New()},
						BoardID:   boardID,
						UserID:    userID,
					}, nil
				}
				m.DeleteFunc = func(ctx context.Context, bID, uID uuid.UUID) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:    "실패: Board가 존재하지 않음",
			boardID: boardID,
			userID:  userID,
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return nil, gorm.ErrRecordNotFound
				}
			},
			mockParticipant: func(m *MockParticipantRepository) {},
			wantErr:         true,
			wantErrCode:     response.ErrCodeNotFound,
		},
		{
			name:    "실패: Participant가 존재하지 않음",
			boardID: boardID,
			userID:  userID,
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return &domain.Board{}, nil
				}
			},
			mockParticipant: func(m *MockParticipantRepository) {
				m.FindByBoardAndUserFunc = func(ctx context.Context, bID, uID uuid.UUID) (*domain.Participant, error) {
					return nil, gorm.ErrRecordNotFound
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeNotFound,
		},
		{
			name:    "실패: Participant 삭제 중 DB 에러",
			boardID: boardID,
			userID:  userID,
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return &domain.Board{}, nil
				}
			},
			mockParticipant: func(m *MockParticipantRepository) {
				m.FindByBoardAndUserFunc = func(ctx context.Context, bID, uID uuid.UUID) (*domain.Participant, error) {
					return &domain.Participant{
						BaseModel: domain.BaseModel{ID: uuid.New()},
						BoardID:   boardID,
						UserID:    userID,
					}, nil
				}
				m.DeleteFunc = func(ctx context.Context, bID, uID uuid.UUID) error {
					return errors.New("database error")
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockBoardRepo := &MockBoardRepository{}
			mockParticipantRepo := &MockParticipantRepository{}
			tt.mockBoard(mockBoardRepo)
			tt.mockParticipant(mockParticipantRepo)

			service := NewParticipantService(mockParticipantRepo, mockBoardRepo)

			// When
			err := service.RemoveParticipant(context.Background(), tt.boardID, tt.userID)

			// Then
			if tt.wantErr {
				if err == nil {
					t.Errorf("RemoveParticipant() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if appErr, ok := err.(*response.AppError); ok {
					if appErr.Code != tt.wantErrCode {
						t.Errorf("RemoveParticipant() error code = %v, want %v", appErr.Code, tt.wantErrCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("RemoveParticipant() unexpected error = %v", err)
				}
			}
		})
	}
}
