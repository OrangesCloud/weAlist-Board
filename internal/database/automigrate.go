package database

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"project-board-api/internal/domain"
)

// modelInfo holds information about a domain model and its table name
type modelInfo struct {
	model     interface{}
	tableName string
}

// AutoMigrate runs GORM auto-migration for all domain models
// It automatically creates tables, indexes, and foreign key constraints
// based on the struct definitions in the domain package
func AutoMigrate(db *gorm.DB) error {
	// List of all domain models to migrate
	models := []interface{}{
		&domain.Project{},
		&domain.ProjectMember{},
		&domain.ProjectJoinRequest{},
		&domain.Board{},
		&domain.Participant{},
		&domain.Comment{},
		&domain.FieldOption{},
		&domain.Attachment{},
	}

	// Run auto-migration for all models
	if err := db.AutoMigrate(models...); err != nil {
		return fmt.Errorf("failed to run auto-migration: %w", err)
	}

	return nil
}

// SafeAutoMigrate runs GORM auto-migration safely by checking table existence first
// It handles both fresh installations and existing databases
// For existing tables, it only updates schema differences (adds columns, indexes)
// For new tables, it creates them from scratch
func SafeAutoMigrate(db *gorm.DB, logger *zap.Logger) error {
	migrator := db.Migrator()

	// List of all domain models with their table names
	models := []modelInfo{
		{&domain.Project{}, "projects"},
		{&domain.ProjectMember{}, "project_members"},
		{&domain.ProjectJoinRequest{}, "project_join_requests"},
		{&domain.Board{}, "boards"},
		{&domain.Participant{}, "participants"},
		{&domain.Comment{}, "comments"},
		{&domain.FieldOption{}, "field_options"},
		{&domain.Attachment{}, "attachments"},
	}

	logger.Info("Starting safe auto-migration",
		zap.Int("total_models", len(models)),
	)

	for _, m := range models {
		// Check if table exists
		tableExists := migrator.HasTable(m.model)

		if tableExists {
			logger.Info("Table exists, updating schema only",
				zap.String("table", m.tableName),
			)
		} else {
			logger.Info("Table does not exist, creating new table",
				zap.String("table", m.tableName),
			)
		}

		// Run auto-migration for this model
		// GORM will handle both creation and updates appropriately
		if err := db.AutoMigrate(m.model); err != nil {
			logger.Error("Failed to migrate table",
				zap.String("table", m.tableName),
				zap.Bool("table_existed", tableExists),
				zap.Error(err),
			)
			return fmt.Errorf("failed to migrate table %s: %w", m.tableName, err)
		}

		logger.Info("Successfully migrated table",
			zap.String("table", m.tableName),
			zap.Bool("was_existing", tableExists),
		)
	}

	logger.Info("Safe auto-migration completed successfully",
		zap.Int("tables_migrated", len(models)),
	)

	return nil
}

// SafeAutoMigrateWithRetry runs SafeAutoMigrate with retry logic
// It attempts migration up to maxRetries times with exponential backoff
func SafeAutoMigrateWithRetry(db *gorm.DB, logger *zap.Logger, maxRetries int) error {
	var err error

	logger.Info("Starting auto-migration with retry logic",
		zap.Int("max_retries", maxRetries),
	)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		logger.Info("Migration attempt",
			zap.Int("attempt", attempt),
			zap.Int("max_retries", maxRetries),
		)

		err = SafeAutoMigrate(db, logger)
		if err == nil {
			logger.Info("Migration completed successfully",
				zap.Int("attempt", attempt),
			)
			return nil
		}

		// Log the error and retry if not the last attempt
		if attempt < maxRetries {
			backoffDuration := time.Duration(attempt) * time.Second
			logger.Warn("Migration attempt failed, retrying...",
				zap.Int("attempt", attempt),
				zap.Int("max_retries", maxRetries),
				zap.Duration("backoff", backoffDuration),
				zap.Error(err),
			)
			time.Sleep(backoffDuration)
		} else {
			logger.Error("Migration failed after all retry attempts",
				zap.Int("total_attempts", maxRetries),
				zap.Error(err),
			)
		}
	}

	return fmt.Errorf("migration failed after %d attempts: %w", maxRetries, err)
}
