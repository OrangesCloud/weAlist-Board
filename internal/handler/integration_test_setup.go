package handler

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"project-board-api/internal/client"
	"project-board-api/internal/config"
	"project-board-api/internal/converter"
	"project-board-api/internal/domain"
	"project-board-api/internal/metrics"
	"project-board-api/internal/repository"
	"project-board-api/internal/service"
)

func setupIntegrationTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "Failed to connect to test database")

	// Register callback to generate UUIDs for SQLite (since it doesn't support gen_random_uuid())
	db.Callback().Create().Before("gorm:create").Register("generate_uuid", func(db *gorm.DB) {
		if db.Statement.Schema != nil {
			for _, field := range db.Statement.Schema.PrimaryFields {
				if field.DataType == "uuid" {
					fieldValue := field.ReflectValueOf(db.Statement.Context, db.Statement.ReflectValue)
					if fieldValue.IsZero() {
						field.Set(db.Statement.Context, db.Statement.ReflectValue, uuid.New())
					}
				}
			}
		}
	})

	// Create tables manually for SQLite compatibility
	// SQLite doesn't support UUID type or gen_random_uuid()
	err = db.Exec(`
		CREATE TABLE projects (
			id TEXT PRIMARY KEY,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			deleted_at DATETIME,
			workspace_id TEXT NOT NULL,
			owner_id TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			start_date DATETIME,
			due_date DATETIME,
			is_default INTEGER DEFAULT 0,
			is_public INTEGER DEFAULT 0
		)
	`).Error
	require.NoError(t, err, "Failed to create projects table")

	err = db.Exec(`
		CREATE TABLE boards (
			id TEXT PRIMARY KEY,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			deleted_at DATETIME,
			project_id TEXT NOT NULL,
			author_id TEXT NOT NULL,
			assignee_id TEXT,
			title TEXT NOT NULL,
			content TEXT,
			custom_fields TEXT,
			start_date DATETIME,
			due_date DATETIME
		)
	`).Error
	require.NoError(t, err, "Failed to create boards table")

	err = db.Exec(`
		CREATE TABLE participants (
			id TEXT PRIMARY KEY,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			deleted_at DATETIME,
			board_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			UNIQUE(board_id, user_id)
		)
	`).Error
	require.NoError(t, err, "Failed to create participants table")

	err = db.Exec(`
		CREATE TABLE comments (
			id TEXT PRIMARY KEY,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			deleted_at DATETIME,
			board_id TEXT NOT NULL,
			author_id TEXT NOT NULL,
			content TEXT NOT NULL
		)
	`).Error
	require.NoError(t, err, "Failed to create comments table")

	err = db.Exec(`
		CREATE TABLE field_options (
			id TEXT PRIMARY KEY,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			deleted_at DATETIME,
			project_id TEXT NOT NULL,
			field_name TEXT NOT NULL,
			option_value TEXT NOT NULL,
			option_label TEXT NOT NULL,
			color TEXT,
			display_order INTEGER DEFAULT 0,
			UNIQUE(project_id, field_name, option_value)
		)
	`).Error
	require.NoError(t, err, "Failed to create field_options table")

	err = db.Exec(`
		CREATE TABLE attachments (
			id TEXT PRIMARY KEY,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			deleted_at DATETIME,
			entity_type TEXT NOT NULL,
			entity_id TEXT,
			status TEXT NOT NULL DEFAULT 'TEMP',
			file_name TEXT NOT NULL,
			file_url TEXT NOT NULL,
			file_size INTEGER NOT NULL,
			content_type TEXT NOT NULL,
			uploaded_by TEXT NOT NULL,
			expires_at DATETIME
		)
	`).Error
	require.NoError(t, err, "Failed to create attachments table")

	return db
}

// setupIntegrationRouter creates a router with real services and repositories
func setupIntegrationRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add test middleware to set user_id from header
	router.Use(func(c *gin.Context) {
		if userIDStr := c.GetHeader("X-User-ID"); userIDStr != "" {
			if userID, err := uuid.Parse(userIDStr); err == nil {
				c.Set("user_id", userID)
			}
		}
		c.Next()
	})

	// Initialize repositories
	projectRepo := repository.NewProjectRepository(db)
	boardRepo := repository.NewBoardRepository(db)
	participantRepo := repository.NewParticipantRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	fieldOptionRepo := repository.NewFieldOptionRepository(db)

	// Initialize converters
	fieldOptionConverter := converter.NewFieldOptionConverter(fieldOptionRepo)

	// Initialize services
	participantService := service.NewParticipantService(participantRepo, boardRepo)
	// Create a no-op logger for tests
	logger, _ := zap.NewDevelopment()
	attachmentRepo := repository.NewAttachmentRepository(db)
	
	// Create S3 client for tests
	cfg := &config.S3Config{
		Bucket:    "test-bucket",
		Region:    "us-east-1",
		AccessKey: "test-key",
		SecretKey: "test-secret",
	}
	s3Client, _ := client.NewS3Client(cfg)
	m := metrics.NewTestMetrics()
	
	boardService := service.NewBoardService(boardRepo, projectRepo, fieldOptionRepo, participantRepo, attachmentRepo, s3Client, fieldOptionConverter, m, logger)

	commentService := service.NewCommentService(commentRepo, boardRepo, attachmentRepo, s3Client, logger)

	// Initialize handlers
	boardHandler := NewBoardHandler(boardService)
	participantHandler := NewParticipantHandler(participantService)
	commentHandler := NewCommentHandler(commentService)

	// Setup routes
	api := router.Group("/api")
	{
		// Board routes
		boards := api.Group("/boards")
		{
			boards.POST("", boardHandler.CreateBoard)
			boards.GET("/:boardId", boardHandler.GetBoard)
			boards.GET("/project/:projectId", boardHandler.GetBoardsByProject)
			boards.PUT("/:boardId", boardHandler.UpdateBoard)
			boards.DELETE("/:boardId", boardHandler.DeleteBoard)
		}

		// Participant routes
		participants := api.Group("/participants")
		{
			participants.POST("", participantHandler.AddParticipants)
			participants.GET("/board/:boardId", participantHandler.GetParticipants)
			participants.DELETE("/board/:boardId/user/:userId", participantHandler.RemoveParticipant)
		}

		// Comment routes
		comments := api.Group("/comments")
		{
			comments.POST("", commentHandler.CreateComment)
			comments.GET("/board/:boardId", commentHandler.GetComments)
			comments.PUT("/:commentId", commentHandler.UpdateComment)
			comments.DELETE("/:commentId", commentHandler.DeleteComment)
		}
	}

	return router
}

// createTestProject creates a test project in the database
func createTestProject(t *testing.T, db *gorm.DB) *domain.Project {
	project := &domain.Project{
		BaseModel: domain.BaseModel{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		WorkspaceID: uuid.New(),
		OwnerID:     uuid.New(),
		Name:        "Test Project",
		Description: "Test Description",
	}
	err := db.Create(project).Error
	require.NoError(t, err, "Failed to create test project")
	return project
}

// createTestBoard creates a test board in the database
func createTestBoard(t *testing.T, db *gorm.DB, projectID uuid.UUID) *domain.Board {
	authorID := uuid.New()
	board := &domain.Board{
		BaseModel: domain.BaseModel{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		ProjectID: projectID,
		AuthorID:  authorID,
		Title:     "Test Board",
		Content:   "Test Content",
	}
	err := db.Create(board).Error
	require.NoError(t, err, "Failed to create test board")
	return board
}

// TestIntegration_AddParticipants_API tests the participant addition API endpoint
// **Validates: Requirements 3.2, 3.4, 3.5**
