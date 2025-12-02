package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Recovery returns a middleware that recovers from panics
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic with stack trace
				logger.Error("Panic recovered",
					zap.Any("error", err),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
					zap.String("query", c.Request.URL.RawQuery),
					zap.Stack("stacktrace"),
				)

				// Also print to stdout for debugging
				logger.Info("Panic details",
					zap.String("error_type", fmt.Sprintf("%T", err)),
					zap.String("error_value", fmt.Sprintf("%v", err)),
				)

				// Return 500 error response
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"code":    "INTERNAL_ERROR",
						"message": "Internal server error",
					},
					"message": "서버 내부 오류가 발생했습니다",
				})

				c.Abort()
			}
		}()

		c.Next()
	}
}
