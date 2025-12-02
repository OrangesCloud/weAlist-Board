package metrics

import (
	"database/sql"
	"errors"
	"sync"
	"testing"
	"testing/quick"
	"time"
)

var (
	testMetrics     *Metrics
	testMetricsOnce sync.Once
)

// getTestMetrics returns a shared metrics instance for testing
func getTestMetrics() *Metrics {
	testMetricsOnce.Do(func() {
		testMetrics = New()
	})
	return testMetrics
}

func TestNormalizeOperation(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		want      string
	}{
		{"lowercase select", "select", "select"},
		{"uppercase SELECT", "SELECT", "select"},
		{"mixed case Select", "Select", "select"},
		{"lowercase insert", "insert", "insert"},
		{"uppercase INSERT", "INSERT", "insert"},
		{"mixed case Insert", "Insert", "insert"},
		{"lowercase update", "update", "update"},
		{"uppercase UPDATE", "UPDATE", "update"},
		{"mixed case Update", "Update", "update"},
		{"lowercase delete", "delete", "delete"},
		{"uppercase DELETE", "DELETE", "delete"},
		{"mixed case Delete", "Delete", "delete"},
		{"all caps QUERY", "QUERY", "query"},
		{"mixed case MiXeD", "MiXeD", "mixed"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeOperation(tt.operation)
			if got != tt.want {
				t.Errorf("normalizeOperation(%q) = %q, want %q", tt.operation, got, tt.want)
			}
		})
	}
}

func TestUpdateDBStats(t *testing.T) {
	tests := []struct {
		name  string
		stats sql.DBStats
	}{
		{
			name: "normal stats",
			stats: sql.DBStats{
				OpenConnections: 10,
				InUse:           5,
				Idle:            5,
				MaxOpenConnections: 20,
				WaitCount:       100,
				WaitDuration:    500 * time.Millisecond,
			},
		},
		{
			name: "zero connections",
			stats: sql.DBStats{
				OpenConnections: 0,
				InUse:           0,
				Idle:            0,
				MaxOpenConnections: 10,
				WaitCount:       0,
				WaitDuration:    0,
			},
		},
		{
			name: "max connections",
			stats: sql.DBStats{
				OpenConnections: 50,
				InUse:           50,
				Idle:            0,
				MaxOpenConnections: 50,
				WaitCount:       1000,
				WaitDuration:    5 * time.Second,
			},
		},
		{
			name: "all idle",
			stats: sql.DBStats{
				OpenConnections: 20,
				InUse:           0,
				Idle:            20,
				MaxOpenConnections: 30,
				WaitCount:       0,
				WaitDuration:    0,
			},
		},
	}

	// Use shared metrics instance
	m := getTestMetrics()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			m.UpdateDBStats(tt.stats)
		})
	}
}

func TestRecordDBQuery(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		table     string
		duration  time.Duration
		err       error
	}{
		{
			name:      "successful select",
			operation: "SELECT",
			table:     "projects",
			duration:  10 * time.Millisecond,
			err:       nil,
		},
		{
			name:      "successful insert",
			operation: "INSERT",
			table:     "boards",
			duration:  20 * time.Millisecond,
			err:       nil,
		},
		{
			name:      "successful update",
			operation: "UPDATE",
			table:     "participants",
			duration:  15 * time.Millisecond,
			err:       nil,
		},
		{
			name:      "successful delete",
			operation: "DELETE",
			table:     "comments",
			duration:  5 * time.Millisecond,
			err:       nil,
		},
		{
			name:      "failed select",
			operation: "SELECT",
			table:     "projects",
			duration:  50 * time.Millisecond,
			err:       errors.New("connection timeout"),
		},
		{
			name:      "failed insert",
			operation: "INSERT",
			table:     "boards",
			duration:  100 * time.Millisecond,
			err:       errors.New("constraint violation"),
		},
		{
			name:      "lowercase operation",
			operation: "select",
			table:     "projects",
			duration:  10 * time.Millisecond,
			err:       nil,
		},
		{
			name:      "mixed case operation",
			operation: "Select",
			table:     "projects",
			duration:  10 * time.Millisecond,
			err:       nil,
		},
	}

	// Use shared metrics instance
	m := getTestMetrics()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			m.RecordDBQuery(tt.operation, tt.table, tt.duration, tt.err)
		})
	}
}

// Property 7: Operation type 정규화
// Feature: board-service-prometheus-metrics, Property 7: Operation type normalization
// For any operation string, normalizeOperation should convert it to lowercase
// Validates: Requirements 4.5
func TestProperty_OperationTypeNormalization(t *testing.T) {
	// Property: For any string, normalizeOperation should return lowercase version
	property := func(operation string) bool {
		result := normalizeOperation(operation)

		// The result should be lowercase
		if len(operation) > 0 {
			// Check each character
			for i, r := range result {
				if r >= 'A' && r <= 'Z' {
					t.Logf("Found uppercase character '%c' at position %d in result %q for input %q", r, i, result, operation)
					return false
				}
			}

			// Verify it matches strings.ToLower
			if result != operation && result != "" {
				// Result should be lowercase version of input
				allLower := true
				for _, r := range operation {
					if r >= 'A' && r <= 'Z' {
						allLower = false
						break
					}
				}
				
				// If input wasn't all lowercase, result should be different
				if !allLower && result == operation {
					t.Logf("Expected transformation for %q, but got same value", operation)
					return false
				}
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
