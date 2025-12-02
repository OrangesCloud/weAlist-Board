package metrics

import (
	"testing"
	"testing/quick"
	"time"
)

func TestCategorizeStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       string
	}{
		// 2xx codes
		{"200 OK", 200, "2xx"},
		{"201 Created", 201, "2xx"},
		{"204 No Content", 204, "2xx"},
		{"299 Edge case", 299, "2xx"},

		// 3xx codes
		{"300 Multiple Choices", 300, "3xx"},
		{"301 Moved Permanently", 301, "3xx"},
		{"302 Found", 302, "3xx"},
		{"304 Not Modified", 304, "3xx"},
		{"399 Edge case", 399, "3xx"},

		// 4xx codes
		{"400 Bad Request", 400, "4xx"},
		{"401 Unauthorized", 401, "4xx"},
		{"403 Forbidden", 403, "4xx"},
		{"404 Not Found", 404, "4xx"},
		{"499 Edge case", 499, "4xx"},

		// 5xx codes
		{"500 Internal Server Error", 500, "5xx"},
		{"502 Bad Gateway", 502, "5xx"},
		{"503 Service Unavailable", 503, "5xx"},
		{"599 Edge case", 599, "5xx"},
		{"600 Beyond standard", 600, "5xx"},

		// Edge cases
		{"100 Continue", 100, "unknown"},
		{"199 Informational", 199, "unknown"},
		{"0 Invalid", 0, "unknown"},
		{"-1 Negative", -1, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := categorizeStatus(tt.statusCode)
			if got != tt.want {
				t.Errorf("categorizeStatus(%d) = %v, want %v", tt.statusCode, got, tt.want)
			}
		})
	}
}

func TestShouldSkipEndpoint(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		// Should skip
		{"metrics endpoint", "/metrics", true},
		{"health endpoint", "/health", true},
		{"metrics with base path", "/api/boards/metrics", true},
		{"health with base path", "/api/boards/health", true},

		// Should not skip
		{"root path", "/", false},
		{"api endpoint", "/api/boards", false},
		{"projects endpoint", "/api/boards/projects", false},
		{"boards endpoint", "/api/boards/boards", false},
		{"similar to metrics", "/api/boards/metrics-data", false},
		{"similar to health", "/api/boards/healthcheck", false},
		{"metrics in middle", "/api/metrics/boards", false},
		{"health in middle", "/api/health/boards", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldSkipEndpoint(tt.path)
			if got != tt.want {
				t.Errorf("ShouldSkipEndpoint(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestRecordHTTPRequest(t *testing.T) {
	m := getTestMetrics()

	// Test recording a request
	method := "GET"
	endpoint := "/api/boards/projects"
	statusCode := 200
	duration := 100 * time.Millisecond

	// This should not panic
	m.RecordHTTPRequest(method, endpoint, statusCode, duration)

	// Test with different status codes
	testCases := []struct {
		method     string
		endpoint   string
		statusCode int
		duration   time.Duration
	}{
		{"GET", "/api/boards", 200, 50 * time.Millisecond},
		{"POST", "/api/boards/projects", 201, 150 * time.Millisecond},
		{"PUT", "/api/boards/projects/1", 204, 75 * time.Millisecond},
		{"DELETE", "/api/boards/projects/1", 204, 80 * time.Millisecond},
		{"GET", "/api/boards/projects", 404, 20 * time.Millisecond},
		{"POST", "/api/boards/projects", 500, 200 * time.Millisecond},
	}

	for _, tc := range testCases {
		t.Run(tc.method+" "+tc.endpoint, func(t *testing.T) {
			// Should not panic
			m.RecordHTTPRequest(tc.method, tc.endpoint, tc.statusCode, tc.duration)
		})
	}
}

// Property 3: HTTP status code 분류
// Feature: board-service-prometheus-metrics, Property 3: HTTP status code classification
// For any HTTP status code, categorizeStatus should correctly classify it as 2xx, 3xx, 4xx, 5xx, or unknown
// Validates: Requirements 1.5
func TestProperty_HTTPStatusCodeClassification(t *testing.T) {
	// Property: For any status code, the categorization should be consistent and correct
	property := func(code int) bool {
		result := categorizeStatus(code)

		// Check that the result is one of the valid categories
		validCategories := map[string]bool{
			"2xx":     true,
			"3xx":     true,
			"4xx":     true,
			"5xx":     true,
			"unknown": true,
		}

		if !validCategories[result] {
			t.Logf("Invalid category returned: %s for code %d", result, code)
			return false
		}

		// Verify the classification logic
		switch {
		case code >= 200 && code < 300:
			if result != "2xx" {
				t.Logf("Expected 2xx for code %d, got %s", code, result)
				return false
			}
		case code >= 300 && code < 400:
			if result != "3xx" {
				t.Logf("Expected 3xx for code %d, got %s", code, result)
				return false
			}
		case code >= 400 && code < 500:
			if result != "4xx" {
				t.Logf("Expected 4xx for code %d, got %s", code, result)
				return false
			}
		case code >= 500:
			if result != "5xx" {
				t.Logf("Expected 5xx for code %d, got %s", code, result)
				return false
			}
		default:
			if result != "unknown" {
				t.Logf("Expected unknown for code %d, got %s", code, result)
				return false
			}
		}

		return true
	}

	// Run the property test with 100 iterations
	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}
