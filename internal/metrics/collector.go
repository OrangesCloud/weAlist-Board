package metrics

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// BusinessMetricsCollector collects business metrics periodically
type BusinessMetricsCollector struct {
	db      *gorm.DB
	metrics *Metrics
	logger  *zap.Logger
	ticker  *time.Ticker
	done    chan bool
}

// NewBusinessMetricsCollector creates a new collector
func NewBusinessMetricsCollector(db *gorm.DB, metrics *Metrics, logger *zap.Logger) *BusinessMetricsCollector {
	return &BusinessMetricsCollector{
		db:      db,
		metrics: metrics,
		logger:  logger,
		ticker:  time.NewTicker(60 * time.Second),
		done:    make(chan bool),
	}
}

// Start begins collecting metrics
func (c *BusinessMetricsCollector) Start() {
	go func() {
		// 즉시 한 번 수집
		c.collect()

		// 주기적 수집
		for {
			select {
			case <-c.ticker.C:
				c.collect()
			case <-c.done:
				return
			}
		}
	}()
}

// Stop stops the collector
func (c *BusinessMetricsCollector) Stop() {
	c.ticker.Stop()
	c.done <- true
}

// collect gathers business metrics
func (c *BusinessMetricsCollector) collect() {
	defer func() {
		if r := recover(); r != nil {
			c.logger.Error("Panic in business metrics collection",
				zap.Any("panic", r),
			)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Count projects
	var projectCount int64
	if err := c.db.WithContext(ctx).Table("projects").Count(&projectCount).Error; err != nil {
		c.logger.Error("Failed to count projects", zap.Error(err))
	} else {
		c.metrics.SetProjectsTotal(projectCount)
	}

	// Count boards
	var boardCount int64
	if err := c.db.WithContext(ctx).Table("boards").Count(&boardCount).Error; err != nil {
		c.logger.Error("Failed to count boards", zap.Error(err))
	} else {
		c.metrics.SetBoardsTotal(boardCount)
	}
}
