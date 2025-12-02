package metrics

import (
	"errors"
	"regexp"
	"strings"
	"testing"
	"testing/quick"
)

func TestNormalizeEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		expected string
	}{
		{
			name:     "UUID in path",
			endpoint: "/api/users/123e4567-e89b-12d3-a456-426614174000",
			expected: "/api/users/{id}",
		},
		{
			name:     "Multiple UUIDs",
			endpoint: "/api/users/123e4567-e89b-12d3-a456-426614174000/projects/987fcdeb-51a2-43f1-b456-789012345678",
			expected: "/api/users/{id}/projects/{id}",
		},
		{
			name:     "No UUID",
			endpoint: "/api/users",
			expected: "/api/users",
		},
		{
			name:     "UUID with query params",
			endpoint: "/api/users/123e4567-e89b-12d3-a456-426614174000?include=profile",
			expected: "/api/users/{id}?include=profile",
		},
		{
			name:     "Lowercase UUID",
			endpoint: "/api/users/abcdef12-3456-7890-abcd-ef1234567890",
			expected: "/api/users/{id}",
		},
		{
			name:     "Empty string",
			endpoint: "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeEndpoint(tt.endpoint)
			if result != tt.expected {
				t.Errorf("normalizeEndpoint(%q) = %q, want %q", tt.endpoint, result, tt.expected)
			}
		})
	}
}

func TestGetErrorType(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		err        error
		expected   string
	}{
		{
			name:       "No error and success status",
			statusCode: 200,
			err:        nil,
			expected:   "unknown",
		},
		{
			name:       "Bad request",
			statusCode: 400,
			err:        nil,
			expected:   "bad_request",
		},
		{
			name:       "Unauthorized",
			statusCode: 401,
			err:        nil,
			expected:   "unauthorized",
		},
		{
			name:       "Forbidden",
			statusCode: 403,
			err:        nil,
			expected:   "forbidden",
		},
		{
			name:       "Not found",
			statusCode: 404,
			err:        nil,
			expected:   "not_found",
		},
		{
			name:       "Request timeout",
			statusCode: 408,
			err:        nil,
			expected:   "request_timeout",
		},
		{
			name:       "Too many requests",
			statusCode: 429,
			err:        nil,
			expected:   "too_many_requests",
		},
		{
			name:       "Generic client error",
			statusCode: 418,
			err:        nil,
			expected:   "client_error",
		},
		{
			name:       "Internal server error",
			statusCode: 500,
			err:        nil,
			expected:   "internal_server_error",
		},
		{
			name:       "Bad gateway",
			statusCode: 502,
			err:        nil,
			expected:   "bad_gateway",
		},
		{
			name:       "Service unavailable",
			statusCode: 503,
			err:        nil,
			expected:   "service_unavailable",
		},
		{
			name:       "Gateway timeout",
			statusCode: 504,
			err:        nil,
			expected:   "gateway_timeout",
		},
		{
			name:       "Generic server error",
			statusCode: 507,
			err:        nil,
			expected:   "server_error",
		},
		{
			name:       "Connection refused",
			statusCode: 0,
			err:        errors.New("connection refused"),
			expected:   "connection_refused",
		},
		{
			name:       "DNS error",
			statusCode: 0,
			err:        errors.New("no such host"),
			expected:   "dns_error",
		},
		{
			name:       "Timeout error",
			statusCode: 0,
			err:        errors.New("timeout exceeded"),
			expected:   "timeout",
		},
		{
			name:       "Deadline exceeded",
			statusCode: 0,
			err:        errors.New("context deadline exceeded"),
			expected:   "timeout",
		},
		{
			name:       "Connection reset",
			statusCode: 0,
			err:        errors.New("connection reset by peer"),
			expected:   "connection_reset",
		},
		{
			name:       "EOF error",
			statusCode: 0,
			err:        errors.New("unexpected EOF"),
			expected:   "connection_reset",
		},
		{
			name:       "TLS error",
			statusCode: 0,
			err:        errors.New("TLS handshake failed"),
			expected:   "tls_error",
		},
		{
			name:       "Certificate error",
			statusCode: 0,
			err:        errors.New("certificate verification failed"),
			expected:   "tls_error",
		},
		{
			name:       "Generic network error",
			statusCode: 0,
			err:        errors.New("some network error"),
			expected:   "network_error",
		},
		{
			name:       "Unknown error",
			statusCode: 0,
			err:        nil,
			expected:   "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getErrorType(tt.statusCode, tt.err)
			if result != tt.expected {
				t.Errorf("getErrorType(%d, %v) = %q, want %q", tt.statusCode, tt.err, result, tt.expected)
			}
		})
	}
}

// Property 10: 엔드포인트 정규화
// Feature: board-service-prometheus-metrics, Property 10: Endpoint normalization
// For any endpoint containing UUIDs, normalizeEndpoint should replace all UUIDs with {id} template
// Validates: Requirements 5.5
func TestProperty_EndpointNormalization(t *testing.T) {
	// UUID pattern to verify
	uuidRegex := regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

	// Property: For any endpoint, after normalization, it should not contain any UUIDs
	property := func(endpoint string) bool {
		// Normalize the endpoint
		normalized := normalizeEndpoint(endpoint)

		// Verify that no UUIDs remain in the normalized endpoint
		if uuidRegex.MatchString(normalized) {
			t.Logf("UUID found in normalized endpoint: %s -> %s", endpoint, normalized)
			return false
		}

		// Verify that if the original had UUIDs, they were replaced with {id}
		originalHadUUID := uuidRegex.MatchString(endpoint)
		normalizedHasTemplate := strings.Contains(normalized, "{id}")

		if originalHadUUID && !normalizedHasTemplate {
			t.Logf("Original had UUID but normalized doesn't have {id}: %s -> %s", endpoint, normalized)
			return false
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// Additional property test: Idempotence of normalization
// Normalizing an already normalized endpoint should not change it
func TestProperty_NormalizationIdempotence(t *testing.T) {
	property := func(endpoint string) bool {
		// Normalize once
		normalized1 := normalizeEndpoint(endpoint)

		// Normalize again
		normalized2 := normalizeEndpoint(normalized1)

		// They should be identical
		if normalized1 != normalized2 {
			t.Logf("Normalization is not idempotent: %s -> %s -> %s", endpoint, normalized1, normalized2)
			return false
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}
