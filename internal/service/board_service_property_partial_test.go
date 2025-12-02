package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
)

// **Feature: board-creation-with-participants, Property 6: Partial failure resilience**
// **Validates: Requirements 3.3, 3.4, 4.2**
// For any board creation where some or all participant additions fail,
// the Board Service should still create the board and return a successful response
// with the successfully added participants
func TestProperty_PartialFailureResilience(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("Board creation succeeds even when some/all participants fail", prop.ForAll(
		func(totalParticipants int, failureRate float64) bool {
			// Generate participants
			participants := make([]uuid.UUID, totalParticipants)
			for i := 0; i < totalParticipants; i++ {
				participants[i] = uuid.New()
			}

			projectID := uuid.New()
			userID := uuid.New()
			boardID := uuid.New()

			// Determine which participants will fail based on failure rate
			failingParticipants := make(map[uuid.UUID]bool)
			expectedSuccessCount := 0
			for _, pID := range participants {
				// Use a deterministic approach based on UUID to decide failure
				if float64(pID[0])/255.0 < failureRate {
					failingParticipants[pID] = true
				} else {
					expectedSuccessCount++
				}
			}

			// Track created participants
			createdParticipants := make(map[uuid.UUID]bool)

			// Setup mocks
			mockProjectRepo := &MockProjectRepository{
				FindByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return &domain.Project{
						BaseModel: domain.BaseModel{ID: projectID},
					}, nil
				},
			}

			mockBoardRepo := &MockBoardRepository{
				CreateFunc: func(ctx context.Context, board *domain.Board) error {
					board.ID = boardID
					return nil
				},
				FindByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					// Return board with successfully created participants
					participantList := make([]domain.Participant, 0)
					for userID := range createdParticipants {
						participantList = append(participantList, domain.Participant{
							BoardID: boardID,
							UserID:  userID,
						})
					}
					return &domain.Board{
						BaseModel:    domain.BaseModel{ID: boardID},
						ProjectID:    projectID,
						Title:        "Test Board",
						Participants: participantList,
					}, nil
				},
			}

			mockParticipantRepo := &MockParticipantRepository{
				FindByBoardAndUserFunc: func(ctx context.Context, bID, uID uuid.UUID) (*domain.Participant, error) {
					if createdParticipants[uID] {
						return &domain.Participant{BoardID: bID, UserID: uID}, nil
					}
					return nil, gorm.ErrRecordNotFound
				},
				CreateFunc: func(ctx context.Context, participant *domain.Participant) error {
					// Simulate failure for specific participants
					if failingParticipants[participant.UserID] {
						return errors.New("simulated participant creation failure")
					}
					createdParticipants[participant.UserID] = true
					return nil
				},
			}

			mockFieldOptionRepo := &MockFieldOptionRepository{}
			mockConverter := &MockFieldOptionConverter{}
			logger, _ := zap.NewDevelopment()

			service := NewBoardService(
				mockBoardRepo,
				mockProjectRepo,
				mockFieldOptionRepo,
				mockParticipantRepo,
				&MockAttachmentRepository{},
				nil, // s3Client
				mockConverter,
				nil, // metrics
				logger,
			)

			ctx := context.WithValue(context.Background(), "user_id", userID)

			req := &dto.CreateBoardRequest{
				ProjectID:    projectID,
				Title:        "Test Board",
				Content:      "Test Content",
				Participants: participants,
			}

			// Execute
			response, err := service.CreateBoard(ctx, req)

			// Verify: board creation should succeed even with participant failures
			if err != nil {
				t.Logf("Board creation failed when it should succeed: %v", err)
				return false
			}

			if response == nil {
				t.Log("Response is nil")
				return false
			}

			// Verify board was created
			if response.ID == uuid.Nil {
				t.Log("Board ID is nil")
				return false
			}

			// Verify only successful participants are in response
			if len(response.ParticipantIDs) != expectedSuccessCount {
				t.Logf("Expected %d successful participants, got %d", expectedSuccessCount, len(response.ParticipantIDs))
				return false
			}

			// Verify all participants in response are from the successful set
			for _, pID := range response.ParticipantIDs {
				if failingParticipants[pID] {
					t.Logf("Response contains a participant that should have failed: %s", pID)
					return false
				}
			}

			return true
		},
		gen.IntRange(1, 30),      // Generate 1-30 participants
		gen.Float64Range(0, 1.0), // Failure rate from 0% to 100%
	))

	properties.TestingRun(t)
}

// **Feature: board-creation-with-participants, Property 7: Failure isolation**
// **Validates: Requirements 4.3**
// For any board creation that fails, the Board Service should not create any participant records
func TestProperty_FailureIsolation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("Board creation failure prevents participant creation", prop.ForAll(
		func(participantCount int) bool {
			// Generate participants
			participants := make([]uuid.UUID, participantCount)
			for i := 0; i < participantCount; i++ {
				participants[i] = uuid.New()
			}

			projectID := uuid.New()
			userID := uuid.New()

			// Track if any participants were created
			participantCreated := false

			// Setup mocks
			mockProjectRepo := &MockProjectRepository{
				FindByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return &domain.Project{
						BaseModel: domain.BaseModel{ID: projectID},
					}, nil
				},
			}

			mockBoardRepo := &MockBoardRepository{
				CreateFunc: func(ctx context.Context, board *domain.Board) error {
					// Simulate board creation failure
					return errors.New("simulated board creation failure")
				},
			}

			mockParticipantRepo := &MockParticipantRepository{
				CreateFunc: func(ctx context.Context, participant *domain.Participant) error {
					// Track if this is called (it shouldn't be)
					participantCreated = true
					return nil
				},
			}

			mockFieldOptionRepo := &MockFieldOptionRepository{}
			mockConverter := &MockFieldOptionConverter{}
			logger, _ := zap.NewDevelopment()

			service := NewBoardService(
				mockBoardRepo,
				mockProjectRepo,
				mockFieldOptionRepo,
				mockParticipantRepo,
				&MockAttachmentRepository{},
				nil, // s3Client
				mockConverter,
				nil, // metrics
				logger,
			)

			ctx := context.WithValue(context.Background(), "user_id", userID)

			req := &dto.CreateBoardRequest{
				ProjectID:    projectID,
				Title:        "Test Board",
				Content:      "Test Content",
				Participants: participants,
			}

			// Execute
			_, err := service.CreateBoard(ctx, req)

			// Verify: board creation should fail
			if err == nil {
				t.Log("Board creation succeeded when it should have failed")
				return false
			}

			// Verify: no participants should have been created
			if participantCreated {
				t.Log("Participants were created despite board creation failure")
				return false
			}

			return true
		},
		gen.IntRange(1, 20), // Generate 1-20 participants
	))

	properties.TestingRun(t)
}
