package metrics

import (
	"regexp"
	"strconv"
	"time"
)

var (
	// UUID pattern for endpoint normalization
	uuidPattern = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
)

// RecordExternalAPICall records external API call metrics
func (m *Metrics) RecordExternalAPICall(endpoint, method string, statusCode int, duration time.Duration, err error) {
	m.safeExecute("RecordExternalAPICall", func() {
		endpoint = normalizeEndpoint(endpoint)
		status := strconv.Itoa(statusCode)

		m.ExternalAPIRequestsTotal.WithLabelValues(endpoint, method, status).Inc()
		m.ExternalAPIRequestDuration.WithLabelValues(endpoint, status).Observe(duration.Seconds())

		// Record errors for both network errors and HTTP error status codes
		if err != nil || statusCode >= 400 {
			errorType := getErrorType(statusCode, err)
			m.ExternalAPIErrors.WithLabelValues(endpoint, errorType).Inc()
		}
	})
}

// normalizeEndpoint converts actual IDs to templates
// Example: /api/users/123e4567-e89b-12d3-a456-426614174000 -> /api/users/{id}
func normalizeEndpoint(endpoint string) string {
	return uuidPattern.ReplaceAllString(endpoint, "{id}")
}

// getErrorType categorizes error types based on status code and error
func getErrorType(statusCode int, err error) string {
	// First, check HTTP status codes (most specific)
	switch {
	case statusCode == 400:
		return "bad_request"
	case statusCode == 401:
		return "unauthorized"
	case statusCode == 403:
		return "forbidden"
	case statusCode == 404:
		return "not_found"
	case statusCode == 408:
		return "request_timeout"
	case statusCode == 429:
		return "too_many_requests"
	case statusCode >= 400 && statusCode < 500:
		return "client_error"
	case statusCode == 500:
		return "internal_server_error"
	case statusCode == 502:
		return "bad_gateway"
	case statusCode == 503:
		return "service_unavailable"
	case statusCode == 504:
		return "gateway_timeout"
	case statusCode >= 500 && statusCode < 600:
		return "server_error"
	}
	
	// If no status code error, check network/connection errors
	if err != nil {
		errMsg := err.Error()
		
		// Network/connection errors
		if contains(errMsg, "connection refused") {
			return "connection_refused"
		}
		if contains(errMsg, "no such host") {
			return "dns_error"
		}
		if contains(errMsg, "timeout") || contains(errMsg, "deadline exceeded") {
			return "timeout"
		}
		if contains(errMsg, "EOF") || contains(errMsg, "connection reset") {
			return "connection_reset"
		}
		if contains(errMsg, "TLS") || contains(errMsg, "certificate") {
			return "tls_error"
		}
		
		return "network_error"
	}
	
	return "unknown"
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
