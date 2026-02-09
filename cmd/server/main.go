package main

import (
	"log"
	"sessiondb/internal/api/handlers"
	"sessiondb/internal/api/middleware"
	"sessiondb/internal/config"
	"sessiondb/internal/repository"
	"sessiondb/internal/service"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load Config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to Database
	repository.ConnectDB(cfg)
	// Run Migrations
	repository.Migrate()

	// Initialize Repositories
	userRepo := repository.NewUserRepository(repository.DB)
	roleRepo := repository.NewRoleRepository(repository.DB)
	approvalRepo := repository.NewApprovalRepository(repository.DB)
	queryRepo := repository.NewQueryRepository(repository.DB)
	auditRepo := repository.NewAuditRepository(repository.DB)
	instanceRepo := repository.NewInstanceRepository(repository.DB)
	metaRepo := repository.NewMetadataRepository(repository.DB)

	// Initialize Services
	auditService := service.NewAuditService(auditRepo)
	authService := service.NewAuthService(userRepo, cfg) // TODO: Inject AuditService into other services for automatic logging
	roleService := service.NewRoleService(roleRepo, userRepo)
	userService := service.NewUserService(userRepo)
	configService := service.NewConfigService()
	approvalService := service.NewApprovalService(approvalRepo)
	queryService := service.NewQueryService(queryRepo, cfg)
	queryService.SetInstanceRepo(instanceRepo) // Inject instance repo for query execution
	instanceService := service.NewInstanceService(instanceRepo)
	syncService := service.NewSyncService(instanceRepo, metaRepo)
	metaService := service.NewMetadataService(metaRepo)
	hub := service.NewNotificationHub()
	syncWorker := service.NewSyncWorker(syncService, hub)

	// Start Background Workers
	go hub.Run()
	syncWorker.Start()

	// Initialize Handlers
	authHandler := handlers.NewAuthHandler(authService)
	roleHandler := handlers.NewRoleHandler(roleService)
	userHandler := handlers.NewUserHandler(userService, roleService)
	approvalHandler := handlers.NewApprovalHandler(approvalService)
	queryHandler := handlers.NewQueryHandler(queryService)
	auditHandler := handlers.NewAuditHandler(auditService, userRepo)
	configHandler := handlers.NewConfigHandler(configService)
	instanceHandler := handlers.NewInstanceHandler(instanceService, syncService)
	metaHandler := handlers.NewMetadataHandler(metaService)

	// Setup Router
	r := gin.Default()
	
	// CORS Middleware (Important for frontend dev)
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "up"})
	})

	r.GET("/ws", func(c *gin.Context) {
		hub.HandleWebSocket(c.Writer, c.Request)
	})

	api := r.Group("/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/register", authHandler.Register)
			auth.POST("/sso", authHandler.SSO)
			auth.POST("/logout", authHandler.Logout)
		}

		confs := api.Group("/config")
		{
			confs.GET("/auth", configHandler.GetAuthConfig)
			confs.PUT("/auth", configHandler.UpdateAuthConfig)
		}

		// Protected Routes
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware(cfg))
		{
			protected.GET("/schema", metaHandler.GetInstanceSchema)
			roles := protected.Group("/roles")
			{
				roles.POST("", roleHandler.CreateRole)
				roles.GET("", roleHandler.GetAllRoles)
				roles.GET("/:id", roleHandler.GetRole)
				roles.PUT("/:id", roleHandler.UpdateRole)
				roles.DELETE("/:id", roleHandler.DeleteRole)
			}

			users := protected.Group("/users")
			{
				users.GET("", userHandler.GetAllUsers)
				users.POST("", userHandler.CreateUser)
				users.GET("/:id", userHandler.GetUser)
				users.PUT("/:id", userHandler.UpdateUser)
				users.DELETE("/:id", userHandler.DeleteUser)
			}

			requests := protected.Group("/requests")
			{
				requests.GET("", approvalHandler.GetRequests)
				// Create request also mapped here? Docs say POST /approvals (now requests)
				requests.POST("", approvalHandler.CreateRequest)
				requests.PUT("/:id", approvalHandler.UpdateRequestStatus)
			}

			// Keep /approvals for backward compatibility if needed, or remove. 
			// Removing as per "Refactor".

			query := protected.Group("/query")
			{
				query.POST("/execute", queryHandler.ExecuteQuery)
				query.GET("/history", queryHandler.GetHistory)
				query.POST("/scripts", queryHandler.SaveScript)
				query.GET("/scripts", queryHandler.GetScripts)
			}

			logs := protected.Group("/logs")
			{
				logs.GET("", auditHandler.GetLogs)
				logs.POST("", auditHandler.CreateLog)
			}
			
			// User Context (Persisted State)
			me := protected.Group("/me")
			{
				me.POST("/scripts", queryHandler.SaveScript)
				me.GET("/scripts", queryHandler.GetScripts)
				me.GET("/tabs", userHandler.GetAllUsers) // Placeholder - user repository already preloads tabs so this works for now if we returns the user
				me.PUT("/tabs", userHandler.SyncTabs)
			}

			instances := protected.Group("/instances")
			{
				instances.GET("", instanceHandler.ListInstances)
				instances.GET("/:id/databases", metaHandler.ListDatabases)
				instances.GET("/:id/databases/:dbName/tables", metaHandler.ListTables)
				instances.GET("/:id/schema", metaHandler.GetInstanceSchema)
				// Global table detail endpoint as per plan
				instances.GET("/tables/:tableId", metaHandler.GetTableDetails)
			}

			admin := protected.Group("/admin")
			{
				insts := admin.Group("/instances")
				{
					insts.GET("", instanceHandler.AdminListInstances)
					insts.POST("", instanceHandler.CreateInstance)
					insts.PUT("/:id", instanceHandler.UpdateInstance)
					insts.POST("/sync/:id", instanceHandler.SyncInstance)
				}
			}
		}
	}

	log.Printf("Starting server on port %s", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
