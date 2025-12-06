package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"project-board-api/internal/dto"
	"project-board-api/internal/response"
)

func (h *ProjectHandler) GetProject(c *gin.Context) {
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

	project, err := h.projectService.GetProject(c.Request.Context(), projectID, userUUID, tokenStr)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.SendSuccess(c, http.StatusOK, project)
}

// UpdateProject godoc
// @Summary      Project 수정
// @Description  Project의 이름, 설명, 날짜를 수정합니다 (OWNER만 가능)
// @Description  startDate와 dueDate를 수정할 수 있으며, startDate는 dueDate보다 이전이어야 합니다
// @Tags         projects
// @Accept       json
// @Produce      json
// @Param        projectId path string true "Project ID (UUID)"
// @Param        request body dto.UpdateProjectRequest true "Project 수정 요청"
// @Success      200 {object} response.SuccessResponse{data=dto.ProjectResponse} "Project 수정 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 요청"
// @Failure      403 {object} response.ErrorResponse "권한 없음"
// @Failure      404 {object} response.ErrorResponse "Project를 찾을 수 없음"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /projects/{projectId} [put]
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid project ID")
		return
	}

	var req dto.UpdateProjectRequest
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

	project, err := h.projectService.UpdateProject(c.Request.Context(), projectID, userUUID, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.SendSuccess(c, http.StatusOK, project)
}

// DeleteProject godoc
// @Summary      Project 삭제
// @Description  Project를 삭제합니다 (OWNER만 가능)
// @Tags         projects
// @Produce      json
// @Param        projectId path string true "Project ID (UUID)"
// @Success      200 {object} response.SuccessResponse{data=map[string]string} "Project 삭제 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 Project ID"
// @Failure      403 {object} response.ErrorResponse "권한 없음"
// @Failure      404 {object} response.ErrorResponse "Project를 찾을 수 없음"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /projects/{projectId} [delete]
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
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

	err = h.projectService.DeleteProject(c.Request.Context(), projectID, userUUID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.SendSuccess(c, http.StatusOK, map[string]string{"message": "Project deleted successfully"})
}

// SearchProjects godoc
// @Summary      Project 검색
// @Description  Workspace 내에서 Project를 이름이나 설명으로 검색합니다
// @Tags         projects
// @Produce      json
// @Param        workspaceId query string true "Workspace ID (UUID)"
// @Param        query query string true "검색어"
// @Param        page query int false "페이지 번호" default(1)
// @Param        limit query int false "페이지 크기" default(10)
// @Success      200 {object} response.SuccessResponse{data=dto.PaginatedProjectsResponse} "검색 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 요청"
// @Failure      403 {object} response.ErrorResponse "권한 없음"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /projects/search [get]
func (h *ProjectHandler) SearchProjects(c *gin.Context) {
	workspaceIDStr := c.Query("workspaceId")
	if workspaceIDStr == "" {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Workspace ID is required")
		return
	}
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid workspace ID")
		return
	}

	query := c.Query("query")
	if query == "" {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Search query is required")
		return
	}

	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
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

	result, err := h.projectService.SearchProjects(c.Request.Context(), workspaceID, userUUID, query, page, limit, tokenStr)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.SendSuccess(c, http.StatusOK, result)
}

// GetProjectInitSettings godoc
// @Summary      Project 초기 설정 조회
// @Description  Project 진입 시 필요한 초기 설정 데이터를 한 번에 조회합니다
// @Tags         projects
// @Produce      json
// @Param        projectId path string true "Project ID (UUID)"
// @Success      200 {object} response.SuccessResponse{data=dto.ProjectInitSettingsResponse} "초기 설정 조회 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 Project ID"
// @Failure      403 {object} response.ErrorResponse "권한 없음"
// @Failure      404 {object} response.ErrorResponse "Project를 찾을 수 없음"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /projects/{projectId}/init-settings [get]
func (h *ProjectHandler) GetProjectInitSettings(c *gin.Context) {
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

	settings, err := h.projectService.GetProjectInitSettings(c.Request.Context(), projectID, userUUID, tokenStr)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.SendSuccess(c, http.StatusOK, settings)
}
