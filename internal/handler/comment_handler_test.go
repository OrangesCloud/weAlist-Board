package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"project-board-api/internal/dto"
	"project-board-api/internal/response"
)

// MockCommentService is a mock implementation of CommentService
type MockCommentService struct {
	CreateCommentFunc func(ctx context.Context, userID uuid.UUID, req *dto.CreateCommentRequest) (*dto.CommentResponse, error)
	GetCommentsFunc   func(ctx context.Context, boardID uuid.UUID) ([]*dto.CommentResponse, error)
	UpdateCommentFunc func(ctx context.Context, commentID uuid.UUID, req *dto.UpdateCommentRequest) (*dto.CommentResponse, error)
	DeleteCommentFunc func(ctx context.Context, commentID uuid.UUID) error
}

func (m *MockCommentService) CreateComment(ctx context.Context, userID uuid.UUID, req *dto.CreateCommentRequest) (*dto.CommentResponse, error) {
	if m.CreateCommentFunc != nil {
		return m.CreateCommentFunc(ctx, userID, req)
	}
	return nil, nil
}

func (m *MockCommentService) GetComments(ctx context.Context, boardID uuid.UUID) ([]*dto.CommentResponse, error) {
	if m.GetCommentsFunc != nil {
		return m.GetCommentsFunc(ctx, boardID)
	}
	return nil, nil
}

func (m *MockCommentService) UpdateComment(ctx context.Context, commentID uuid.UUID, req *dto.UpdateCommentRequest) (*dto.CommentResponse, error) {
	if m.UpdateCommentFunc != nil {
		return m.UpdateCommentFunc(ctx, commentID, req)
	}
	return nil, nil
}

func (m *MockCommentService) DeleteComment(ctx context.Context, commentID uuid.UUID) error {
	if m.DeleteCommentFunc != nil {
		return m.DeleteCommentFunc(ctx, commentID)
	}
	return nil
}

func TestCommentHandler_CreateComment(t *testing.T) {
	boardID := uuid.New()
	userID := uuid.New()
	commentID := uuid.New()

	tests := []struct {
		name           string
		requestBody    interface{}
		mockService    func(*MockCommentService)
		expectedStatus int
	}{
		{
			name: "성공: 댓글 생성",
			requestBody: dto.CreateCommentRequest{
				BoardID: boardID,
				Content: "Test Comment",
			},
			mockService: func(m *MockCommentService) {
				m.CreateCommentFunc = func(ctx context.Context, uid uuid.UUID, req *dto.CreateCommentRequest) (*dto.CommentResponse, error) {
					return &dto.CommentResponse{
						CommentID: commentID,
						BoardID:   req.BoardID,
						UserID:    uid,
						Content:   req.Content,
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "실패: 잘못된 요청 본문",
			requestBody:    "invalid json",
			mockService:    func(m *MockCommentService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "실패: Board가 존재하지 않음",
			requestBody: dto.CreateCommentRequest{
				BoardID: boardID,
				Content: "Test Comment",
			},
			mockService: func(m *MockCommentService) {
				m.CreateCommentFunc = func(ctx context.Context, uid uuid.UUID, req *dto.CreateCommentRequest) (*dto.CommentResponse, error) {
					return nil, response.NewAppError(response.ErrCodeNotFound, "Board not found", "")
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockService := &MockCommentService{}
			tt.mockService(mockService)
			handler := NewCommentHandler(mockService)

			router := setupTestRouter()
			router.POST("/api/comments", func(c *gin.Context) {
				// Set user_id in context for testing
				c.Set("user_id", userID)
				handler.CreateComment(c)
			})

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/comments", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("CreateComment() status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestCommentHandler_GetComments(t *testing.T) {
	boardID := uuid.New()

	tests := []struct {
		name           string
		boardID        string
		mockService    func(*MockCommentService)
		expectedStatus int
	}{
		{
			name:    "성공: 댓글 목록 조회",
			boardID: boardID.String(),
			mockService: func(m *MockCommentService) {
				m.GetCommentsFunc = func(ctx context.Context, id uuid.UUID) ([]*dto.CommentResponse, error) {
					return []*dto.CommentResponse{
						{
							CommentID: uuid.New(),
							BoardID:   id,
							UserID:    uuid.New(),
							Content:   "Comment 1",
						},
						{
							CommentID: uuid.New(),
							BoardID:   id,
							UserID:    uuid.New(),
							Content:   "Comment 2",
						},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "실패: 잘못된 UUID",
			boardID:        "invalid-uuid",
			mockService:    func(m *MockCommentService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:    "실패: Board가 존재하지 않음",
			boardID: boardID.String(),
			mockService: func(m *MockCommentService) {
				m.GetCommentsFunc = func(ctx context.Context, id uuid.UUID) ([]*dto.CommentResponse, error) {
					return nil, response.NewAppError(response.ErrCodeNotFound, "Board not found", "")
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockService := &MockCommentService{}
			tt.mockService(mockService)
			handler := NewCommentHandler(mockService)

			router := setupTestRouter()
			router.GET("/api/comments/board/:boardId", handler.GetComments)

			req := httptest.NewRequest(http.MethodGet, "/api/comments/board/"+tt.boardID, nil)
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("GetComments() status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestCommentHandler_UpdateComment(t *testing.T) {
	commentID := uuid.New()
	newContent := "Updated Comment"

	tests := []struct {
		name           string
		commentID      string
		requestBody    interface{}
		mockService    func(*MockCommentService)
		expectedStatus int
	}{
		{
			name:      "성공: 댓글 수정",
			commentID: commentID.String(),
			requestBody: dto.UpdateCommentRequest{
				Content: newContent,
			},
			mockService: func(m *MockCommentService) {
				m.UpdateCommentFunc = func(ctx context.Context, id uuid.UUID, req *dto.UpdateCommentRequest) (*dto.CommentResponse, error) {
					return &dto.CommentResponse{
						CommentID: id,
						Content:   req.Content,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "실패: 잘못된 UUID",
			commentID:      "invalid-uuid",
			requestBody:    dto.UpdateCommentRequest{Content: newContent},
			mockService:    func(m *MockCommentService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "실패: 잘못된 요청 본문",
			commentID:      commentID.String(),
			requestBody:    "invalid json",
			mockService:    func(m *MockCommentService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "실패: 댓글이 존재하지 않음",
			commentID: commentID.String(),
			requestBody: dto.UpdateCommentRequest{
				Content: newContent,
			},
			mockService: func(m *MockCommentService) {
				m.UpdateCommentFunc = func(ctx context.Context, id uuid.UUID, req *dto.UpdateCommentRequest) (*dto.CommentResponse, error) {
					return nil, response.NewAppError(response.ErrCodeNotFound, "Comment not found", "")
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockService := &MockCommentService{}
			tt.mockService(mockService)
			handler := NewCommentHandler(mockService)

			router := setupTestRouter()
			router.PUT("/api/comments/:commentId", handler.UpdateComment)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPut, "/api/comments/"+tt.commentID, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("UpdateComment() status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestCommentHandler_DeleteComment(t *testing.T) {
	commentID := uuid.New()

	tests := []struct {
		name           string
		commentID      string
		mockService    func(*MockCommentService)
		expectedStatus int
	}{
		{
			name:      "성공: 댓글 삭제",
			commentID: commentID.String(),
			mockService: func(m *MockCommentService) {
				m.DeleteCommentFunc = func(ctx context.Context, id uuid.UUID) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "실패: 잘못된 UUID",
			commentID:      "invalid-uuid",
			mockService:    func(m *MockCommentService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "실패: 댓글이 존재하지 않음",
			commentID: commentID.String(),
			mockService: func(m *MockCommentService) {
				m.DeleteCommentFunc = func(ctx context.Context, id uuid.UUID) error {
					return response.NewAppError(response.ErrCodeNotFound, "Comment not found", "")
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockService := &MockCommentService{}
			tt.mockService(mockService)
			handler := NewCommentHandler(mockService)

			router := setupTestRouter()
			router.DELETE("/api/comments/:commentId", handler.DeleteComment)

			req := httptest.NewRequest(http.MethodDelete, "/api/comments/"+tt.commentID, nil)
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("DeleteComment() status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}
