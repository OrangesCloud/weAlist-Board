package metrics

import (
	"database/sql"
	"strings"
	"time"
)

// UpdateDBStats updates database connection pool metrics
func (m *Metrics) UpdateDBStats(statsInterface interface{}) {
	m.safeExecute("UpdateDBStats", func() {
		stats, ok := statsInterface.(sql.DBStats)
		if !ok {
			return
		}
		m.DBConnectionsOpen.Set(float64(stats.OpenConnections))
		m.DBConnectionsInUse.Set(float64(stats.InUse))
		m.DBConnectionsIdle.Set(float64(stats.Idle))
		m.DBConnectionsMax.Set(float64(stats.MaxOpenConnections))
		m.DBConnectionWaitTotal.Add(float64(stats.WaitCount))
		m.DBConnectionWaitDuration.Add(stats.WaitDuration.Seconds())
	})
}

// RecordDBQuery records database query metrics
func (m *Metrics) RecordDBQuery(operation, table string, duration time.Duration, err error) {
	m.safeExecute("RecordDBQuery", func() {
		operation = normalizeOperation(operation)
		m.DBQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())

		if err != nil {
			m.DBQueryErrors.WithLabelValues(operation, table).Inc()
		}
	})
}

// normalizeOperation converts operation to lowercase
func normalizeOperation(op string) string {
	return strings.ToLower(op)
}
