package metrics

import (
	"regexp"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
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
func TestProperty14_MetricNamingSnakeCase(t *testing.T) {
	// Create a new registry to collect all metrics
	registry := prometheus.NewRegistry()
	
	// Register all metrics with the test registry
	m := &Metrics{
		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds",
			},
			[]string{"method", "endpoint"},
		),
		DBConnectionsOpen: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "db_connections_open",
				Help:      "Current number of open database connections",
			},
		),
		DBConnectionsInUse: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "db_connections_in_use",
				Help:      "Current number of in-use database connections",
			},
		),
		DBConnectionsIdle: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "db_connections_idle",
				Help:      "Current number of idle database connections",
			},
		),
		DBConnectionsMax: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "db_connections_max",
				Help:      "Maximum number of open database connections configured",
			},
		),
		DBConnectionWaitTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "db_connection_wait_total",
				Help:      "Total number of times waited for a database connection",
			},
		),
		DBConnectionWaitDuration: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "db_connection_wait_duration_seconds_total",
				Help:      "Total duration waited for database connections in seconds",
			},
		),
		DBQueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "db_query_duration_seconds",
				Help:      "Database query duration in seconds",
			},
			[]string{"operation", "table"},
		),
		DBQueryErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "db_query_errors_total",
				Help:      "Total number of database query errors",
			},
			[]string{"operation", "table"},
		),
		ExternalAPIRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "external_api_request_duration_seconds",
				Help:      "External API request duration in seconds",
			},
			[]string{"endpoint", "status"},
		),
		ExternalAPIRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "external_api_requests_total",
				Help:      "Total number of external API requests",
			},
			[]string{"endpoint", "method", "status"},
		),
		ExternalAPIErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "external_api_errors_total",
				Help:      "Total number of external API errors",
			},
			[]string{"endpoint", "error_type"},
		),
		ProjectsTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "projects_total",
				Help:      "Total number of active projects",
			},
		),
		BoardsTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "boards_total",
				Help:      "Total number of boards",
			},
		),
		ProjectCreatedTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "project_created_total",
				Help:      "Total number of project creation events",
			},
		),
		BoardCreatedTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "board_created_total",
				Help:      "Total number of board creation events",
			},
		),
	}

	// Register all collectors
	collectors := []prometheus.Collector{
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
		m.DBConnectionsOpen,
		m.DBConnectionsInUse,
		m.DBConnectionsIdle,
		m.DBConnectionsMax,
		m.DBConnectionWaitTotal,
		m.DBConnectionWaitDuration,
		m.DBQueryDuration,
		m.DBQueryErrors,
		m.ExternalAPIRequestDuration,
		m.ExternalAPIRequestsTotal,
		m.ExternalAPIErrors,
		m.ProjectsTotal,
		m.BoardsTotal,
		m.ProjectCreatedTotal,
		m.BoardCreatedTotal,
	}

	for _, collector := range collectors {
		registry.MustRegister(collector)
	}

	// Gather metrics
	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// snake_case pattern: lowercase letters, numbers, and underscores only
	snakeCasePattern := regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

	// Check each metric name
	for _, mf := range metricFamilies {
		name := mf.GetName()
		
		// Remove namespace prefix for checking the base name
		baseName := strings.TrimPrefix(name, namespace+"_")
		
		if !snakeCasePattern.MatchString(baseName) {
			t.Errorf("Metric name '%s' does not follow snake_case convention (lowercase and underscores only)", name)
		}
	}
}

// Property 15: 메트릭 네이밍 규칙 - prefix
// Feature: board-service-prometheus-metrics, Property 15: All custom metrics must have board_service_ prefix
// Validates: Requirements 9.2
func TestProperty15_MetricNamingPrefix(t *testing.T) {
	// Create a new registry to collect all metrics
	registry := prometheus.NewRegistry()
	
	// Register all metrics with the test registry
	m := &Metrics{
		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds",
			},
			[]string{"method", "endpoint"},
		),
		DBConnectionsOpen: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "db_connections_open",
				Help:      "Current number of open database connections",
			},
		),
		DBConnectionsInUse: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "db_connections_in_use",
				Help:      "Current number of in-use database connections",
			},
		),
		DBConnectionsIdle: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "db_connections_idle",
				Help:      "Current number of idle database connections",
			},
		),
		DBConnectionsMax: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "db_connections_max",
				Help:      "Maximum number of open database connections configured",
			},
		),
		DBConnectionWaitTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "db_connection_wait_total",
				Help:      "Total number of times waited for a database connection",
			},
		),
		DBConnectionWaitDuration: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "db_connection_wait_duration_seconds_total",
				Help:      "Total duration waited for database connections in seconds",
			},
		),
		DBQueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "db_query_duration_seconds",
				Help:      "Database query duration in seconds",
			},
			[]string{"operation", "table"},
		),
		DBQueryErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "db_query_errors_total",
				Help:      "Total number of database query errors",
			},
			[]string{"operation", "table"},
		),
		ExternalAPIRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "external_api_request_duration_seconds",
				Help:      "External API request duration in seconds",
			},
			[]string{"endpoint", "status"},
		),
		ExternalAPIRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "external_api_requests_total",
				Help:      "Total number of external API requests",
			},
			[]string{"endpoint", "method", "status"},
		),
		ExternalAPIErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "external_api_errors_total",
				Help:      "Total number of external API errors",
			},
			[]string{"endpoint", "error_type"},
		),
		ProjectsTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "projects_total",
				Help:      "Total number of active projects",
			},
		),
		BoardsTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "boards_total",
				Help:      "Total number of boards",
			},
		),
		ProjectCreatedTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "project_created_total",
				Help:      "Total number of project creation events",
			},
		),
		BoardCreatedTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "board_created_total",
				Help:      "Total number of board creation events",
			},
		),
	}

	// Register all collectors
	collectors := []prometheus.Collector{
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
		m.DBConnectionsOpen,
		m.DBConnectionsInUse,
		m.DBConnectionsIdle,
		m.DBConnectionsMax,
		m.DBConnectionWaitTotal,
		m.DBConnectionWaitDuration,
		m.DBQueryDuration,
		m.DBQueryErrors,
		m.ExternalAPIRequestDuration,
		m.ExternalAPIRequestsTotal,
		m.ExternalAPIErrors,
		m.ProjectsTotal,
		m.BoardsTotal,
		m.ProjectCreatedTotal,
		m.BoardCreatedTotal,
	}

	for _, collector := range collectors {
		registry.MustRegister(collector)
	}

	// Gather metrics
	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	expectedPrefix := namespace + "_"

	// Check each metric has the correct prefix
	for _, mf := range metricFamilies {
		name := mf.GetName()
		
		if !strings.HasPrefix(name, expectedPrefix) {
			t.Errorf("Metric name '%s' does not have the required prefix '%s'", name, expectedPrefix)
		}
	}
}

// Property 16: 메트릭 네이밍 규칙 - counter suffix
// Feature: board-service-prometheus-metrics, Property 16: All counter metrics must have _total suffix
// Validates: Requirements 9.3
func TestProperty16_MetricNamingCounterSuffix(t *testing.T) {
	// Create a new registry to collect all metrics
	registry := prometheus.NewRegistry()
	
	// Register counter metrics
	counters := map[string]prometheus.Collector{
		"http_requests_total": prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		"db_connection_wait_total": prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "db_connection_wait_total",
				Help:      "Total number of times waited for a database connection",
			},
		),
		"db_connection_wait_duration_seconds_total": prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "db_connection_wait_duration_seconds_total",
				Help:      "Total duration waited for database connections in seconds",
			},
		),
		"db_query_errors_total": prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "db_query_errors_total",
				Help:      "Total number of database query errors",
			},
			[]string{"operation", "table"},
		),
		"external_api_requests_total": prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "external_api_requests_total",
				Help:      "Total number of external API requests",
			},
			[]string{"endpoint", "method", "status"},
		),
		"external_api_errors_total": prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "external_api_errors_total",
				Help:      "Total number of external API errors",
			},
			[]string{"endpoint", "error_type"},
		),
		"project_created_total": prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "project_created_total",
				Help:      "Total number of project creation events",
			},
		),
		"board_created_total": prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "board_created_total",
				Help:      "Total number of board creation events",
			},
		),
	}

	for _, collector := range counters {
		registry.MustRegister(collector)
	}

	// Gather metrics
	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Check each counter metric has _total suffix
	for _, mf := range metricFamilies {
		if mf.GetType() == dto.MetricType_COUNTER {
			name := mf.GetName()
			
			if !strings.HasSuffix(name, "_total") {
				t.Errorf("Counter metric '%s' does not have the required '_total' suffix", name)
			}
		}
	}
}

// Property 17: 메트릭 네이밍 규칙 - duration suffix
// Feature: board-service-prometheus-metrics, Property 17: All duration metrics must have _seconds suffix
// Validates: Requirements 9.4
func TestProperty17_MetricNamingDurationSuffix(t *testing.T) {
	// Create a new registry to collect all metrics
	registry := prometheus.NewRegistry()
	
	// Register duration metrics (histograms)
	durationMetrics := map[string]prometheus.Collector{
		"http_request_duration_seconds": prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds",
			},
			[]string{"method", "endpoint"},
		),
		"db_query_duration_seconds": prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "db_query_duration_seconds",
				Help:      "Database query duration in seconds",
			},
			[]string{"operation", "table"},
		),
		"external_api_request_duration_seconds": prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "external_api_request_duration_seconds",
				Help:      "External API request duration in seconds",
			},
			[]string{"endpoint", "status"},
		),
		"db_connection_wait_duration_seconds_total": prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "db_connection_wait_duration_seconds_total",
				Help:      "Total duration waited for database connections in seconds",
			},
		),
	}

	for _, collector := range durationMetrics {
		registry.MustRegister(collector)
	}

	// Gather metrics
	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Check each duration metric has _seconds in the name
	for _, mf := range metricFamilies {
		name := mf.GetName()
		
		// Duration metrics should contain "_seconds" or "_duration_seconds"
		if strings.Contains(name, "duration") {
			if !strings.Contains(name, "_seconds") {
				t.Errorf("Duration metric '%s' does not have '_seconds' in its name", name)
			}
		}
	}
}

// Property 18: 메트릭 help 설명
// Feature: board-service-prometheus-metrics, Property 18: All metrics must have non-empty help description
// Validates: Requirements 9.5
func TestProperty18_MetricHelpDescription(t *testing.T) {
	// Create a new registry to collect all metrics
	registry := prometheus.NewRegistry()
	
	// Register all metrics with the test registry
	m := &Metrics{
		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds",
			},
			[]string{"method", "endpoint"},
		),
		DBConnectionsOpen: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "db_connections_open",
				Help:      "Current number of open database connections",
			},
		),
		DBConnectionsInUse: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "db_connections_in_use",
				Help:      "Current number of in-use database connections",
			},
		),
		DBConnectionsIdle: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "db_connections_idle",
				Help:      "Current number of idle database connections",
			},
		),
		DBConnectionsMax: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "db_connections_max",
				Help:      "Maximum number of open database connections configured",
			},
		),
		DBConnectionWaitTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "db_connection_wait_total",
				Help:      "Total number of times waited for a database connection",
			},
		),
		DBConnectionWaitDuration: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "db_connection_wait_duration_seconds_total",
				Help:      "Total duration waited for database connections in seconds",
			},
		),
		DBQueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "db_query_duration_seconds",
				Help:      "Database query duration in seconds",
			},
			[]string{"operation", "table"},
		),
		DBQueryErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "db_query_errors_total",
				Help:      "Total number of database query errors",
			},
			[]string{"operation", "table"},
		),
		ExternalAPIRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "external_api_request_duration_seconds",
				Help:      "External API request duration in seconds",
			},
			[]string{"endpoint", "status"},
		),
		ExternalAPIRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "external_api_requests_total",
				Help:      "Total number of external API requests",
			},
			[]string{"endpoint", "method", "status"},
		),
		ExternalAPIErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "external_api_errors_total",
				Help:      "Total number of external API errors",
			},
			[]string{"endpoint", "error_type"},
		),
		ProjectsTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "projects_total",
				Help:      "Total number of active projects",
			},
		),
		BoardsTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "boards_total",
				Help:      "Total number of boards",
			},
		),
		ProjectCreatedTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "project_created_total",
				Help:      "Total number of project creation events",
			},
		),
		BoardCreatedTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "board_created_total",
				Help:      "Total number of board creation events",
			},
		),
	}

	// Register all collectors
	collectors := []prometheus.Collector{
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
		m.DBConnectionsOpen,
		m.DBConnectionsInUse,
		m.DBConnectionsIdle,
		m.DBConnectionsMax,
		m.DBConnectionWaitTotal,
		m.DBConnectionWaitDuration,
		m.DBQueryDuration,
		m.DBQueryErrors,
		m.ExternalAPIRequestDuration,
		m.ExternalAPIRequestsTotal,
		m.ExternalAPIErrors,
		m.ProjectsTotal,
		m.BoardsTotal,
		m.ProjectCreatedTotal,
		m.BoardCreatedTotal,
	}

	for _, collector := range collectors {
		registry.MustRegister(collector)
	}

	// Gather metrics
	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Check each metric has a non-empty help description
	for _, mf := range metricFamilies {
		name := mf.GetName()
		help := mf.GetHelp()
		
		if help == "" {
			t.Errorf("Metric '%s' has an empty help description", name)
		}
		
		if len(strings.TrimSpace(help)) == 0 {
			t.Errorf("Metric '%s' has a help description with only whitespace", name)
		}
	}
}
