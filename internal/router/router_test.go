package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"project-board-api/internal/client"
	"project-board-api/internal/metrics"
)

// setupTestRouter creates a test router with minimal configuration
func setupTestRouter(basePath string, m *metrics.Metrics) *Config {
	// Create in-memory SQLite database for testing
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}

	logger := zap.NewNop()

	// Create mock user client
	mockUserClient := &mockUserClient{}

	return &Config{
		DB:         db,
		Logger:     logger,
		JWTSecret:  "test-secret",
		UserClient: mockUserClient,
		BasePath:   basePath,
		Metrics:    m,
	}
}

// mockUserClient is a minimal mock implementation
type mockUserClient struct{}

func (m *mockUserClient) ValidateWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID, token string) (bool, error) {
	return false, nil
}

func (m *mockUserClient) GetUserProfile(ctx context.Context, userID uuid.UUID, token string) (*client.UserProfile, error) {
	return nil, nil
}

func (m *mockUserClient) GetWorkspaceProfile(ctx context.Context, workspaceID, userID uuid.UUID, token string) (*client.WorkspaceProfile, error) {
	return nil, nil
}

func (m *mockUserClient) GetWorkspace(ctx context.Context, workspaceID uuid.UUID, token string) (*client.Workspace, error) {
	return nil, nil
}

func (m *mockUserClient) ValidateToken(ctx context.Context, tokenStr string) (uuid.UUID, error) {
	return uuid.Nil, nil
}

// TestMetricsEndpoint_RootPath tests /metrics endpoint at root path
func TestMetricsEndpoint_RootPath(t *testing.T) {
	// Use default registry for this test to match production behavior
	registry := prometheus.NewRegistry()
	logger := zap.NewNop()
	m := metrics.NewWithRegistry(registry, logger)
	
	cfg := setupTestRouter("", m)
	router := Setup(*cfg)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// HTTP 200 응답 확인
	assert.Equal(t, http.StatusOK, w.Code, "Expected status 200")

	// Content-Type: text/plain 확인
	contentType := w.Header().Get("Content-Type")
	assert.Contains(t, contentType, "text/plain", "Expected Content-Type to contain text/plain")

	// Prometheus 형식 검증 - 응답 본문에 메트릭이 포함되어 있는지 확인
	body := w.Body.String()
	assert.NotEmpty(t, body, "Response body should not be empty")

	// 기본 Prometheus 메트릭 형식 검증 (# HELP, # TYPE 포함)
	assert.Contains(t, body, "# HELP", "Response should contain Prometheus HELP comments")
	assert.Contains(t, body, "# TYPE", "Response should contain Prometheus TYPE comments")

	// Go 런타임 메트릭은 항상 포함됨 (기본 레지스트리 사용)
	assert.Contains(t, body, "go_goroutines", "Response should contain Go runtime metrics")
}

// TestMetricsEndpoint_NoAuthentication tests that /metrics does not require authentication
func TestMetricsEndpoint_NoAuthentication(t *testing.T) {
	registry := prometheus.NewRegistry()
	logger := zap.NewNop()
	m := metrics.NewWithRegistry(registry, logger)
	
	cfg := setupTestRouter("", m)
	router := Setup(*cfg)

	// 인증 헤더 없이 요청
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// 인증 없이 접근 가능 확인 (401이 아닌 200 응답)
	assert.Equal(t, http.StatusOK, w.Code, "Metrics endpoint should be accessible without authentication")
}

// TestMetricsEndpoint_WithBasePath tests /metrics endpoint with base path configured
func TestMetricsEndpoint_WithBasePath(t *testing.T) {
	registry := prometheus.NewRegistry()
	logger := zap.NewNop()
	m := metrics.NewWithRegistry(registry, logger)
	
	basePath := "/api/boards"
	cfg := setupTestRouter(basePath, m)
	router := Setup(*cfg)

	t.Run("root path /metrics works", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "Root /metrics should work")
		assert.Contains(t, w.Header().Get("Content-Type"), "text/plain")
	})

	t.Run("base path /api/boards/metrics works", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, basePath+"/metrics", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "Base path /api/boards/metrics should work")
		assert.Contains(t, w.Header().Get("Content-Type"), "text/plain")
	})
}

// TestMetricsEndpoint_ContainsAllMetrics tests that all expected metrics are exposed
func TestMetricsEndpoint_ContainsAllMetrics(t *testing.T) {
	// Create a new registry and gather metrics from it
	registry := prometheus.NewRegistry()
	logger := zap.NewNop()
	_ = metrics.NewWithRegistry(registry, logger)
	
	// Gather metrics directly from the custom registry
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	// Convert to map for easier checking
	metricNames := make(map[string]bool)
	for _, mf := range metricFamilies {
		metricNames[mf.GetName()] = true
	}

	// Gauge 메트릭은 초기화 시 바로 등록되므로 확인 가능
	// Counter와 Histogram은 값이 기록되기 전까지는 나타나지 않을 수 있음
	expectedGaugeMetrics := []string{
		// 데이터베이스 메트릭 (Gauge)
		"board_service_db_connections_open",
		"board_service_db_connections_in_use",
		"board_service_db_connections_idle",
		"board_service_db_connections_max",
		// 비즈니스 메트릭 (Gauge)
		"board_service_projects_total",
		"board_service_boards_total",
	}

	for _, metric := range expectedGaugeMetrics {
		assert.True(t, metricNames[metric], "Registry should contain metric: %s", metric)
	}
	
	// Counter 메트릭도 초기화 시 등록됨
	expectedCounterMetrics := []string{
		"board_service_db_connection_wait_total",
		"board_service_db_connection_wait_duration_seconds_total",
		"board_service_project_created_total",
		"board_service_board_created_total",
	}
	
	for _, metric := range expectedCounterMetrics {
		assert.True(t, metricNames[metric], "Registry should contain metric: %s", metric)
	}
}

// TestMetricsEndpoint_PrometheusFormat tests Prometheus format validation
func TestMetricsEndpoint_PrometheusFormat(t *testing.T) {
	registry := prometheus.NewRegistry()
	logger := zap.NewNop()
	m := metrics.NewWithRegistry(registry, logger)
	
	cfg := setupTestRouter("", m)
	router := Setup(*cfg)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()

	// Prometheus 형식 검증
	lines := strings.Split(body, "\n")
	
	hasHelpLine := false
	hasTypeLine := false
	hasMetricLine := false

	for _, line := range lines {
		if strings.HasPrefix(line, "# HELP") {
			hasHelpLine = true
		}
		if strings.HasPrefix(line, "# TYPE") {
			hasTypeLine = true
		}
		// 메트릭 라인은 # 으로 시작하지 않고 값을 포함
		if !strings.HasPrefix(line, "#") && strings.Contains(line, " ") && line != "" {
			hasMetricLine = true
		}
	}

	assert.True(t, hasHelpLine, "Should have at least one HELP line")
	assert.True(t, hasTypeLine, "Should have at least one TYPE line")
	assert.True(t, hasMetricLine, "Should have at least one metric line with value")
}
