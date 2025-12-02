package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

const (
	namespace = "board_service"
)

// Metrics holds all application metrics
type Metrics struct {
	// HTTP metrics
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec

	// Database metrics
	DBConnectionsOpen        prometheus.Gauge
	DBConnectionsInUse       prometheus.Gauge
	DBConnectionsIdle        prometheus.Gauge
	DBConnectionsMax         prometheus.Gauge
	DBConnectionWaitTotal    prometheus.Counter
	DBConnectionWaitDuration prometheus.Counter
	DBQueryDuration          *prometheus.HistogramVec
	DBQueryErrors            *prometheus.CounterVec

	// External API metrics
	ExternalAPIRequestDuration *prometheus.HistogramVec
	ExternalAPIRequestsTotal   *prometheus.CounterVec
	ExternalAPIErrors          *prometheus.CounterVec

	// Business metrics
	ProjectsTotal       prometheus.Gauge
	BoardsTotal         prometheus.Gauge
	ProjectCreatedTotal prometheus.Counter
	BoardCreatedTotal   prometheus.Counter

	// Logger for error reporting
	logger *zap.Logger
}

// New creates and registers all metrics with the default registry
func New() *Metrics {
	return NewWithRegistry(prometheus.DefaultRegisterer, nil)
}

// NewWithLogger creates and registers all metrics with the default registry and a logger
func NewWithLogger(logger *zap.Logger) *Metrics {
	return NewWithRegistry(prometheus.DefaultRegisterer, logger)
}

// NewWithRegistry creates and registers all metrics with a custom registry
func NewWithRegistry(registerer prometheus.Registerer, logger *zap.Logger) *Metrics {
	factory := promauto.With(registerer)
	
	// Use a no-op logger if none provided
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	
	return &Metrics{
		// HTTP metrics
		HTTPRequestsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		HTTPRequestDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "endpoint"},
		),

		// Database connection pool metrics
		DBConnectionsOpen: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "db_connections_open",
				Help:      "Current number of open database connections",
			},
		),
		DBConnectionsInUse: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "db_connections_in_use",
				Help:      "Current number of in-use database connections",
			},
		),
		DBConnectionsIdle: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "db_connections_idle",
				Help:      "Current number of idle database connections",
			},
		),
		DBConnectionsMax: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "db_connections_max",
				Help:      "Maximum number of open database connections configured",
			},
		),
		DBConnectionWaitTotal: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "db_connection_wait_total",
				Help:      "Total number of times waited for a database connection",
			},
		),
		DBConnectionWaitDuration: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "db_connection_wait_duration_seconds_total",
				Help:      "Total duration waited for database connections in seconds",
			},
		),

		// Database query metrics
		DBQueryDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "db_query_duration_seconds",
				Help:      "Database query duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5},
			},
			[]string{"operation", "table"},
		),
		DBQueryErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "db_query_errors_total",
				Help:      "Total number of database query errors",
			},
			[]string{"operation", "table"},
		),

		// External API metrics
		ExternalAPIRequestDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "external_api_request_duration_seconds",
				Help:      "External API request duration in seconds",
				Buckets:   []float64{.01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"endpoint", "status"},
		),
		ExternalAPIRequestsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "external_api_requests_total",
				Help:      "Total number of external API requests",
			},
			[]string{"endpoint", "method", "status"},
		),
		ExternalAPIErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "external_api_errors_total",
				Help:      "Total number of external API errors",
			},
			[]string{"endpoint", "error_type"},
		),

		// Business metrics
		ProjectsTotal: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "projects_total",
				Help:      "Total number of active projects",
			},
		),
		BoardsTotal: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "boards_total",
				Help:      "Total number of boards",
			},
		),
		ProjectCreatedTotal: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "project_created_total",
				Help:      "Total number of project creation events",
			},
		),
		BoardCreatedTotal: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "board_created_total",
				Help:      "Total number of board creation events",
			},
		),
		
		logger: logger,
	}
}

// safeExecute wraps metric operations with panic recovery
func (m *Metrics) safeExecute(operation string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			if m.logger != nil {
				m.logger.Error("Panic in metrics operation",
					zap.String("operation", operation),
					zap.Any("panic", r),
				)
			}
		}
	}()
	fn()
}
