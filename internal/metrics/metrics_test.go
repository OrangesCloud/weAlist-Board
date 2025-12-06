package metrics

import (
	"testing"
)

// TestMetricsInitialization tests that all metrics are properly initialized
func TestMetricsInitialization(t *testing.T) {
	m := getTestMetrics()

	// Test that all metrics are non-nil
	if m.HTTPRequestsTotal == nil {
		t.Error("HTTPRequestsTotal should not be nil")
	}
	if m.HTTPRequestDuration == nil {
		t.Error("HTTPRequestDuration should not be nil")
	}
	if m.DBConnectionsOpen == nil {
		t.Error("DBConnectionsOpen should not be nil")
	}
	if m.DBConnectionsInUse == nil {
		t.Error("DBConnectionsInUse should not be nil")
	}
	if m.DBConnectionsIdle == nil {
		t.Error("DBConnectionsIdle should not be nil")
	}
	if m.DBConnectionsMax == nil {
		t.Error("DBConnectionsMax should not be nil")
	}
	if m.DBConnectionWaitTotal == nil {
		t.Error("DBConnectionWaitTotal should not be nil")
	}
	if m.DBConnectionWaitDuration == nil {
		t.Error("DBConnectionWaitDuration should not be nil")
	}
	if m.DBQueryDuration == nil {
		t.Error("DBQueryDuration should not be nil")
	}
	if m.DBQueryErrors == nil {
		t.Error("DBQueryErrors should not be nil")
	}
	if m.ExternalAPIRequestDuration == nil {
		t.Error("ExternalAPIRequestDuration should not be nil")
	}
	if m.ExternalAPIRequestsTotal == nil {
		t.Error("ExternalAPIRequestsTotal should not be nil")
	}
	if m.ExternalAPIErrors == nil {
		t.Error("ExternalAPIErrors should not be nil")
	}
	if m.ProjectsTotal == nil {
		t.Error("ProjectsTotal should not be nil")
	}
	if m.BoardsTotal == nil {
		t.Error("BoardsTotal should not be nil")
	}
	if m.ProjectCreatedTotal == nil {
		t.Error("ProjectCreatedTotal should not be nil")
	}
	if m.BoardCreatedTotal == nil {
		t.Error("BoardCreatedTotal should not be nil")
	}
}

// Property 14: 메트릭 네이밍 규칙 - snake_case
// Feature: board-service-prometheus-metrics, Property 14: All metric names must use snake_case
// Validates: Requirements 9.1
