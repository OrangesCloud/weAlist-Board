package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"project-board-api/internal/dto"
	"project-board-api/internal/response"
	"project-board-api/internal/service"
)

type ProjectMemberHandler struct {
	memberService service.ProjectMemberService
}

func NewProjectMemberHandler(memberService service.ProjectMemberService) *ProjectMemberHandler {
	return &ProjectMemberHandler{
		memberService: memberService,
	}
}

// GetMembers godoc
// @Summary      프로젝트 멤버 목록 조회
// @Description  프로젝트의 모든 멤버 정보를 조회합니다
// @Tags         project-members
// @Produce      json
// @Param        projectId path string true "Project ID (UUID)"
// @Success      200 {object} response.SuccessResponse{data=[]dto.ProjectMemberResponse} "멤버 목록 조회 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 Project ID"
// @Failure      403 {object} response.ErrorResponse "권한 없음"
// @Failure      404 {object} response.ErrorResponse "Project를 찾을 수 없음"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /projects/{projectId}/members [get]
func (h *ProjectMemberHandler) GetMembers(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid project ID")
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

	members, err := h.memberService.GetMembers(c.Request.Context(), projectID, userUUID, tokenStr)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.SendSuccess(c, http.StatusOK, members)
}

// RemoveMember godoc
// @Summary      프로젝트 멤버 제거
// @Description  프로젝트에서 멤버를 제거합니다 (OWNER 또는 ADMIN만 가능)
// @Tags         project-members
// @Produce      json
// @Param        projectId path string true "Project ID (UUID)"
// @Param        memberId path string true "Member ID (UUID)"
// @Success      200 {object} response.SuccessResponse{data=map[string]string} "멤버 제거 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 요청"
// @Failure      403 {object} response.ErrorResponse "권한 없음"
// @Failure      404 {object} response.ErrorResponse "멤버를 찾을 수 없음"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /projects/{projectId}/members/{memberId} [delete]
func (h *ProjectMemberHandler) RemoveMember(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid project ID")
		return
	}

	memberIDStr := c.Param("memberId")
	memberID, err := uuid.Parse(memberIDStr)
	if err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid member ID")
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

	err = h.memberService.RemoveMember(c.Request.Context(), projectID, userUUID, memberID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.SendSuccess(c, http.StatusOK, map[string]string{"message": "Member removed successfully"})
}

// UpdateMemberRole godoc
// @Summary      프로젝트 멤버 역할 변경
// @Description  멤버의 역할을 변경합니다 (OWNER만 가능)
// @Tags         project-members
// @Accept       json
// @Produce      json
// @Param        projectId path string true "Project ID (UUID)"
// @Param        memberId path string true "Member ID (UUID)"
// @Param        request body dto.UpdateProjectMemberRoleRequest true "역할 변경 요청"
// @Success      200 {object} response.SuccessResponse{data=dto.ProjectMemberResponse} "역할 변경 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 요청"
// @Failure      403 {object} response.ErrorResponse "권한 없음"
// @Failure      404 {object} response.ErrorResponse "멤버를 찾을 수 없음"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /projects/{projectId}/members/{memberId}/role [put]
func (h *ProjectMemberHandler) UpdateMemberRole(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid project ID")
		return
	}

	memberIDStr := c.Param("memberId")
	memberID, err := uuid.Parse(memberIDStr)
	if err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid member ID")
		return
	}

	var req dto.UpdateProjectMemberRoleRequest
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

	member, err := h.memberService.UpdateMemberRole(c.Request.Context(), projectID, userUUID, memberID, req.RoleName)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.SendSuccess(c, http.StatusOK, member)
}
