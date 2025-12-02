package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"project-board-api/internal/dto"
	"project-board-api/internal/response"
	"project-board-api/internal/service"
)

type ProjectJoinRequestHandler struct {
	joinRequestService service.ProjectJoinRequestService
}

func NewProjectJoinRequestHandler(joinRequestService service.ProjectJoinRequestService) *ProjectJoinRequestHandler {
	return &ProjectJoinRequestHandler{
		joinRequestService: joinRequestService,
	}
}

// CreateJoinRequest godoc
// @Summary      프로젝트 가입 요청 생성
// @Description  프로젝트에 가입 요청을 보냅니다
// @Tags         project-join-requests
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateProjectJoinRequestRequest true "가입 요청 생성"
// @Success      201 {object} response.SuccessResponse{data=dto.ProjectJoinRequestResponse} "가입 요청 생성 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 요청"
// @Failure      403 {object} response.ErrorResponse "권한 없음"
// @Failure      404 {object} response.ErrorResponse "Project를 찾을 수 없음"
// @Failure      409 {object} response.ErrorResponse "이미 멤버이거나 요청이 존재함"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /join-requests [post]
func (h *ProjectJoinRequestHandler) CreateJoinRequest(c *gin.Context) {
	var req dto.CreateProjectJoinRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid request body")
		return
	}

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

	token, exists := c.Get("jwtToken")
	if !exists {
		response.SendError(c, http.StatusUnauthorized, response.ErrCodeUnauthorized, "JWT token not found in context")
		return
	}
	tokenStr, ok := token.(string)
	if !ok {
		response.SendError(c, http.StatusUnauthorized, response.ErrCodeUnauthorized, "Invalid token format")
		return
	}

	joinRequest, err := h.joinRequestService.CreateJoinRequest(c.Request.Context(), req.ProjectID, userUUID, tokenStr)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.SendSuccess(c, http.StatusCreated, joinRequest)
}

// GetJoinRequests godoc
// @Summary      프로젝트 가입 요청 목록 조회
// @Description  프로젝트에 대한 가입 요청 목록을 조회합니다 (OWNER 또는 ADMIN만 가능)
// @Tags         project-join-requests
// @Produce      json
// @Param        projectId path string true "Project ID (UUID)"
// @Param        status query string false "요청 상태 필터 (PENDING, APPROVED, REJECTED)"
// @Success      200 {object} response.SuccessResponse{data=[]dto.ProjectJoinRequestResponse} "가입 요청 목록 조회 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 Project ID"
// @Failure      403 {object} response.ErrorResponse "권한 없음"
// @Failure      404 {object} response.ErrorResponse "Project를 찾을 수 없음"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /projects/{projectId}/join-requests [get]
func (h *ProjectJoinRequestHandler) GetJoinRequests(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid project ID")
		return
	}

	var status *string
	if statusQuery := c.Query("status"); statusQuery != "" {
		status = &statusQuery
	}

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

	token, exists := c.Get("jwtToken")
	if !exists {
		response.SendError(c, http.StatusUnauthorized, response.ErrCodeUnauthorized, "JWT token not found in context")
		return
	}
	tokenStr, ok := token.(string)
	if !ok {
		response.SendError(c, http.StatusUnauthorized, response.ErrCodeUnauthorized, "Invalid token format")
		return
	}

	joinRequests, err := h.joinRequestService.GetJoinRequests(c.Request.Context(), projectID, userUUID, status, tokenStr)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.SendSuccess(c, http.StatusOK, joinRequests)
}

// UpdateJoinRequest godoc
// @Summary      프로젝트 가입 요청 승인/거부
// @Description  가입 요청을 승인하거나 거부합니다 (OWNER 또는 ADMIN만 가능)
// @Tags         project-join-requests
// @Accept       json
// @Produce      json
// @Param        joinRequestId path string true "Join Request ID (UUID)"
// @Param        request body dto.UpdateProjectJoinRequestRequest true "가입 요청 상태 변경"
// @Success      200 {object} response.SuccessResponse{data=dto.ProjectJoinRequestResponse} "가입 요청 처리 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 요청"
// @Failure      403 {object} response.ErrorResponse "권한 없음"
// @Failure      404 {object} response.ErrorResponse "가입 요청을 찾을 수 없음"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /join-requests/{joinRequestId} [put]
func (h *ProjectJoinRequestHandler) UpdateJoinRequest(c *gin.Context) {
	joinRequestIDStr := c.Param("joinRequestId")
	joinRequestID, err := uuid.Parse(joinRequestIDStr)
	if err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid join request ID")
		return
	}

	var req dto.UpdateProjectJoinRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid request body")
		return
	}

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

	token, exists := c.Get("jwtToken")
	if !exists {
		response.SendError(c, http.StatusUnauthorized, response.ErrCodeUnauthorized, "JWT token not found in context")
		return
	}
	tokenStr, ok := token.(string)
	if !ok {
		response.SendError(c, http.StatusUnauthorized, response.ErrCodeUnauthorized, "Invalid token format")
		return
	}

	joinRequest, err := h.joinRequestService.UpdateJoinRequest(c.Request.Context(), joinRequestID, userUUID, req.Status, tokenStr)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.SendSuccess(c, http.StatusOK, joinRequest)
}
