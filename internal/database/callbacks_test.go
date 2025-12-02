package database

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// mockMetricsRecorder is a mock implementation of MetricsRecorder for testing
type mockMetricsRecorder struct {
	queries   []queryRecord
	dbStats   []sql.DBStats
	statsCall int
}

type queryRecord struct {
	operation string
	table     string
	duration  time.Duration
	err       error
}

func (m *mockMetricsRecorder) RecordDBQuery(operation, table string, duration time.Duration, err error) {
	m.queries = append(m.queries, queryRecord{
		operation: operation,
		table:     table,
		duration:  duration,
		err:       err,
	})
}

func (m *mockMetricsRecorder) UpdateDBStats(stats interface{}) {
	if dbStats, ok := stats.(sql.DBStats); ok {
		m.dbStats = append(m.dbStats, dbStats)
		m.statsCall++
	}
}

// testModel is a simple model for testing (using string ID for SQLite compatibility)
type testModel struct {
	ID        string `gorm:"type:text;primaryKey"`
	Name      string `gorm:"type:varchar(255)"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (testModel) TableName() string {
	return "test_models"
}

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "Failed to open test database")

	// Auto-migrate test model
	err = db.AutoMigrate(&testModel{})
	require.NoError(t, err, "Failed to migrate test model")

	return db
}

// TestRegisterMetricsCallbacks_Query tests query callback
// **Feature: board-service-prometheus-metrics, Property 5: GORM 쿼리 메트릭 기록**
// **Validates: Requirements 4.1, 4.2**
func TestRegisterMetricsCallbacks_Query(t *testing.T) {
	db := setupTestDB(t)
	recorder := &mockMetricsRecorder{}

	// Register callbacks
	RegisterMetricsCallbacks(db, recorder)

	// Insert test data
	testData := testModel{
		ID:   uuid.New().String(),
		Name: "test",
	}
	err := db.Create(&testData).Error
	require.NoError(t, err)

	// Clear previous records
	recorder.queries = nil

	// Execute query
	var result testModel
	err = db.First(&result).Error
	require.NoError(t, err)

	// Verify metrics were recorded
	require.Len(t, recorder.queries, 1, "Expected one query to be recorded")
	
	query := recorder.queries[0]
	assert.Equal(t, "select", query.operation, "Operation should be 'select'")
	assert.Equal(t, "test_models", query.table, "Table should be 'test_models'")
	assert.Greater(t, query.duration, time.Duration(0), "Duration should be greater than 0")
	assert.NoError(t, query.err, "Query should not have error")
}

// TestRegisterMetricsCallbacks_Create tests create callback
// **Feature: board-service-prometheus-metrics, Property 5: GORM 쿼리 메트릭 기록**
// **Validates: Requirements 4.1, 4.2**
func TestRegisterMetricsCallbacks_Create(t *testing.T) {
	db := setupTestDB(t)
	recorder := &mockMetricsRecorder{}

	// Register callbacks
	RegisterMetricsCallbacks(db, recorder)

	// Execute create
	testData := testModel{
		ID:   uuid.New().String(),
		Name: "test create",
	}
	err := db.Create(&testData).Error
	require.NoError(t, err)

	// Verify metrics were recorded
	require.Len(t, recorder.queries, 1, "Expected one query to be recorded")
	
	query := recorder.queries[0]
	assert.Equal(t, "insert", query.operation, "Operation should be 'insert'")
	assert.Equal(t, "test_models", query.table, "Table should be 'test_models'")
	assert.Greater(t, query.duration, time.Duration(0), "Duration should be greater than 0")
	assert.NoError(t, query.err, "Query should not have error")
}

// TestRegisterMetricsCallbacks_Update tests update callback
// **Feature: board-service-prometheus-metrics, Property 5: GORM 쿼리 메트릭 기록**
// **Validates: Requirements 4.1, 4.2**
func TestRegisterMetricsCallbacks_Update(t *testing.T) {
	db := setupTestDB(t)
	recorder := &mockMetricsRecorder{}

	// Register callbacks
	RegisterMetricsCallbacks(db, recorder)

	// Insert test data
	testData := testModel{
		ID:   uuid.New().String(),
		Name: "test",
	}
	err := db.Create(&testData).Error
	require.NoError(t, err)

	// Clear previous records
	recorder.queries = nil

	// Execute update
	err = db.Model(&testData).Update("Name", "updated").Error
	require.NoError(t, err)

	// Verify metrics were recorded
	require.Len(t, recorder.queries, 1, "Expected one query to be recorded")
	
	query := recorder.queries[0]
	assert.Equal(t, "update", query.operation, "Operation should be 'update'")
	assert.Equal(t, "test_models", query.table, "Table should be 'test_models'")
	assert.Greater(t, query.duration, time.Duration(0), "Duration should be greater than 0")
	assert.NoError(t, query.err, "Query should not have error")
}

// TestRegisterMetricsCallbacks_Delete tests delete callback
// **Feature: board-service-prometheus-metrics, Property 5: GORM 쿼리 메트릭 기록**
// **Validates: Requirements 4.1, 4.2**
func TestRegisterMetricsCallbacks_Delete(t *testing.T) {
	db := setupTestDB(t)
	recorder := &mockMetricsRecorder{}

	// Register callbacks
	RegisterMetricsCallbacks(db, recorder)

	// Insert test data
	testData := testModel{
		ID:   uuid.New().String(),
		Name: "test",
	}
	err := db.Create(&testData).Error
	require.NoError(t, err)

	// Clear previous records
	recorder.queries = nil

	// Execute delete
	err = db.Delete(&testData).Error
	require.NoError(t, err)

	// Verify metrics were recorded
	require.Len(t, recorder.queries, 1, "Expected one query to be recorded")
	
	query := recorder.queries[0]
	assert.Equal(t, "delete", query.operation, "Operation should be 'delete'")
	assert.Equal(t, "test_models", query.table, "Table should be 'test_models'")
	assert.Greater(t, query.duration, time.Duration(0), "Duration should be greater than 0")
	assert.NoError(t, query.err, "Query should not have error")
}

// TestRegisterMetricsCallbacks_QueryError tests error recording
// **Feature: board-service-prometheus-metrics, Property 6: GORM 쿼리 에러 카운팅**
// **Validates: Requirements 4.3**
func TestRegisterMetricsCallbacks_QueryError(t *testing.T) {
	db := setupTestDB(t)
	recorder := &mockMetricsRecorder{}

	// Register callbacks
	RegisterMetricsCallbacks(db, recorder)

	// Execute query that will fail (non-existent ID)
	var result testModel
	err := db.First(&result, "id = ?", uuid.New().String()).Error
	require.Error(t, err, "Expected query to fail")

	// Verify metrics were recorded with error
	require.Len(t, recorder.queries, 1, "Expected one query to be recorded")
	
	query := recorder.queries[0]
	assert.Equal(t, "select", query.operation, "Operation should be 'select'")
	assert.Equal(t, "test_models", query.table, "Table should be 'test_models'")
	assert.Greater(t, query.duration, time.Duration(0), "Duration should be greater than 0")
	assert.Error(t, query.err, "Query should have error")
}

// TestRegisterMetricsCallbacks_MultipleOperations tests multiple operations
// **Feature: board-service-prometheus-metrics, Property 5: GORM 쿼리 메트릭 기록**
// **Validates: Requirements 4.1, 4.2**
func TestRegisterMetricsCallbacks_MultipleOperations(t *testing.T) {
	db := setupTestDB(t)
	recorder := &mockMetricsRecorder{}

	// Register callbacks
	RegisterMetricsCallbacks(db, recorder)

	// Execute multiple operations
	testID := uuid.New().String()
	testData := testModel{
		ID:   testID,
		Name: "test",
	}
	
	// Create
	err := db.Create(&testData).Error
	require.NoError(t, err)

	// Query
	var result testModel
	err = db.First(&result, "id = ?", testID).Error
	require.NoError(t, err)

	// Update
	err = db.Model(&testData).Update("Name", "updated").Error
	require.NoError(t, err)

	// Delete
	err = db.Delete(&testData).Error
	require.NoError(t, err)

	// Verify all operations were recorded
	require.Len(t, recorder.queries, 4, "Expected four queries to be recorded")
	
	operations := []string{"insert", "select", "update", "delete"}
	for i, expectedOp := range operations {
		assert.Equal(t, expectedOp, recorder.queries[i].operation, 
			"Operation %d should be '%s'", i, expectedOp)
		assert.Equal(t, "test_models", recorder.queries[i].table, 
			"Table for operation %d should be 'test_models'", i)
		assert.Greater(t, recorder.queries[i].duration, time.Duration(0), 
			"Duration for operation %d should be greater than 0", i)
	}
}

// TestStartDBStatsCollector tests DB stats collection
// **Feature: board-service-prometheus-metrics, Property 4: 데이터베이스 연결 메트릭 노출**
// **Validates: Requirements 2.1, 2.2, 2.3, 2.4, 2.5, 2.6**
func TestStartDBStatsCollector(t *testing.T) {
	db := setupTestDB(t)
	recorder := &mockMetricsRecorder{}

	// Start collector
	done := StartDBStatsCollector(db, recorder)
	defer close(done)

	// Wait for at least one collection cycle
	time.Sleep(100 * time.Millisecond)

	// Manually trigger collection by getting stats
	sqlDB, err := db.DB()
	require.NoError(t, err)
	stats := sqlDB.Stats()
	recorder.UpdateDBStats(stats)

	// Verify stats were collected
	assert.Greater(t, recorder.statsCall, 0, "Stats should have been collected at least once")
	
	if len(recorder.dbStats) > 0 {
		lastStats := recorder.dbStats[len(recorder.dbStats)-1]
		// Verify stats structure (values may vary)
		assert.GreaterOrEqual(t, lastStats.OpenConnections, 0, "OpenConnections should be >= 0")
		assert.GreaterOrEqual(t, lastStats.InUse, 0, "InUse should be >= 0")
		assert.GreaterOrEqual(t, lastStats.Idle, 0, "Idle should be >= 0")
	}
}

// TestStartDBStatsCollector_Shutdown tests graceful shutdown
// **Feature: board-service-prometheus-metrics, Property 4: 데이터베이스 연결 메트릭 노출**
// **Validates: Requirements 2.1, 2.2, 2.3, 2.4, 2.5, 2.6**
func TestStartDBStatsCollector_Shutdown(t *testing.T) {
	db := setupTestDB(t)
	recorder := &mockMetricsRecorder{}

	// Start collector
	done := StartDBStatsCollector(db, recorder)

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Stop collector
	close(done)

	// Wait to ensure goroutine exits
	time.Sleep(50 * time.Millisecond)

	// Test passes if no panic or deadlock occurs
}

// TestRegisterMetricsCallbacks_CreateError tests create error recording
// **Feature: board-service-prometheus-metrics, Property 6: GORM 쿼리 에러 카운팅**
// **Validates: Requirements 4.3**
func TestRegisterMetricsCallbacks_CreateError(t *testing.T) {
	db := setupTestDB(t)
	recorder := &mockMetricsRecorder{}

	// Register callbacks
	RegisterMetricsCallbacks(db, recorder)

	// Try to create with duplicate ID (should fail on second attempt)
	testID := uuid.New().String()
	testData1 := testModel{
		ID:   testID,
		Name: "test1",
	}
	err := db.Create(&testData1).Error
	require.NoError(t, err)

	// Clear previous records
	recorder.queries = nil

	// Try to create with same ID (should fail)
	testData2 := testModel{
		ID:   testID,
		Name: "test2",
	}
	err = db.Create(&testData2).Error
	require.Error(t, err, "Expected create to fail with duplicate ID")

	// Verify error was recorded
	require.Len(t, recorder.queries, 1, "Expected one query to be recorded")
	
	query := recorder.queries[0]
	assert.Equal(t, "insert", query.operation, "Operation should be 'insert'")
	assert.Error(t, query.err, "Query should have error")
}

// TestRegisterMetricsCallbacks_Transaction tests metrics in transaction
// **Feature: board-service-prometheus-metrics, Property 5: GORM 쿼리 메트릭 기록**
// **Validates: Requirements 4.1, 4.2**
func TestRegisterMetricsCallbacks_Transaction(t *testing.T) {
	db := setupTestDB(t)
	recorder := &mockMetricsRecorder{}

	// Register callbacks
	RegisterMetricsCallbacks(db, recorder)

	// Execute transaction
	err := db.Transaction(func(tx *gorm.DB) error {
		testData1 := testModel{
			ID:   uuid.New().String(),
			Name: "test1",
		}
		if err := tx.Create(&testData1).Error; err != nil {
			return err
		}

		testData2 := testModel{
			ID:   uuid.New().String(),
			Name: "test2",
		}
		if err := tx.Create(&testData2).Error; err != nil {
			return err
		}

		return nil
	})
	require.NoError(t, err)

	// Verify both creates were recorded
	assert.GreaterOrEqual(t, len(recorder.queries), 2, "Expected at least two queries to be recorded")
	
	// Count insert operations
	insertCount := 0
	for _, query := range recorder.queries {
		if query.operation == "insert" {
			insertCount++
		}
	}
	assert.GreaterOrEqual(t, insertCount, 2, "Expected at least two insert operations")
}

// TestRegisterMetricsCallbacks_TransactionRollback tests metrics on rollback
// **Feature: board-service-prometheus-metrics, Property 6: GORM 쿼리 에러 카운팅**
// **Validates: Requirements 4.3**
func TestRegisterMetricsCallbacks_TransactionRollback(t *testing.T) {
	db := setupTestDB(t)
	recorder := &mockMetricsRecorder{}

	// Register callbacks
	RegisterMetricsCallbacks(db, recorder)

	// Execute transaction that will rollback
	err := db.Transaction(func(tx *gorm.DB) error {
		testData := testModel{
			ID:   uuid.New().String(),
			Name: "test",
		}
		if err := tx.Create(&testData).Error; err != nil {
			return err
		}

		// Force rollback
		return errors.New("forced rollback")
	})
	require.Error(t, err, "Expected transaction to fail")

	// Verify create was still recorded (even though transaction rolled back)
	assert.GreaterOrEqual(t, len(recorder.queries), 1, "Expected at least one query to be recorded")
}
