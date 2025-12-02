package metrics

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// testProject is a simple project model for testing
type testProject struct {
	ID        string `gorm:"type:text;primaryKey"`
	Name      string `gorm:"type:varchar(255)"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (testProject) TableName() string {
	return "projects"
}

// testBoard is a simple board model for testing
type testBoard struct {
	ID        string `gorm:"type:text;primaryKey"`
	Title     string `gorm:"type:varchar(255)"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (testBoard) TableName() string {
	return "boards"
}

// setupCollectorTestDB creates an in-memory SQLite database for testing
func setupCollectorTestDB(t *testing.T) *gorm.DB {
	// Use a file-based database instead of :memory: to avoid connection issues
	// Each test gets a unique database file
	dbPath := t.TempDir() + "/test.db"
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	require.NoError(t, err, "Failed to open test database")

	// Auto-migrate test models
	err = db.AutoMigrate(&testProject{}, &testBoard{})
	require.NoError(t, err, "Failed to migrate test models")

	// Verify tables were created
	var tableCount int64
	err = db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('projects', 'boards')").Scan(&tableCount).Error
	require.NoError(t, err, "Failed to verify tables")
	require.Equal(t, int64(2), tableCount, "Both projects and boards tables should exist")

	return db
}

// TestNewBusinessMetricsCollector tests collector creation
func TestNewBusinessMetricsCollector(t *testing.T) {
	t.Parallel()
	db := setupCollectorTestDB(t)
	m := getTestMetrics()
	logger := zap.NewNop()

	collector := NewBusinessMetricsCollector(db, m, logger)

	assert.NotNil(t, collector, "Collector should not be nil")
	assert.NotNil(t, collector.db, "DB should not be nil")
	assert.NotNil(t, collector.metrics, "Metrics should not be nil")
	assert.NotNil(t, collector.logger, "Logger should not be nil")
	assert.NotNil(t, collector.ticker, "Ticker should not be nil")
	assert.NotNil(t, collector.done, "Done channel should not be nil")
}

// TestBusinessMetricsCollector_Collect tests the collect method
// **Feature: board-service-prometheus-metrics, Property 12: 비즈니스 메트릭 노출**
// **Validates: Requirements 7.1, 7.2**
func TestBusinessMetricsCollector_Collect(t *testing.T) {
	t.Parallel()
	db := setupCollectorTestDB(t)
	m := getTestMetrics()
	logger := zap.NewNop()

	// Insert test data
	projects := []testProject{
		{ID: uuid.New().String(), Name: "Project 1"},
		{ID: uuid.New().String(), Name: "Project 2"},
		{ID: uuid.New().String(), Name: "Project 3"},
	}
	for _, p := range projects {
		err := db.Create(&p).Error
		require.NoError(t, err)
	}

	boards := []testBoard{
		{ID: uuid.New().String(), Title: "Board 1"},
		{ID: uuid.New().String(), Title: "Board 2"},
		{ID: uuid.New().String(), Title: "Board 3"},
		{ID: uuid.New().String(), Title: "Board 4"},
		{ID: uuid.New().String(), Title: "Board 5"},
	}
	for _, b := range boards {
		err := db.Create(&b).Error
		require.NoError(t, err)
	}

	// Create collector
	collector := NewBusinessMetricsCollector(db, m, logger)

	// Manually trigger collection
	collector.collect()

	// Verify metrics
	projectsTotal := getGaugeValue(t, m.ProjectsTotal)
	boardsTotal := getGaugeValue(t, m.BoardsTotal)

	assert.Equal(t, float64(3), projectsTotal, "ProjectsTotal should be 3")
	assert.Equal(t, float64(5), boardsTotal, "BoardsTotal should be 5")
}

// TestBusinessMetricsCollector_CollectEmpty tests collection with empty tables
// **Feature: board-service-prometheus-metrics, Property 12: 비즈니스 메트릭 노출**
// **Validates: Requirements 7.1, 7.2**
func TestBusinessMetricsCollector_CollectEmpty(t *testing.T) {
	t.Parallel()
	db := setupCollectorTestDB(t)
	m := getTestMetrics()
	logger := zap.NewNop()

	// Create collector
	collector := NewBusinessMetricsCollector(db, m, logger)

	// Manually trigger collection
	collector.collect()

	// Verify metrics
	projectsTotal := getGaugeValue(t, m.ProjectsTotal)
	boardsTotal := getGaugeValue(t, m.BoardsTotal)

	assert.Equal(t, float64(0), projectsTotal, "ProjectsTotal should be 0")
	assert.Equal(t, float64(0), boardsTotal, "BoardsTotal should be 0")
}

// TestBusinessMetricsCollector_StartStop tests start and stop
// **Feature: board-service-prometheus-metrics, Property 12: 비즈니스 메트릭 노출**
// **Validates: Requirements 7.1, 7.2**
func TestBusinessMetricsCollector_StartStop(t *testing.T) {
	db := setupCollectorTestDB(t)
	m := getTestMetrics()
	logger := zap.NewNop()

	// Insert test data
	project := testProject{ID: uuid.New().String(), Name: "Test Project"}
	err := db.Create(&project).Error
	require.NoError(t, err)

	// Create collector with shorter interval for testing
	collector := NewBusinessMetricsCollector(db, m, logger)
	collector.ticker.Stop()
	collector.ticker = time.NewTicker(20 * time.Millisecond)

	// Start collector
	collector.Start()

	// Wait for at least one collection cycle
	time.Sleep(30 * time.Millisecond)

	// Verify metrics were collected
	projectsTotal := getGaugeValue(t, m.ProjectsTotal)
	assert.Equal(t, float64(1), projectsTotal, "ProjectsTotal should be 1")

	// Stop collector
	collector.Stop()

	// Wait to ensure goroutine exits
	time.Sleep(10 * time.Millisecond)

	// Test passes if no panic or deadlock occurs
}

// TestBusinessMetricsCollector_PeriodicCollection tests periodic collection
// **Feature: board-service-prometheus-metrics, Property 12: 비즈니스 메트릭 노출**
// **Validates: Requirements 7.1, 7.2, 7.5**
func TestBusinessMetricsCollector_PeriodicCollection(t *testing.T) {
	db := setupCollectorTestDB(t)
	m := getTestMetrics()
	logger := zap.NewNop()

	// Ensure database tables are properly initialized
	var tableCount int64
	err := db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='projects'").Scan(&tableCount).Error
	require.NoError(t, err, "Failed to check if projects table exists")
	require.Equal(t, int64(1), tableCount, "Projects table should exist")

	// Create collector with shorter interval for faster test execution
	collector := NewBusinessMetricsCollector(db, m, logger)
	collector.ticker.Stop()
	collector.ticker = time.NewTicker(20 * time.Millisecond)

	// Start collector
	collector.Start()
	defer collector.Stop()

	// Insert initial data
	project1 := testProject{ID: uuid.New().String(), Name: "Project 1"}
	err = db.Create(&project1).Error
	require.NoError(t, err)

	// Wait for at least 2 collection cycles (20ms * 2 = 40ms)
	time.Sleep(50 * time.Millisecond)

	// Verify initial count with retry logic to handle timing variations
	var projectsTotal float64
	for i := 0; i < 3; i++ {
		projectsTotal = getGaugeValue(t, m.ProjectsTotal)
		if projectsTotal == 1.0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	assert.Equal(t, float64(1), projectsTotal, "ProjectsTotal should be 1")

	// Insert more data
	project2 := testProject{ID: uuid.New().String(), Name: "Project 2"}
	err = db.Create(&project2).Error
	require.NoError(t, err)

	// Wait for at least 2 more collection cycles
	time.Sleep(50 * time.Millisecond)

	// Verify updated count with retry logic to handle timing variations
	var finalCount float64
	for i := 0; i < 3; i++ {
		finalCount = getGaugeValue(t, m.ProjectsTotal)
		if finalCount == 2.0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	assert.Equal(t, float64(2), finalCount, "ProjectsTotal should be 2")
}

// TestBusinessMetricsCollector_ImmediateCollection tests immediate collection on start
// **Feature: board-service-prometheus-metrics, Property 12: 비즈니스 메트릭 노출**
// **Validates: Requirements 7.1, 7.2**
func TestBusinessMetricsCollector_ImmediateCollection(t *testing.T) {
	db := setupCollectorTestDB(t)
	m := getTestMetrics()
	logger := zap.NewNop()

	// Insert test data before starting collector
	project := testProject{ID: uuid.New().String(), Name: "Test Project"}
	err := db.Create(&project).Error
	require.NoError(t, err)

	board := testBoard{ID: uuid.New().String(), Title: "Test Board"}
	err = db.Create(&board).Error
	require.NoError(t, err)

	// Create and start collector
	collector := NewBusinessMetricsCollector(db, m, logger)
	collector.Start()
	defer collector.Stop()

	// Wait a bit for immediate collection to complete
	time.Sleep(20 * time.Millisecond)

	// Verify metrics were collected immediately
	projectsTotal := getGaugeValue(t, m.ProjectsTotal)
	boardsTotal := getGaugeValue(t, m.BoardsTotal)

	assert.Equal(t, float64(1), projectsTotal, "ProjectsTotal should be 1")
	assert.Equal(t, float64(1), boardsTotal, "BoardsTotal should be 1")
}

// TestBusinessMetricsCollector_Integration tests full integration with event counters
// **Feature: board-service-prometheus-metrics, Property 13: 비즈니스 이벤트 카운팅**
// **Validates: Requirements 7.3, 7.4**
func TestBusinessMetricsCollector_Integration(t *testing.T) {
	t.Parallel()
	db := setupCollectorTestDB(t)
	m := getTestMetrics()
	logger := zap.NewNop()

	// Get initial counter values
	initialProjectCreated := getCounterValue(t, m.ProjectCreatedTotal)
	initialBoardCreated := getCounterValue(t, m.BoardCreatedTotal)

	// Simulate project creation event
	project := testProject{ID: uuid.New().String(), Name: "New Project"}
	err := db.Create(&project).Error
	require.NoError(t, err)
	m.IncrementProjectCreated()

	// Simulate board creation events
	board1 := testBoard{ID: uuid.New().String(), Title: "New Board 1"}
	err = db.Create(&board1).Error
	require.NoError(t, err)
	m.IncrementBoardCreated()

	board2 := testBoard{ID: uuid.New().String(), Title: "New Board 2"}
	err = db.Create(&board2).Error
	require.NoError(t, err)
	m.IncrementBoardCreated()

	// Create collector and manually trigger collection
	collector := NewBusinessMetricsCollector(db, m, logger)
	collector.collect()

	// Verify event counters incremented
	newProjectCreated := getCounterValue(t, m.ProjectCreatedTotal)
	newBoardCreated := getCounterValue(t, m.BoardCreatedTotal)

	assert.Greater(t, newProjectCreated, initialProjectCreated, "ProjectCreatedTotal should increment")
	assert.Greater(t, newBoardCreated, initialBoardCreated, "BoardCreatedTotal should increment")

	// Verify gauge values match actual counts
	projectsTotal := getGaugeValue(t, m.ProjectsTotal)
	boardsTotal := getGaugeValue(t, m.BoardsTotal)

	assert.Equal(t, float64(1), projectsTotal, "ProjectsTotal should be 1")
	assert.Equal(t, float64(2), boardsTotal, "BoardsTotal should be 2")
}

// TestBusinessMetricsCollector_ErrorHandling tests error handling during collection
// **Feature: board-service-prometheus-metrics, Property 12: 비즈니스 메트릭 노출**
// **Validates: Requirements 7.1, 7.2**
func TestBusinessMetricsCollector_ErrorHandling(t *testing.T) {
	db := setupCollectorTestDB(t)
	m := getTestMetrics()
	logger := zap.NewNop()

	// Create collector
	collector := NewBusinessMetricsCollector(db, m, logger)

	// Set initial values
	m.SetProjectsTotal(10)
	m.SetBoardsTotal(20)

	// Close the database to simulate error
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.Close()

	// Trigger collection (should handle error gracefully)
	collector.collect()

	// Verify metrics retain previous values (not updated due to error)
	projectsTotal := getGaugeValue(t, m.ProjectsTotal)
	boardsTotal := getGaugeValue(t, m.BoardsTotal)

	assert.Equal(t, float64(10), projectsTotal, "ProjectsTotal should retain previous value")
	assert.Equal(t, float64(20), boardsTotal, "BoardsTotal should retain previous value")

	// Test passes if no panic occurs
}

// TestBusinessMetricsCollector_ConcurrentAccess tests concurrent access safety
// **Feature: board-service-prometheus-metrics, Property 12: 비즈니스 메트릭 노출**
// **Validates: Requirements 7.1, 7.2**
func TestBusinessMetricsCollector_ConcurrentAccess(t *testing.T) {
	db := setupCollectorTestDB(t)
	m := getTestMetrics()
	logger := zap.NewNop()

	// Insert data sequentially to avoid table creation issues
	for i := 0; i < 5; i++ {
		project := testProject{ID: uuid.New().String(), Name: "Project"}
		err := db.Create(&project).Error
		require.NoError(t, err)
		m.IncrementProjectCreated()
	}

	for i := 0; i < 10; i++ {
		board := testBoard{ID: uuid.New().String(), Title: "Board"}
		err := db.Create(&board).Error
		require.NoError(t, err)
		m.IncrementBoardCreated()
	}

	// Create collector with shorter interval to test concurrent collection
	collector := NewBusinessMetricsCollector(db, m, logger)
	collector.ticker.Stop()
	collector.ticker = time.NewTicker(10 * time.Millisecond)

	// Start collector
	collector.Start()
	defer collector.Stop()

	// Concurrently trigger multiple collections
	done := make(chan bool, 3)
	for i := 0; i < 3; i++ {
		go func() {
			collector.collect()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// Wait for periodic collection
	time.Sleep(20 * time.Millisecond)

	// Verify final counts
	projectsTotal := getGaugeValue(t, m.ProjectsTotal)
	boardsTotal := getGaugeValue(t, m.BoardsTotal)

	assert.Equal(t, float64(5), projectsTotal, "ProjectsTotal should be 5")
	assert.Equal(t, float64(10), boardsTotal, "BoardsTotal should be 10")

	// Test passes if no race conditions or panics occur
}
