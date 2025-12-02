package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"project-board-api/internal/dto"
	"project-board-api/internal/response"
	"project-board-api/internal/service"
)

type CommentHandler struct {
	commentService service.CommentService
}

func NewCommentHandler(commentService service.CommentService) *CommentHandler {
	return &CommentHandler{
		commentService: commentService,
	}
}

// CreateComment godoc
// @Summary      Comment 생성
// @Description  Board에 새로운 Comment를 작성합니다
// @Tags         comments
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateCommentRequest true "Comment 생성 요청"
// @Success      201 {object} response.SuccessResponse{data=dto.CommentResponse} "Comment 생성 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 요청"
// @Failure      404 {object} response.ErrorResponse "Board를 찾을 수 없음"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /comments [post]
func (h *CommentHandler) CreateComment(c *gin.Context) {
	// Extract user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		response.SendError(c, http.StatusUnauthorized, response.ErrCodeUnauthorized, "User ID not found in context")
		return
	}
	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		response.SendError(c, http.StatusUnauthorized, response.ErrCodeUnauthorized, "Invalid user ID format")
		return
	}

	var req dto.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid request body")
		return
	}

	comment, err := h.commentService.CreateComment(c.Request.Context(), userUUID, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.SendSuccess(c, http.StatusCreated, comment)
}

// GetComments godoc
// @Summary      Board의 Comment 목록 조회
// @Description  특정 Board의 모든 Comment를 조회합니다
// @Tags         comments
// @Produce      json
// @Param        boardId path string true "Board ID (UUID)"
// @Success      200 {object} response.SuccessResponse{data=[]dto.CommentResponse} "Comment 목록 조회 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 Board ID"
// @Failure      404 {object} response.ErrorResponse "Board를 찾을 수 없음"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /comments/board/{boardId} [get]
func (h *CommentHandler) GetComments(c *gin.Context) {
	boardIDStr := c.Param("boardId")
	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid board ID")
		return
	}

	comments, err := h.commentService.GetComments(c.Request.Context(), boardID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.SendSuccess(c, http.StatusOK, comments)
}

// GetCommentsByQuery godoc
// @Summary      Board의 Comment 목록 조회 (쿼리 파라미터 방식)
// @Description  특정 Board의 모든 Comment를 조회합니다. 프론트엔드 호환용 엔드포인트
// @Tags         comments
// @Produce      json
// @Param        boardId query string true "Board ID (UUID)"
// @Success      200 {object} response.SuccessResponse{data=[]dto.CommentResponse} "Comment 목록 조회 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 Board ID"
// @Failure      404 {object} response.ErrorResponse "Board를 찾을 수 없음"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /comments [get]
func (h *CommentHandler) GetCommentsByQuery(c *gin.Context) {
	boardIDStr := c.Query("boardId")
	if boardIDStr == "" {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Board ID is required")
		return
	}
	
	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid board ID")
		return
	}

	comments, err := h.commentService.GetComments(c.Request.Context(), boardID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.SendSuccess(c, http.StatusOK, comments)
}

// UpdateComment godoc
// @Summary      Comment 수정
// @Description  Comment 내용을 수정합니다
// @Tags         comments
// @Accept       json
// @Produce      json
// @Param        commentId path string true "Comment ID (UUID)"
// @Param        request body dto.UpdateCommentRequest true "Comment 수정 요청"
// @Success      200 {object} response.SuccessResponse{data=dto.CommentResponse} "Comment 수정 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 요청"
// @Failure      404 {object} response.ErrorResponse "Comment를 찾을 수 없음"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /comments/{commentId} [put]
func (h *CommentHandler) UpdateComment(c *gin.Context) {
	commentIDStr := c.Param("commentId")
	commentID, err := uuid.Parse(commentIDStr)
	if err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid comment ID")
		return
	}

	var req dto.UpdateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid request body")
		return
	}

	comment, err := h.commentService.UpdateComment(c.Request.Context(), commentID, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.SendSuccess(c, http.StatusOK, comment)
}

// DeleteComment godoc
// @Summary      Comment 삭제
// @Description  Comment를 소프트 삭제합니다
// @Tags         comments
// @Produce      json
// @Param        commentId path string true "Comment ID (UUID)"
// @Success      200 {object} response.SuccessResponse "Comment 삭제 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 Comment ID"
// @Failure      404 {object} response.ErrorResponse "Comment를 찾을 수 없음"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /comments/{commentId} [delete]
func (h *CommentHandler) DeleteComment(c *gin.Context) {
	commentIDStr := c.Param("commentId")
	commentID, err := uuid.Parse(commentIDStr)
	if err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid comment ID")
		return
	}

	err = h.commentService.DeleteComment(c.Request.Context(), commentID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.SendSuccess(c, http.StatusOK, nil)
}
