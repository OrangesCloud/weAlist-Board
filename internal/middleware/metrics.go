package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"project-board-api/internal/metrics"
)

// Metrics returns a middleware that records HTTP metrics
func Metrics(m *metrics.Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip metrics and health endpoints
		if metrics.ShouldSkipEndpoint(c.Request.URL.Path) {
			c.Next()
			return
		}

		start := time.Now()

		// Process request
		c.Next()

		// Record metrics
		duration := time.Since(start)
		m.RecordHTTPRequest(
			c.Request.Method,
			c.FullPath(), // Use route pattern, not actual path
			c.Writer.Status(),
			duration,
		)
	}
}
