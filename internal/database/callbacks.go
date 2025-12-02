package database

import (
	"time"

	"gorm.io/gorm"
)

// MetricsRecorder is an interface for recording database metrics
type MetricsRecorder interface {
	RecordDBQuery(operation, table string, duration time.Duration, err error)
	UpdateDBStats(stats interface{})
}

// RegisterMetricsCallbacks registers GORM callbacks for metrics collection
func RegisterMetricsCallbacks(db *gorm.DB, recorder MetricsRecorder) {
	// Query callback
	db.Callback().Query().Before("gorm:query").Register("metrics:query_before", func(db *gorm.DB) {
		db.InstanceSet("query_start_time", time.Now())
	})

	db.Callback().Query().After("gorm:query").Register("metrics:query_after", func(db *gorm.DB) {
		if startTime, ok := db.InstanceGet("query_start_time"); ok {
			duration := time.Since(startTime.(time.Time))
			table := db.Statement.Table
			if table == "" {
				table = "unknown"
			}
			recorder.RecordDBQuery("select", table, duration, db.Error)
		}
	})

	// Create callback
	db.Callback().Create().Before("gorm:create").Register("metrics:create_before", func(db *gorm.DB) {
		db.InstanceSet("query_start_time", time.Now())
	})

	db.Callback().Create().After("gorm:create").Register("metrics:create_after", func(db *gorm.DB) {
		if startTime, ok := db.InstanceGet("query_start_time"); ok {
			duration := time.Since(startTime.(time.Time))
			table := db.Statement.Table
			if table == "" {
				table = "unknown"
			}
			recorder.RecordDBQuery("insert", table, duration, db.Error)
		}
	})

	// Update callback
	db.Callback().Update().Before("gorm:update").Register("metrics:update_before", func(db *gorm.DB) {
		db.InstanceSet("query_start_time", time.Now())
	})

	db.Callback().Update().After("gorm:update").Register("metrics:update_after", func(db *gorm.DB) {
		if startTime, ok := db.InstanceGet("query_start_time"); ok {
			duration := time.Since(startTime.(time.Time))
			table := db.Statement.Table
			if table == "" {
				table = "unknown"
			}
			recorder.RecordDBQuery("update", table, duration, db.Error)
		}
	})

	// Delete callback
	db.Callback().Delete().Before("gorm:delete").Register("metrics:delete_before", func(db *gorm.DB) {
		db.InstanceSet("query_start_time", time.Now())
	})

	db.Callback().Delete().After("gorm:delete").Register("metrics:delete_after", func(db *gorm.DB) {
		if startTime, ok := db.InstanceGet("query_start_time"); ok {
			duration := time.Since(startTime.(time.Time))
			table := db.Statement.Table
			if table == "" {
				table = "unknown"
			}
			recorder.RecordDBQuery("delete", table, duration, db.Error)
		}
	})
}

// StartDBStatsCollector starts periodic DB stats collection
func StartDBStatsCollector(db *gorm.DB, recorder MetricsRecorder) chan struct{} {
	done := make(chan struct{})

	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				sqlDB, err := db.DB()
				if err != nil {
					continue
				}
				stats := sqlDB.Stats()
				recorder.UpdateDBStats(stats)
			case <-done:
				return
			}
		}
	}()

	return done
}
