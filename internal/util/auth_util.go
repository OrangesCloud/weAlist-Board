// internal/handler/auth_util.go (새 파일이라고 가정)

package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"project-board-api/internal/response"
)

// AuthData holds the extracted user ID and JWT token string.
type AuthData struct {
	UserID uuid.UUID
	Token  string
}

// ExtractAuthData extracts user_id and jwtToken from the Gin context.
func ExtractAuthData(c *gin.Context) (AuthData, bool) {
	// 1. User ID 추출
	userID, exists := c.Get("user_id")
	if !exists {
		response.SendError(c, http.StatusUnauthorized, response.ErrCodeUnauthorized, "User ID not found in context")
		return AuthData{}, false
	}
	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		response.SendError(c, http.StatusUnauthorized, response.ErrCodeUnauthorized, "Invalid user ID format")
		return AuthData{}, false
	}

	// 2. JWT Token 추출
	token, exists := c.Get("jwtToken")
	if !exists {
		response.SendError(c, http.StatusUnauthorized, response.ErrCodeUnauthorized, "JWT token not found in context")
		return AuthData{}, false
	}
	tokenStr, ok := token.(string)
	if !ok {
		response.SendError(c, http.StatusUnauthorized, response.ErrCodeUnauthorized, "Invalid token format")
		return AuthData{}, false
	}

	return AuthData{
		UserID: userUUID,
		Token:  tokenStr,
	}, true
}
