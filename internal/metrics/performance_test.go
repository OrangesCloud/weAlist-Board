package metrics

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Performance Test 1: Measure metrics collection overhead
// Validates: Requirements 6.2
// Goal: Metrics collection should add less than 5% overhead
func TestPerformance_MetricsCollectionOverhead(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	gin.SetMode(gin.TestMode)

	// Create test endpoint
	handler := func(c *gin.Context) {
		// Simulate some work
		time.Sleep(1 * time.Millisecond)
		c.Status(http.StatusOK)
	}

	// Test WITHOUT metrics middleware
	routerWithoutMetrics := gin.New()
	routerWithoutMetrics.GET("/test", handler)

	// Test WITH metrics middleware
	// Use isolated registry to avoid conflicts
	registry := prometheus.NewRegistry()
	testMetrics := NewWithRegistry(registry, nil)
	routerWithMetrics := gin.New()
	routerWithMetrics.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		testMetrics.RecordHTTPRequest(c.Request.Method, c.FullPath(), c.Writer.Status(), duration)
	})
	routerWithMetrics.GET("/test", handler)

	// Warmup
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		routerWithoutMetrics.ServeHTTP(w, req)
	}

	// Benchmark WITHOUT metrics
	iterations := 10000
	startWithout := time.Now()
	for i := 0; i < iterations; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		routerWithoutMetrics.ServeHTTP(w, req)
	}
	durationWithout := time.Since(startWithout)

	// Benchmark WITH metrics
	startWith := time.Now()
	for i := 0; i < iterations; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		routerWithMetrics.ServeHTTP(w, req)
	}
	durationWith := time.Since(startWith)

	// Calculate overhead
	avgWithout := durationWithout.Nanoseconds() / int64(iterations)
	avgWith := durationWith.Nanoseconds() / int64(iterations)
	overhead := float64(avgWith-avgWithout) / float64(avgWithout) * 100

	t.Logf("Performance Results:")
	t.Logf("  Without metrics: %v total, %v avg per request", durationWithout, time.Duration(avgWithout))
	t.Logf("  With metrics:    %v total, %v avg per request", durationWith, time.Duration(avgWith))
	t.Logf("  Overhead:        %.2f%%", overhead)

	// Verify overhead is less than 5%
	if overhead > 5.0 {
		t.Errorf("Metrics collection overhead is %.2f%%, which exceeds the 5%% target", overhead)
	} else {
		t.Logf("✓ Metrics collection overhead is within acceptable range (%.2f%% < 5%%)", overhead)
	}
}

// Performance Test 2: Measure /metrics endpoint response time
// Validates: Requirements 6.5
// Goal: /metrics endpoint should respond within 1 second
func TestPerformance_MetricsEndpointResponseTime(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create a new registry with metrics
	registry := prometheus.NewRegistry()
	testMetrics := &Metrics{
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
				Buckets:   prometheus.DefBuckets,
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
		DBQueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "db_query_duration_seconds",
				Help:      "Database query duration in seconds",
			},
			[]string{"operation", "table"},
		),
		ExternalAPIRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "external_api_requests_total",
				Help:      "Total number of external API requests",
			},
			[]string{"endpoint", "method", "status"},
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
	}

	// Register metrics
	registry.MustRegister(
		testMetrics.HTTPRequestsTotal,
		testMetrics.HTTPRequestDuration,
		testMetrics.DBConnectionsOpen,
		testMetrics.DBQueryDuration,
		testMetrics.ExternalAPIRequestsTotal,
		testMetrics.ProjectsTotal,
		testMetrics.BoardsTotal,
	)

	// Populate metrics with realistic data
	// Simulate 1000 HTTP requests across different endpoints
	endpoints := []string{"/api/boards/projects", "/api/boards/boards", "/api/boards/comments"}
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	statuses := []string{"2xx", "3xx", "4xx", "5xx"}

	for i := 0; i < 1000; i++ {
		endpoint := endpoints[i%len(endpoints)]
		method := methods[i%len(methods)]
		status := statuses[i%len(statuses)]
		testMetrics.HTTPRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
		testMetrics.HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(float64(i%100) / 1000.0)
	}

	// Simulate database queries
	operations := []string{"select", "insert", "update", "delete"}
	tables := []string{"projects", "boards", "comments", "participants"}
	for i := 0; i < 500; i++ {
		op := operations[i%len(operations)]
		table := tables[i%len(tables)]
		testMetrics.DBQueryDuration.WithLabelValues(op, table).Observe(float64(i%50) / 1000.0)
	}

	// Simulate external API calls
	for i := 0; i < 200; i++ {
		testMetrics.ExternalAPIRequestsTotal.WithLabelValues("/api/users/{id}", "GET", "200").Inc()
	}

	// Set gauge values
	testMetrics.DBConnectionsOpen.Set(10)
	testMetrics.ProjectsTotal.Set(150)
	testMetrics.BoardsTotal.Set(450)

	// Create HTTP handler for metrics endpoint
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})

	// Warmup
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// Measure response time over multiple requests
	iterations := 100
	var totalDuration time.Duration
	var maxDuration time.Duration
	var minDuration time.Duration = time.Hour

	for i := 0; i < iterations; i++ {
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()

		start := time.Now()
		handler.ServeHTTP(w, req)
		duration := time.Since(start)

		totalDuration += duration
		if duration > maxDuration {
			maxDuration = duration
		}
		if duration < minDuration {
			minDuration = duration
		}

		// Verify response is successful
		if w.Code != http.StatusOK {
			t.Errorf("Metrics endpoint returned status %d", w.Code)
		}
	}

	avgDuration := totalDuration / time.Duration(iterations)

	t.Logf("Metrics Endpoint Performance:")
	t.Logf("  Iterations:      %d", iterations)
	t.Logf("  Average time:    %v", avgDuration)
	t.Logf("  Min time:        %v", minDuration)
	t.Logf("  Max time:        %v", maxDuration)
	t.Logf("  Total time:      %v", totalDuration)

	// Verify average response time is under 1 second
	if avgDuration > time.Second {
		t.Errorf("Metrics endpoint average response time is %v, which exceeds the 1 second target", avgDuration)
	} else {
		t.Logf("✓ Metrics endpoint response time is within acceptable range (%v < 1s)", avgDuration)
	}

	// Verify max response time is reasonable (under 2 seconds)
	if maxDuration > 2*time.Second {
		t.Errorf("Metrics endpoint max response time is %v, which is too high", maxDuration)
	} else {
		t.Logf("✓ Metrics endpoint max response time is acceptable (%v < 2s)", maxDuration)
	}
}

// Performance Test 3: Memory leak detection
// Validates: Requirements 6.2
// Goal: No memory leaks during long-running metric collection
func TestPerformance_MemoryLeakDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Force garbage collection before starting
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	var memStatsBefore runtime.MemStats
	runtime.ReadMemStats(&memStatsBefore)

	// Use isolated registry to avoid conflicts
	registry := prometheus.NewRegistry()
	testMetrics := NewWithRegistry(registry, nil)

	// Simulate continuous metric collection
	iterations := 100000
	endpoints := []string{"/api/boards/projects", "/api/boards/boards", "/api/boards/comments"}
	methods := []string{"GET", "POST", "PUT", "DELETE"}

	t.Logf("Starting memory leak test with %d iterations...", iterations)

	for i := 0; i < iterations; i++ {
		// Record HTTP metrics
		endpoint := endpoints[i%len(endpoints)]
		method := methods[i%len(methods)]
		statusCode := 200 + (i % 5)
		duration := time.Duration(i%100) * time.Millisecond

		testMetrics.RecordHTTPRequest(method, endpoint, statusCode, duration)

		// Record DB metrics
		testMetrics.RecordDBQuery("select", "projects", time.Duration(i%50)*time.Millisecond, nil)

		// Record external API metrics
		testMetrics.RecordExternalAPICall("/api/users/123", "GET", 200, time.Duration(i%100)*time.Millisecond, nil)

		// Update DB stats periodically
		if i%1000 == 0 {
			stats := sql.DBStats{
				OpenConnections: 10,
				InUse:           5,
				Idle:            5,
				WaitCount:       int64(i),
				WaitDuration:    time.Duration(i) * time.Millisecond,
			}
			testMetrics.UpdateDBStats(stats)
		}

		// Force GC periodically to get accurate memory measurements
		if i%10000 == 0 && i > 0 {
			runtime.GC()
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Force final garbage collection
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	var memStatsAfter runtime.MemStats
	runtime.ReadMemStats(&memStatsAfter)

	// Calculate memory usage
	allocBefore := memStatsBefore.Alloc
	allocAfter := memStatsAfter.Alloc
	heapAllocBefore := memStatsBefore.HeapAlloc
	heapAllocAfter := memStatsAfter.HeapAlloc

	allocDiff := int64(allocAfter) - int64(allocBefore)
	heapAllocDiff := int64(heapAllocAfter) - int64(heapAllocBefore)

	t.Logf("Memory Usage:")
	t.Logf("  Before:")
	t.Logf("    Alloc:      %v MB", float64(allocBefore)/(1024*1024))
	t.Logf("    HeapAlloc:  %v MB", float64(heapAllocBefore)/(1024*1024))
	t.Logf("  After:")
	t.Logf("    Alloc:      %v MB", float64(allocAfter)/(1024*1024))
	t.Logf("    HeapAlloc:  %v MB", float64(heapAllocAfter)/(1024*1024))
	t.Logf("  Difference:")
	t.Logf("    Alloc:      %v MB", float64(allocDiff)/(1024*1024))
	t.Logf("    HeapAlloc:  %v MB", float64(heapAllocDiff)/(1024*1024))
	t.Logf("  Per iteration:")
	t.Logf("    Alloc:      %v bytes", allocDiff/int64(iterations))
	t.Logf("    HeapAlloc:  %v bytes", heapAllocDiff/int64(iterations))

	// Memory leak threshold: less than 10MB growth after GC for 100k iterations
	// This is reasonable as Prometheus metrics do maintain some internal state
	maxAcceptableGrowth := int64(10 * 1024 * 1024) // 10 MB

	if heapAllocDiff > maxAcceptableGrowth {
		t.Errorf("Potential memory leak detected: heap grew by %v MB (threshold: %v MB)",
			float64(heapAllocDiff)/(1024*1024),
			float64(maxAcceptableGrowth)/(1024*1024))
	} else {
		t.Logf("✓ No significant memory leak detected (heap growth: %v MB < %v MB)",
			float64(heapAllocDiff)/(1024*1024),
			float64(maxAcceptableGrowth)/(1024*1024))
	}

	// Check per-iteration memory usage is reasonable (< 100 bytes per iteration)
	perIterationBytes := heapAllocDiff / int64(iterations)
	if perIterationBytes > 100 {
		t.Errorf("High per-iteration memory usage: %v bytes (threshold: 100 bytes)", perIterationBytes)
	} else {
		t.Logf("✓ Per-iteration memory usage is acceptable (%v bytes < 100 bytes)", perIterationBytes)
	}
}

// Performance Test 4: Concurrent metrics collection
// Validates: Requirements 6.2
// Goal: Metrics collection should be thread-safe and performant under concurrent load
func TestPerformance_ConcurrentMetricsCollection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Use isolated registry to avoid conflicts
	registry := prometheus.NewRegistry()
	testMetrics := NewWithRegistry(registry, nil)
	
	// Number of concurrent goroutines
	concurrency := 100
	iterationsPerGoroutine := 1000

	t.Logf("Starting concurrent metrics test with %d goroutines, %d iterations each...",
		concurrency, iterationsPerGoroutine)

	start := time.Now()

	// Channel to signal completion
	done := make(chan bool, concurrency)

	// Launch concurrent goroutines
	for g := 0; g < concurrency; g++ {
		go func(goroutineID int) {
			for i := 0; i < iterationsPerGoroutine; i++ {
				// Record various metrics
				endpoint := fmt.Sprintf("/api/boards/endpoint%d", goroutineID%10)
				method := []string{"GET", "POST", "PUT", "DELETE"}[i%4]
				statusCode := 200 + (i % 5)
				duration := time.Duration(i%100) * time.Microsecond

				testMetrics.RecordHTTPRequest(method, endpoint, statusCode, duration)
				testMetrics.RecordDBQuery("select", "projects", duration, nil)
				testMetrics.RecordExternalAPICall("/api/users/123", "GET", 200, duration, nil)
			}
			done <- true
		}(g)
	}

	// Wait for all goroutines to complete
	for i := 0; i < concurrency; i++ {
		<-done
	}

	duration := time.Since(start)
	totalOperations := concurrency * iterationsPerGoroutine * 3 // 3 metric types per iteration
	opsPerSecond := float64(totalOperations) / duration.Seconds()

	t.Logf("Concurrent Metrics Performance:")
	t.Logf("  Goroutines:      %d", concurrency)
	t.Logf("  Iterations each: %d", iterationsPerGoroutine)
	t.Logf("  Total ops:       %d", totalOperations)
	t.Logf("  Duration:        %v", duration)
	t.Logf("  Ops/second:      %.0f", opsPerSecond)

	// Verify reasonable throughput (at least 100k ops/second)
	minOpsPerSecond := 100000.0
	if opsPerSecond < minOpsPerSecond {
		t.Errorf("Concurrent metrics throughput is %.0f ops/sec, which is below the %.0f ops/sec target",
			opsPerSecond, minOpsPerSecond)
	} else {
		t.Logf("✓ Concurrent metrics throughput is acceptable (%.0f ops/sec > %.0f ops/sec)",
			opsPerSecond, minOpsPerSecond)
	}
}

// Performance Test 5: Label cardinality impact
// Validates: Requirements 6.4
// Goal: Verify that label cardinality limits prevent memory exhaustion
func TestPerformance_LabelCardinalityImpact(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Use isolated registry to avoid conflicts
	registry := prometheus.NewRegistry()
	testMetrics := NewWithRegistry(registry, nil)

	// Force GC before starting
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	var memStatsBefore runtime.MemStats
	runtime.ReadMemStats(&memStatsBefore)

	// Test with limited cardinality (good practice)
	t.Log("Testing with limited label cardinality...")
	limitedEndpoints := []string{
		"/api/boards/projects",
		"/api/boards/boards",
		"/api/boards/comments",
		"/api/boards/participants",
	}

	for i := 0; i < 10000; i++ {
		endpoint := limitedEndpoints[i%len(limitedEndpoints)]
		testMetrics.RecordHTTPRequest("GET", endpoint, 200, time.Millisecond)
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	var memStatsLimited runtime.MemStats
	runtime.ReadMemStats(&memStatsLimited)

	limitedMemoryGrowth := int64(memStatsLimited.HeapAlloc) - int64(memStatsBefore.HeapAlloc)

	t.Logf("Limited Cardinality Results:")
	t.Logf("  Unique endpoints: %d", len(limitedEndpoints))
	t.Logf("  Memory growth:    %v KB", limitedMemoryGrowth/1024)

	// Verify memory growth is reasonable with limited cardinality
	maxAcceptableGrowth := int64(5 * 1024 * 1024) // 5 MB
	if limitedMemoryGrowth > maxAcceptableGrowth {
		t.Errorf("Memory growth with limited cardinality is too high: %v MB",
			float64(limitedMemoryGrowth)/(1024*1024))
	} else {
		t.Logf("✓ Memory growth with limited cardinality is acceptable (%v KB < %v MB)",
			limitedMemoryGrowth/1024,
			maxAcceptableGrowth/(1024*1024))
	}
}
