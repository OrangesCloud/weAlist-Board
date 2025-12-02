package response

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SuccessResponse represents a successful API response
type SuccessResponse struct {
	Data      interface{} `json:"data"`
	RequestID string      `json:"requestId"`
}

// ErrorResponse represents an error API response
type ErrorResponse struct {
	Error     interface{} `json:"error"`
	RequestID string      `json:"requestId"`
}

// getRequestID gets or generates a request ID from context
func getRequestID(c *gin.Context) string {
	// Try to get request ID from context (if set by middleware)
	if requestID, exists := c.Get("requestId"); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	// Generate new request ID if not exists
	return uuid.New().String()
}

// SendSuccess sends a successful response with the given data
func SendSuccess(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, SuccessResponse{
		Data:      data,
		RequestID: getRequestID(c),
	})
}

// SendSuccessMessage sends a successful response with a message (for backwards compatibility)
func SendSuccessMessage(c *gin.Context, statusCode int, data interface{}, message string) {
	responseData := data
	if message != "" && data == nil {
		responseData = map[string]string{"message": message}
	}
	c.JSON(statusCode, SuccessResponse{
		Data:      responseData,
		RequestID: getRequestID(c),
	})
}

// SendError sends an error response with the given error code and message
func SendError(c *gin.Context, statusCode int, code string, message string) {
	errorData := map[string]interface{}{
		"code":    code,
		"message": message,
	}

	c.JSON(statusCode, ErrorResponse{
		Error:     errorData,
		RequestID: getRequestID(c),
	})
}

// SendErrorWithDetails sends an error response with additional details (deprecated, use SendError)
func SendErrorWithDetails(c *gin.Context, statusCode int, code string, message string, details string) {
	errorData := map[string]interface{}{
		"code":    code,
		"message": message,
	}
	if details != "" {
		errorData["details"] = details
	}

	c.JSON(statusCode, ErrorResponse{
		Error:     errorData,
		RequestID: getRequestID(c),
	})
}
