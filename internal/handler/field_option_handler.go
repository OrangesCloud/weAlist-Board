package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
	"project-board-api/internal/response"
	"project-board-api/internal/service"
)

type FieldOptionHandler struct {
	fieldOptionService service.FieldOptionService
}

func NewFieldOptionHandler(fieldOptionService service.FieldOptionService) *FieldOptionHandler {
	return &FieldOptionHandler{
		fieldOptionService: fieldOptionService,
	}
}

// GetFieldOptions godoc
// @Summary      필드 옵션 목록 조회
// @Description  특정 필드 타입의 옵션 목록을 조회합니다
// @Tags         field-options
// @Produce      json
// @Param        fieldType query string true "Field Type" Enums(stage, role, importance)
// @Success      200 {object} response.SuccessResponse{data=[]dto.FieldOptionResponse} "필드 옵션 목록 조회 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 요청"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /field-options [get]
func (h *FieldOptionHandler) GetFieldOptions(c *gin.Context) {
	fieldTypeStr := c.Query("fieldType")
	if fieldTypeStr == "" {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "fieldType query parameter is required")
		return
	}

	// Validate field type
	if fieldTypeStr != "stage" && fieldTypeStr != "role" && fieldTypeStr != "importance" {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "fieldType must be one of: stage, role, importance")
		return
	}

	options, err := h.fieldOptionService.GetFieldOptions(c.Request.Context(), domain.FieldType(fieldTypeStr))
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.SendSuccess(c, http.StatusOK, options)
}

// CreateFieldOption godoc
// @Summary      필드 옵션 생성
// @Description  새로운 필드 옵션을 생성합니다
// @Tags         field-options
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateFieldOptionRequest true "필드 옵션 생성 요청"
// @Success      201 {object} response.SuccessResponse{data=dto.FieldOptionResponse} "필드 옵션 생성 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 요청"
// @Failure      409 {object} response.ErrorResponse "중복된 필드 옵션"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /field-options [post]
func (h *FieldOptionHandler) CreateFieldOption(c *gin.Context) {
	var req dto.CreateFieldOptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid request body")
		return
	}

	option, err := h.fieldOptionService.CreateFieldOption(c.Request.Context(), &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.SendSuccess(c, http.StatusCreated, option)
}

// UpdateFieldOption godoc
// @Summary      필드 옵션 수정
// @Description  필드 옵션의 정보를 수정합니다
// @Tags         field-options
// @Accept       json
// @Produce      json
// @Param        optionId path string true "Option ID (UUID)"
// @Param        request body dto.UpdateFieldOptionRequest true "필드 옵션 수정 요청"
// @Success      200 {object} response.SuccessResponse{data=dto.FieldOptionResponse} "필드 옵션 수정 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 요청"
// @Failure      404 {object} response.ErrorResponse "필드 옵션을 찾을 수 없음"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /field-options/{optionId} [patch]
func (h *FieldOptionHandler) UpdateFieldOption(c *gin.Context) {
	optionIDStr := c.Param("optionId")
	optionID, err := uuid.Parse(optionIDStr)
	if err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid option ID")
		return
	}

	var req dto.UpdateFieldOptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid request body")
		return
	}

	option, err := h.fieldOptionService.UpdateFieldOption(c.Request.Context(), optionID, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.SendSuccess(c, http.StatusOK, option)
}

// DeleteFieldOption godoc
// @Summary      필드 옵션 삭제
// @Description  필드 옵션을 삭제합니다 (시스템 기본 옵션은 삭제 불가)
// @Tags         field-options
// @Produce      json
// @Param        optionId path string true "Option ID (UUID)"
// @Success      200 {object} response.SuccessResponse "필드 옵션 삭제 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 Option ID"
// @Failure      403 {object} response.ErrorResponse "시스템 기본 옵션은 삭제 불가"
// @Failure      404 {object} response.ErrorResponse "필드 옵션을 찾을 수 없음"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /field-options/{optionId} [delete]
func (h *FieldOptionHandler) DeleteFieldOption(c *gin.Context) {
	optionIDStr := c.Param("optionId")
	optionID, err := uuid.Parse(optionIDStr)
	if err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid option ID")
		return
	}

	err = h.fieldOptionService.DeleteFieldOption(c.Request.Context(), optionID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.SendSuccess(c, http.StatusOK, map[string]string{"message": "Field option deleted successfully"})
}
