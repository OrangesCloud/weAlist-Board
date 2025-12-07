package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"project-board-api/internal/dto"
	"project-board-api/internal/response"
	"project-board-api/internal/service"
)

type ProjectHandler struct {
	projectService service.ProjectService
}

func NewProjectHandler(projectService service.ProjectService) *ProjectHandler {
	return &ProjectHandler{
		projectService: projectService,
	}
}

// CreateProject godoc
// @Summary      Project 생성
// @Description  새로운 Project를 생성합니다
// @Description  startDate와 dueDate는 선택 사항이며, startDate는 dueDate보다 이전이어야 합니다
// @Tags         projects
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateProjectRequest true "Project 생성 요청"
// @Success      201 {object} response.SuccessResponse{data=dto.ProjectResponse} "Project 생성 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 요청"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /projects [post]
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	var req dto.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid request body")
		return
	}

	// Extract user ID from context (set by Auth middleware)
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

	// Extract JWT token from context (set by Auth middleware)
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

	project, err := h.projectService.CreateProject(c.Request.Context(), &req, userUUID, tokenStr)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.SendSuccess(c, http.StatusCreated, project)
}

// GetProjectsByWorkspace godoc
// @Summary      Workspace의 Project 목록 조회
// @Description  특정 Workspace에 속한 모든 Project를 조회합니다
// @Description  각 프로젝트는 startDate, dueDate (설정된 경우), attachments (첨부파일 메타데이터 배열)를 포함합니다
// @Tags         projects
// @Produce      json
// @Param        workspaceId path string true "Workspace ID (UUID)"
// @Success      200 {object} response.SuccessResponse{data=[]dto.ProjectResponse} "Project 목록 조회 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 Workspace ID"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /projects/workspace/{workspaceId} [get]
func (h *ProjectHandler) GetProjectsByWorkspace(c *gin.Context) {
	workspaceIDStr := c.Param("workspaceId")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid workspace ID")
		return
	}

	// Extract user ID from context (set by Auth middleware)
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

	// Extract JWT token from context (set by Auth middleware)
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

	projects, err := h.projectService.GetProjectsByWorkspace(c.Request.Context(), workspaceID, userUUID, tokenStr)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.SendSuccess(c, http.StatusOK, projects)
}

// GetProjectsByWorkspaceQuery godoc
// @Summary      Workspace의 Project 목록 조회 (쿼리 파라미터 방식)
// @Description  특정 Workspace에 속한 모든 Project를 조회합니다. 프론트엔드 호환용 엔드포인트
// @Description  각 프로젝트는 startDate, dueDate (설정된 경우), attachments (첨부파일 메타데이터 배열)를 포함합니다
// @Tags         projects
// @Produce      json
// @Param        workspaceId query string true "Workspace ID (UUID)"
// @Success      200 {object} response.SuccessResponse{data=dto.PaginatedProjectsResponse} "Project 목록 조회 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 Workspace ID"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /projects [get]
func (h *ProjectHandler) GetProjectsByWorkspaceQuery(c *gin.Context) {
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

	// Extract user ID from context (set by Auth middleware)
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

	// Extract JWT token from context (set by Auth middleware)
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

	projects, err := h.projectService.GetProjectsByWorkspace(c.Request.Context(), workspaceID, userUUID, tokenStr)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	// Convert []*ProjectResponse to []ProjectResponse with nil check
	projectList := make([]dto.ProjectResponse, 0, len(projects))
	for _, p := range projects {
		// Skip nil pointers to prevent panic
		if p != nil {
			projectList = append(projectList, *p)
		}
	}

	// 프론트엔드가 기대하는 형식으로 응답 (PaginatedProjectsResponse 형태)
	response.SendSuccess(c, http.StatusOK, dto.PaginatedProjectsResponse{
		Projects: projectList,
		Total:    int64(len(projectList)),
		Page:     1,
		Limit:    len(projectList),
	})
}

// GetDefaultProject godoc
// @Summary      Workspace의 기본 Project 조회
// @Description  특정 Workspace의 기본(default) Project를 조회합니다
// @Description  프로젝트는 startDate, dueDate (설정된 경우), attachments (첨부파일 메타데이터 배열)를 포함합니다
// @Tags         projects
// @Produce      json
// @Param        workspaceId path string true "Workspace ID (UUID)"
// @Success      200 {object} response.SuccessResponse{data=dto.ProjectResponse} "기본 Project 조회 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 Workspace ID"
// @Failure      404 {object} response.ErrorResponse "기본 Project를 찾을 수 없음"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /projects/workspace/{workspaceId}/default [get]
func (h *ProjectHandler) GetDefaultProject(c *gin.Context) {
	workspaceIDStr := c.Param("workspaceId")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid workspace ID")
		return
	}

	// Extract user ID from context (set by Auth middleware)
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

	// Extract JWT token from context (set by Auth middleware)
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

	project, err := h.projectService.GetDefaultProject(c.Request.Context(), workspaceID, userUUID, tokenStr)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.SendSuccess(c, http.StatusOK, project)
}

// GetProject godoc
// @Summary      Project 상세 조회
// @Description  특정 Project의 상세 정보를 조회합니다
// @Description  프로젝트는 startDate, dueDate (설정된 경우), attachments (첨부파일 메타데이터 배열)를 포함합니다
// @Tags         projects
// @Produce      json
// @Param        projectId path string true "Project ID (UUID)"
// @Success      200 {object} response.SuccessResponse{data=dto.ProjectResponse} "Project 조회 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 Project ID"
// @Failure      403 {object} response.ErrorResponse "권한 없음"
// @Failure      404 {object} response.ErrorResponse "Project를 찾을 수 없음"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /projects/{projectId} [get]
