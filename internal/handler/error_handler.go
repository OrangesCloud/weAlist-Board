package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"project-board-api/internal/response"
)

// handleServiceError maps service layer errors to appropriate HTTP responses
func handleServiceError(c *gin.Context, err error) {
	// Log the error for debugging
	fmt.Printf("[ERROR] Service error: %v\n", err)
	
	// Check for GORM errors
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.SendError(c, http.StatusNotFound, response.ErrCodeNotFound, "Resource not found")
		return
	}

	// Check for custom AppError
	var appErr *response.AppError
	if errors.As(err, &appErr) {
		fmt.Printf("[ERROR] AppError - Code: %s, Message: %s, Details: %s\n", appErr.Code, appErr.Message, appErr.Details)
		statusCode := mapErrorCodeToHTTPStatus(appErr.Code)
		response.SendError(c, statusCode, appErr.Code, appErr.Message)
		return
	}

	// Default to internal server error
	fmt.Printf("[ERROR] Unhandled error type: %T, value: %v\n", err, err)
	response.SendError(c, http.StatusInternalServerError, response.ErrCodeInternal, "Internal server error")
}

// mapErrorCodeToHTTPStatus maps error codes to HTTP status codes
func mapErrorCodeToHTTPStatus(code string) int {
	switch code {
	case response.ErrCodeNotFound:
		return http.StatusNotFound
	case response.ErrCodeAlreadyExists:
		return http.StatusConflict
	case response.ErrCodeValidation:
		return http.StatusBadRequest
	case response.ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case response.ErrCodeForbidden:
		return http.StatusForbidden
	case "ALREADY_MEMBER", "PENDING_REQUEST_EXISTS":
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
