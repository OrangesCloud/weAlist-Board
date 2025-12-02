package metrics

import (
	"time"
)

// RecordHTTPRequest records HTTP request metrics
func (m *Metrics) RecordHTTPRequest(method, endpoint string, statusCode int, duration time.Duration) {
	m.safeExecute("RecordHTTPRequest", func() {
		status := categorizeStatus(statusCode)
		m.HTTPRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
		m.HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
	})
}

// categorizeStatus converts status code to category (2xx, 3xx, 4xx, 5xx)
func categorizeStatus(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}

// ShouldSkipEndpoint checks if endpoint should be excluded from metrics
func ShouldSkipEndpoint(path string) bool {
	return path == "/metrics" || path == "/health" ||
		path == "/api/boards/metrics" || path == "/api/boards/health"
}
