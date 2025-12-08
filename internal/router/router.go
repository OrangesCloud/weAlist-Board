// Package router provides HTTP routing configuration.
package router

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"project-board-api/internal/client"
	"project-board-api/internal/converter"
	"project-board-api/internal/database"
	"project-board-api/internal/handler"
	"project-board-api/internal/metrics"
	"project-board-api/internal/middleware"
	"project-board-api/internal/repository"
	"project-board-api/internal/service"
)

type Config struct {
	DB                 *gorm.DB
	Logger             *zap.Logger
	JWTSecret          string
	UserClient         client.UserClient
	BasePath           string
	UserServiceBaseURL string
	Metrics            *metrics.Metrics
	S3Client           *client.S3Client
}

// Setup initializes the router with all dependencies and routes.
func Setup(cfg Config) *gin.Engine {
	// Create Gin router
	router := gin.New()

	// Apply global middleware chain
	router.Use(
		middleware.Recovery(cfg.Logger), // 1. Panic recovery
		middleware.RequestID(),          // 2. Request ID tracking
		middleware.Logger(cfg.Logger),   // 3. Request logging
		middleware.CORS(),               // 4. CORS configuration
	)

	// Add metrics middleware if metrics is configured
	if cfg.Metrics != nil {
		router.Use(middleware.Metrics(cfg.Metrics))
		cfg.Logger.Info("Metrics middleware enabled")
	}

	// Initialize repositories
	projectRepo := repository.NewProjectRepository(cfg.DB)
	boardRepo := repository.NewBoardRepository(cfg.DB)
	participantRepo := repository.NewParticipantRepository(cfg.DB)
	commentRepo := repository.NewCommentRepository(cfg.DB)
	fieldOptionRepo := repository.NewFieldOptionRepository(cfg.DB)
	attachmentRepo := repository.NewAttachmentRepository(cfg.DB)

	// Initialize converters
	fieldOptionConverter := converter.NewFieldOptionConverter(fieldOptionRepo)

	// Initialize services with repository dependencies
	projectService := service.NewProjectService(projectRepo, fieldOptionRepo, attachmentRepo, cfg.S3Client, cfg.UserClient, cfg.Metrics, cfg.Logger)
	boardService := service.NewBoardService(boardRepo, projectRepo, fieldOptionRepo, participantRepo, attachmentRepo, cfg.S3Client, fieldOptionConverter, cfg.Metrics, cfg.Logger)
	participantService := service.NewParticipantService(participantRepo, boardRepo)
	commentService := service.NewCommentService(commentRepo, boardRepo, attachmentRepo, cfg.S3Client, cfg.Logger)
	fieldOptionService := service.NewFieldOptionService(fieldOptionRepo)
	projectMemberService := service.NewProjectMemberService(projectRepo, cfg.UserClient)
	projectJoinRequestService := service.NewProjectJoinRequestService(projectRepo, cfg.UserClient)

	// Initialize handlers with service dependencies
	projectHandler := handler.NewProjectHandler(projectService)
	boardHandler := handler.NewBoardHandler(boardService)
	participantHandler := handler.NewParticipantHandler(participantService)
	commentHandler := handler.NewCommentHandler(commentService)
	fieldOptionHandler := handler.NewFieldOptionHandler(fieldOptionService)
	projectMemberHandler := handler.NewProjectMemberHandler(projectMemberService)
	projectJoinRequestHandler := handler.NewProjectJoinRequestHandler(projectJoinRequestService)
	attachmentHandler := handler.NewAttachmentHandler(cfg.S3Client, attachmentRepo)

	// 💡 WebSocket Handler 초기화
	wsHandler := handler.NewWSHandler(cfg.Logger, cfg.UserClient)

	// Create base path group if configured
	var baseGroup *gin.RouterGroup
	if cfg.BasePath != "" {
		baseGroup = router.Group(cfg.BasePath)
		cfg.Logger.Info("Base path configured for ALB routing", zap.String("base_path", cfg.BasePath))
	} else {
		baseGroup = router.Group("")
		cfg.Logger.Info("No base path configured, using root path")
	}

	// Health check endpoints
	// Liveness probe - 앱 프로세스만 확인 (DB/Redis 무관)
	baseGroup.GET("/health/live", livenessHandler())
	// Readiness probe - DB, Redis 연결 확인
	baseGroup.GET("/health/ready", readinessHandler(cfg.DB))
	// 기존 /health 엔드포인트 유지 (하위 호환성)
	baseGroup.GET("/health", readinessHandler(cfg.DB))

	// Swagger documentation endpoint
	baseGroup.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Metrics endpoint (no authentication required)
	// Add metrics endpoint at root level for compatibility
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Also add metrics endpoint under base path if configured
	if cfg.BasePath != "" {
		baseGroup.GET("/metrics", gin.WrapH(promhttp.Handler()))
		cfg.Logger.Info("Metrics endpoint configured at both root and base path",
			zap.String("root_path", "/metrics"),
			zap.String("base_path", cfg.BasePath+"/metrics"))
	} else {
		cfg.Logger.Info("Metrics endpoint configured at root path", zap.String("path", "/metrics"))
	}

	// Setup API routes
	setupRoutes(baseGroup, cfg.JWTSecret, projectHandler, boardHandler, participantHandler, commentHandler, fieldOptionHandler, projectMemberHandler, projectJoinRequestHandler, attachmentHandler)

	// 🔥 [중요] WebSocket은 baseGroup을 사용하되 인증 미들웨어 없이 직접 등록
	// basePath가 /api/boards일 때: /api/boards/api/ws/project/:projectId
	wsGroup := baseGroup.Group("/api")
	wsGroup.GET("/ws/project/:projectId", wsHandler.HandleWebSocket)

	return router
}

// livenessHandler returns a handler for the liveness probe
// This only checks if the application process is running (no DB/Redis check)
// Used by Kubernetes to determine if the pod should be restarted
func livenessHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "alive",
		})
	}
}

// readinessHandler returns a handler for the readiness probe
// This checks database and Redis connections
// Used by Kubernetes to determine if the pod should receive traffic
func readinessHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		checks := gin.H{}
		isReady := true

		// Check database connection
		sqlDB, err := db.DB()
		if err != nil {
			c.JSON(503, gin.H{
				"status":   "not_ready",
				"database": "error",
				"error":    err.Error(),
			})
			return
		}

		if err := sqlDB.Ping(); err != nil {
			c.JSON(503, gin.H{
				"status":   "not_ready",
				"database": "disconnected",
				"error":    err.Error(),
			})
			return
		}
		checks["database"] = "connected"

		// Check Redis connection
		redisClient := database.GetRedis()
		if redisClient != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			if err := redisClient.Ping(ctx).Err(); err != nil {
				checks["redis"] = "disconnected"
				isReady = false
			} else {
				checks["redis"] = "connected"
			}
		} else {
			checks["redis"] = "not_configured"
		}

		if isReady {
			c.JSON(200, gin.H{
				"status": "ready",
				"checks": checks,
			})
		} else {
			c.JSON(503, gin.H{
				"status": "not_ready",
				"checks": checks,
			})
		}
	}
}

// setupRoutes configures all API routes
func setupRoutes(
	baseGroup *gin.RouterGroup,
	jwtSecret string,
	projectHandler *handler.ProjectHandler,
	boardHandler *handler.BoardHandler,
	participantHandler *handler.ParticipantHandler,
	commentHandler *handler.CommentHandler,
	fieldOptionHandler *handler.FieldOptionHandler,
	projectMemberHandler *handler.ProjectMemberHandler,
	projectJoinRequestHandler *handler.ProjectJoinRequestHandler,
	attachmentHandler *handler.AttachmentHandler,
) {
	// API group with authentication
	api := baseGroup.Group("/api")
	api.Use(middleware.Auth(jwtSecret))
	{
		// Project routes
		projects := api.Group("/projects")
		{
			// Frontend compatibility route (query parameter style)
			projects.GET("", projectHandler.GetProjectsByWorkspaceQuery)

			// Existing routes
			projects.POST("", projectHandler.CreateProject)
			projects.GET("/workspace/:workspaceId", projectHandler.GetProjectsByWorkspace)
			projects.GET("/workspace/:workspaceId/default", projectHandler.GetDefaultProject)

			// New project management extension routes
			projects.GET("/search", projectHandler.SearchProjects)
			projects.GET("/:projectId", projectHandler.GetProject)
			projects.PUT("/:projectId", projectHandler.UpdateProject)
			projects.DELETE("/:projectId", projectHandler.DeleteProject)
			projects.GET("/:projectId/init-settings", projectHandler.GetProjectInitSettings)

			// Project member routes
			projects.GET("/:projectId/members", projectMemberHandler.GetMembers)
			projects.DELETE("/:projectId/members/:memberId", projectMemberHandler.RemoveMember)
			projects.PUT("/:projectId/members/:memberId/role", projectMemberHandler.UpdateMemberRole)

			// Project join request routes
			projects.GET("/:projectId/join-requests", projectJoinRequestHandler.GetJoinRequests)

			// Attachment routes for projects
			projects.GET("/:projectId/attachments", attachmentHandler.GetProjectAttachments)
		}

		// Join request routes (not nested under project)
		joinRequests := api.Group("/join-requests")
		{
			joinRequests.POST("", projectJoinRequestHandler.CreateJoinRequest)
			joinRequests.PUT("/:joinRequestId", projectJoinRequestHandler.UpdateJoinRequest)
		}

		// Board routes
		boards := api.Group("/boards")
		{
			// Frontend compatibility route (query parameter style)
			boards.GET("", boardHandler.GetBoardsByProjectQuery)

			boards.POST("", boardHandler.CreateBoard)
			boards.GET("/:boardId", boardHandler.GetBoard)
			boards.GET("/project/:projectId", boardHandler.GetBoardsByProject)
			boards.PUT("/:boardId", boardHandler.UpdateBoard)
			boards.DELETE("/:boardId", boardHandler.DeleteBoard)
			boards.PUT("/:boardId/move", boardHandler.MoveBoard) // ✅ 이 라인 추가

			// Attachment routes for boards
			boards.GET("/:boardId/attachments", attachmentHandler.GetBoardAttachments)
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
			// Frontend compatibility route (query parameter style)
			comments.GET("", commentHandler.GetCommentsByQuery)

			comments.POST("", commentHandler.CreateComment)
			comments.GET("/board/:boardId", commentHandler.GetComments)
			comments.PUT("/:commentId", commentHandler.UpdateComment)
			comments.DELETE("/:commentId", commentHandler.DeleteComment)

			// Attachment routes for comments
			comments.GET("/:commentId/attachments", attachmentHandler.GetCommentAttachments)
		}

		// Field option routes
		fieldOptions := api.Group("/field-options")
		{
			fieldOptions.GET("", fieldOptionHandler.GetFieldOptions)
			fieldOptions.POST("", fieldOptionHandler.CreateFieldOption)
			fieldOptions.PATCH("/:optionId", fieldOptionHandler.UpdateFieldOption)
			fieldOptions.DELETE("/:optionId", fieldOptionHandler.DeleteFieldOption)
		}

		// Attachment routes (Presigned URL approach)
		attachments := api.Group("/attachments")
		{
			// Generate presigned URL for direct S3 upload
			attachments.POST("/presigned-url", attachmentHandler.GeneratePresignedURL)
			// Save attachment metadata after successful S3 upload
			attachments.POST("", attachmentHandler.SaveAttachmentMetadata)
			// Delete attachment
			attachments.DELETE("/:attachmentId", attachmentHandler.DeleteAttachment)
		}
	}
}
