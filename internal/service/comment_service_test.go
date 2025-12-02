package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
	"project-board-api/internal/response"
)

// MockCommentRepository is a mock implementation of CommentRepository
type MockCommentRepository struct {
	CreateFunc      func(ctx context.Context, comment *domain.Comment) error
	FindByIDFunc    func(ctx context.Context, id uuid.UUID) (*domain.Comment, error)
	FindByBoardIDFunc func(ctx context.Context, boardID uuid.UUID) ([]*domain.Comment, error)
	UpdateFunc      func(ctx context.Context, comment *domain.Comment) error
	DeleteFunc      func(ctx context.Context, id uuid.UUID) error
}

func (m *MockCommentRepository) Create(ctx context.Context, comment *domain.Comment) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, comment)
	}
	return nil
}

func (m *MockCommentRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Comment, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockCommentRepository) FindByBoardID(ctx context.Context, boardID uuid.UUID) ([]*domain.Comment, error) {
	if m.FindByBoardIDFunc != nil {
		return m.FindByBoardIDFunc(ctx, boardID)
	}
	return nil, nil
}

func (m *MockCommentRepository) Update(ctx context.Context, comment *domain.Comment) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, comment)
	}
	return nil
}

func (m *MockCommentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func TestCommentService_CreateComment(t *testing.T) {
	boardID := uuid.New()

	tests := []struct {
		name        string
		req         *dto.CreateCommentRequest
		mockBoard   func(*MockBoardRepository)
		mockComment func(*MockCommentRepository)
		wantErr     bool
		wantErrCode string
	}{
		{
			name: "성공: 정상적인 Comment 생성",
			req: &dto.CreateCommentRequest{
				BoardID: boardID,
				Content: "Test Comment",
			},
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return &domain.Board{}, nil
				}
			},
			mockComment: func(m *MockCommentRepository) {
				m.CreateFunc = func(ctx context.Context, comment *domain.Comment) error {
					comment.ID = uuid.New()
					comment.CreatedAt = time.Now()
					comment.UpdatedAt = time.Now()
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "실패: Board가 존재하지 않음",
			req: &dto.CreateCommentRequest{
				BoardID: boardID,
				Content: "Test Comment",
			},
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return nil, gorm.ErrRecordNotFound
				}
			},
			mockComment: func(m *MockCommentRepository) {},
			wantErr:     true,
			wantErrCode: response.ErrCodeNotFound,
		},
		{
			name: "실패: Comment 생성 중 DB 에러",
			req: &dto.CreateCommentRequest{
				BoardID: boardID,
				Content: "Test Comment",
			},
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return &domain.Board{}, nil
				}
			},
			mockComment: func(m *MockCommentRepository) {
				m.CreateFunc = func(ctx context.Context, comment *domain.Comment) error {
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
			mockCommentRepo := &MockCommentRepository{}
			tt.mockBoard(mockBoardRepo)
			tt.mockComment(mockCommentRepo)

			logger, _ := zap.NewDevelopment()
			service := NewCommentService(mockCommentRepo, mockBoardRepo, &MockAttachmentRepository{}, nil, logger)

			// When
			userID := uuid.New()
			got, err := service.CreateComment(context.Background(), userID, tt.req)

			// Then
			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateComment() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if appErr, ok := err.(*response.AppError); ok {
					if appErr.Code != tt.wantErrCode {
						t.Errorf("CreateComment() error code = %v, want %v", appErr.Code, tt.wantErrCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("CreateComment() unexpected error = %v", err)
					return
				}
				if got == nil {
					t.Error("CreateComment() returned nil response")
					return
				}
				if got.Content != tt.req.Content {
					t.Errorf("CreateComment() Content = %v, want %v", got.Content, tt.req.Content)
				}
			}
		})
	}
}

func TestCommentService_GetComments(t *testing.T) {
	boardID := uuid.New()

	tests := []struct {
		name        string
		boardID     uuid.UUID
		mockBoard   func(*MockBoardRepository)
		mockComment func(*MockCommentRepository)
		wantErr     bool
		wantErrCode string
		wantCount   int
	}{
		{
			name:    "성공: Comment 목록 조회",
			boardID: boardID,
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return &domain.Board{}, nil
				}
			},
			mockComment: func(m *MockCommentRepository) {
				m.FindByBoardIDFunc = func(ctx context.Context, bID uuid.UUID) ([]*domain.Comment, error) {
					return []*domain.Comment{
						{
							BaseModel: domain.BaseModel{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
							BoardID:   boardID,
							UserID:    uuid.New(),
							Content:   "Comment 1",
						},
						{
							BaseModel: domain.BaseModel{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
							BoardID:   boardID,
							UserID:    uuid.New(),
							Content:   "Comment 2",
						},
					}, nil
				}
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:    "성공: 빈 Comment 목록",
			boardID: boardID,
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return &domain.Board{}, nil
				}
			},
			mockComment: func(m *MockCommentRepository) {
				m.FindByBoardIDFunc = func(ctx context.Context, bID uuid.UUID) ([]*domain.Comment, error) {
					return []*domain.Comment{}, nil
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
			mockComment: func(m *MockCommentRepository) {},
			wantErr:     true,
			wantErrCode: response.ErrCodeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockBoardRepo := &MockBoardRepository{}
			mockCommentRepo := &MockCommentRepository{}
			tt.mockBoard(mockBoardRepo)
			tt.mockComment(mockCommentRepo)

			logger, _ := zap.NewDevelopment()
			service := NewCommentService(mockCommentRepo, mockBoardRepo, &MockAttachmentRepository{}, nil, logger)

			// When
			got, err := service.GetComments(context.Background(), tt.boardID)

			// Then
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetComments() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if appErr, ok := err.(*response.AppError); ok {
					if appErr.Code != tt.wantErrCode {
						t.Errorf("GetComments() error code = %v, want %v", appErr.Code, tt.wantErrCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("GetComments() unexpected error = %v", err)
					return
				}
				if got == nil {
					t.Error("GetComments() returned nil response")
					return
				}
				if len(got) != tt.wantCount {
					t.Errorf("GetComments() count = %v, want %v", len(got), tt.wantCount)
				}
			}
		})
	}
}

func TestCommentService_UpdateComment(t *testing.T) {
	commentID := uuid.New()

	tests := []struct {
		name        string
		commentID   uuid.UUID
		req         *dto.UpdateCommentRequest
		mockComment func(*MockCommentRepository)
		wantErr     bool
		wantErrCode string
	}{
		{
			name:      "성공: Comment 업데이트",
			commentID: commentID,
			req: &dto.UpdateCommentRequest{
				Content: "Updated Comment",
			},
			mockComment: func(m *MockCommentRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Comment, error) {
					return &domain.Comment{
						BaseModel: domain.BaseModel{ID: commentID, CreatedAt: time.Now(), UpdatedAt: time.Now()},
						Content:   "Old Comment",
					}, nil
				}
				m.UpdateFunc = func(ctx context.Context, comment *domain.Comment) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:      "실패: Comment가 존재하지 않음",
			commentID: commentID,
			req: &dto.UpdateCommentRequest{
				Content: "Updated Comment",
			},
			mockComment: func(m *MockCommentRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Comment, error) {
					return nil, gorm.ErrRecordNotFound
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeNotFound,
		},
		{
			name:      "실패: Comment 업데이트 중 DB 에러",
			commentID: commentID,
			req: &dto.UpdateCommentRequest{
				Content: "Updated Comment",
			},
			mockComment: func(m *MockCommentRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Comment, error) {
					return &domain.Comment{
						BaseModel: domain.BaseModel{ID: commentID, CreatedAt: time.Now(), UpdatedAt: time.Now()},
						Content:   "Old Comment",
					}, nil
				}
				m.UpdateFunc = func(ctx context.Context, comment *domain.Comment) error {
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
			mockCommentRepo := &MockCommentRepository{}
			tt.mockComment(mockCommentRepo)

			logger, _ := zap.NewDevelopment()
			service := NewCommentService(mockCommentRepo, mockBoardRepo, &MockAttachmentRepository{}, nil, logger)

			// When
			got, err := service.UpdateComment(context.Background(), tt.commentID, tt.req)

			// Then
			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateComment() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if appErr, ok := err.(*response.AppError); ok {
					if appErr.Code != tt.wantErrCode {
						t.Errorf("UpdateComment() error code = %v, want %v", appErr.Code, tt.wantErrCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("UpdateComment() unexpected error = %v", err)
					return
				}
				if got == nil {
					t.Error("UpdateComment() returned nil response")
					return
				}
				if got.Content != tt.req.Content {
					t.Errorf("UpdateComment() Content = %v, want %v", got.Content, tt.req.Content)
				}
			}
		})
	}
}

func TestCommentService_DeleteComment(t *testing.T) {
	commentID := uuid.New()

	tests := []struct {
		name        string
		commentID   uuid.UUID
		mockComment func(*MockCommentRepository)
		wantErr     bool
		wantErrCode string
	}{
		{
			name:      "성공: Comment 삭제",
			commentID: commentID,
			mockComment: func(m *MockCommentRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Comment, error) {
					return &domain.Comment{
						BaseModel: domain.BaseModel{ID: commentID},
					}, nil
				}
				m.DeleteFunc = func(ctx context.Context, id uuid.UUID) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:      "실패: Comment가 존재하지 않음",
			commentID: commentID,
			mockComment: func(m *MockCommentRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Comment, error) {
					return nil, gorm.ErrRecordNotFound
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeNotFound,
		},
		{
			name:      "실패: Comment 삭제 중 DB 에러",
			commentID: commentID,
			mockComment: func(m *MockCommentRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Comment, error) {
					return &domain.Comment{
						BaseModel: domain.BaseModel{ID: commentID},
					}, nil
				}
				m.DeleteFunc = func(ctx context.Context, id uuid.UUID) error {
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
			mockCommentRepo := &MockCommentRepository{}
			tt.mockComment(mockCommentRepo)

			logger, _ := zap.NewDevelopment()
			service := NewCommentService(mockCommentRepo, mockBoardRepo, &MockAttachmentRepository{}, nil, logger)

			// When
			err := service.DeleteComment(context.Background(), tt.commentID)

			// Then
			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteComment() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if appErr, ok := err.(*response.AppError); ok {
					if appErr.Code != tt.wantErrCode {
						t.Errorf("DeleteComment() error code = %v, want %v", appErr.Code, tt.wantErrCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("DeleteComment() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestCommentService_toCommentResponse_Attachments tests attachment conversion in toCommentResponse
func TestCommentService_toCommentResponse_Attachments(t *testing.T) {
	mockCommentRepo := &MockCommentRepository{}
	mockBoardRepo := &MockBoardRepository{}
	logger, _ := zap.NewDevelopment()
			service := NewCommentService(mockCommentRepo, mockBoardRepo, &MockAttachmentRepository{}, nil, logger)

	t.Run("첨부파일 변환: 여러 첨부파일", func(t *testing.T) {
		commentID := uuid.New()
		boardID := uuid.New()
		userID := uuid.New()
		uploader1 := uuid.New()
		uploader2 := uuid.New()

		comment := &domain.Comment{
			BaseModel: domain.BaseModel{
				ID:        commentID,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			BoardID: boardID,
			UserID:  userID,
			Content: "Test comment with attachments",
			Attachments: []domain.Attachment{
				{
					BaseModel: domain.BaseModel{
						ID:        uuid.New(),
						CreatedAt: time.Now(),
					},
					EntityType:  domain.EntityTypeComment,
					EntityID:    &commentID,
					FileName:    "document.pdf",
					FileURL:     "https://s3.example.com/document.pdf",
					FileSize:    1024000,
					ContentType: "application/pdf",
					UploadedBy:  uploader1,
				},
				{
					BaseModel: domain.BaseModel{
						ID:        uuid.New(),
						CreatedAt: time.Now(),
					},
					EntityType:  domain.EntityTypeComment,
					EntityID:    &commentID,
					FileName:    "image.png",
					FileURL:     "https://s3.example.com/image.png",
					FileSize:    512000,
					ContentType: "image/png",
					UploadedBy:  uploader2,
				},
			},
		}

		serviceImpl := service.(*commentServiceImpl)
		response := serviceImpl.toCommentResponse(comment)

		if len(response.Attachments) != 2 {
			t.Errorf("Expected 2 attachments, got %d", len(response.Attachments))
		}

		if response.Attachments[0].FileName != "document.pdf" {
			t.Errorf("Expected first attachment filename 'document.pdf', got '%s'", response.Attachments[0].FileName)
		}

		if response.Attachments[1].FileName != "image.png" {
			t.Errorf("Expected second attachment filename 'image.png', got '%s'", response.Attachments[1].FileName)
		}
	})

	t.Run("첨부파일 변환: 첨부파일 없음 (빈 배열)", func(t *testing.T) {
		comment := &domain.Comment{
			BaseModel: domain.BaseModel{
				ID:        uuid.New(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			BoardID:     uuid.New(),
			UserID:      uuid.New(),
			Content:     "Test comment without attachments",
			Attachments: []domain.Attachment{},
		}

		serviceImpl := service.(*commentServiceImpl)
		response := serviceImpl.toCommentResponse(comment)

		if len(response.Attachments) != 0 {
			t.Errorf("Expected 0 attachments, got %d", len(response.Attachments))
		}
	})

	t.Run("첨부파일 변환: nil 첨부파일 슬라이스", func(t *testing.T) {
		comment := &domain.Comment{
			BaseModel: domain.BaseModel{
				ID:        uuid.New(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			BoardID:     uuid.New(),
			UserID:      uuid.New(),
			Content:     "Test comment with nil attachments",
			Attachments: nil,
		}

		serviceImpl := service.(*commentServiceImpl)
		response := serviceImpl.toCommentResponse(comment)

		if response.Attachments == nil {
			t.Error("Expected empty slice, got nil")
		}

		if len(response.Attachments) != 0 {
			t.Errorf("Expected 0 attachments, got %d", len(response.Attachments))
		}
	})
}
