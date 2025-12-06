package metrics

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

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
