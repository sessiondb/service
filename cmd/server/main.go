// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package main

import (
	"log"
	"os"
	"strings"

	sessionapi "sessiondb/internal/api"
	"sessiondb/internal/api/handlers"
	"sessiondb/internal/api/middleware"
	"sessiondb/internal/config"
	community_access "sessiondb/internal/community/access"
	community_ai "sessiondb/internal/community/ai"
	"sessiondb/internal/repository"
	"sessiondb/internal/service"
	"sessiondb/internal/utils"

	"github.com/gin-gonic/gin"
)

// skipDefaultLogins returns true if env SKIP_DEFAULT_LOGINS is set to true/1/yes (case-insensitive).
func skipDefaultLogins() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("SKIP_DEFAULT_LOGINS")))
	return v == "true" || v == "1" || v == "yes"
}

func main() {
	// Load Config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Migrate-only mode: run migrations and exit (for Docker entrypoint or K8s Job).
	if len(os.Args) > 1 && strings.TrimSpace(strings.ToLower(os.Args[1])) == "migrate" {
		repository.ConnectDB(cfg)
		repository.Migrate()
		log.Println("Migration complete, exiting.")
		os.Exit(0)
	}

	// Connect to Database (migrations are run via POST /v1/migrate or Docker entrypoint, not on startup)
	repository.ConnectDB(cfg)

	// Initialize Repositories
	userRepo := repository.NewUserRepository(repository.DB)
	roleRepo := repository.NewRoleRepository(repository.DB)
	approvalRepo := repository.NewApprovalRepository(repository.DB)
	queryRepo := repository.NewQueryRepository(repository.DB)
	auditRepo := repository.NewAuditRepository(repository.DB)
	instanceRepo := repository.NewInstanceRepository(repository.DB)
	metaRepo := repository.NewMetadataRepository(repository.DB)
	dbUserCredRepo := repository.NewDBUserCredentialRepository(repository.DB)
	monitorRepo := repository.NewMonitoringRepository(repository.DB)
	permRepo := repository.NewPermissionRepository(repository.DB)
	aiConfigRepo := repository.NewAIConfigRepository(repository.DB)
	aiUsageRepo := repository.NewAIUsageRepository(repository.DB)
	featureNotifyRepo := repository.NewFeatureNotifyRepository(repository.DB)

	// Seed default platform logins from TOML once when no users exist (unless SKIP_DEFAULT_LOGINS is set)
	if !skipDefaultLogins() {
		if err := service.SeedDefaultLogins(cfg, userRepo, roleRepo); err != nil {
			log.Printf("Seed default logins: %v (continuing)", err)
		}
	}

	// Initialize new Mock Tenant Client early so it can be injected
	tenantClient := service.NewMockTenantClient()

	// Initialize Services
	auditService := service.NewAuditService(auditRepo)
	authService := service.NewAuthService(userRepo, cfg, tenantClient) // Inject AuditService into other services for automatic logging
	roleService := service.NewRoleService(roleRepo, userRepo)
	provisioningService := service.NewDBUserProvisioningService(dbUserCredRepo, instanceRepo)
	userService := service.NewUserService(userRepo, provisioningService, instanceRepo)
	configService := service.NewConfigService()
	approvalService := service.NewApprovalService(approvalRepo)
	queryService := service.NewQueryService(queryRepo, cfg)
	queryService.SetInstanceRepo(instanceRepo)     // Inject instance repo for query execution
	queryService.SetAuditService(auditService)     // Inject audit service for history logging
	queryService.SetDBUserCredRepo(dbUserCredRepo) // Inject DB user cred repo for user-level auth
	accessEngine := community_access.NewEngine(permRepo)
	queryService.SetAccessEngine(accessEngine)
	aiEngine := community_ai.NewEngine(aiConfigRepo, accessEngine, metaRepo, userRepo)
	instanceService := service.NewInstanceService(instanceRepo)
	syncService := service.NewSyncService(instanceRepo, metaRepo)
	metaService := service.NewMetadataService(metaRepo)
	hub := service.NewNotificationHub()
	syncWorker := service.NewSyncWorker(syncService, hub)
	monitoringService := service.NewMonitoringService(instanceRepo, monitorRepo, hub)
	monitoringWorker := service.NewMonitoringWorker(monitoringService)

	// Start Background Workers
	go hub.Run()
	syncWorker.Start()
	monitoringWorker.Start()

	// Initialize Handlers
	authHandler := handlers.NewAuthHandler(authService)
	roleHandler := handlers.NewRoleHandler(roleService)
	mailService := service.NewMailService(cfg)
	userHandler := handlers.NewUserHandler(userService, roleService, mailService)
	approvalHandler := handlers.NewApprovalHandler(approvalService)
	queryHandler := handlers.NewQueryHandler(queryService)
	auditHandler := handlers.NewAuditHandler(auditService, userRepo)
	configHandler := handlers.NewConfigHandler(configService)
	instanceHandler := handlers.NewInstanceHandler(instanceService, syncService)
	metaHandler := handlers.NewMetadataHandler(metaService, metaRepo, dbUserCredRepo)
	dbUserHandler := handlers.NewDBUserHandler(provisioningService, metaRepo, instanceRepo)
	dbRoleHandler := handlers.NewDBRoleHandler(metaRepo, instanceRepo)
	aiHandler := handlers.NewAIHandler(aiEngine, aiConfigRepo, aiUsageRepo)
	featureNotifyService := service.NewFeatureNotifyService(featureNotifyRepo)
	featureNotifyHandler := handlers.NewFeatureNotifyHandler(featureNotifyService)

	// Setup Router
	r := gin.Default()

	// CORS Middleware (Important for frontend dev)
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Migrate-Token")
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

		// Migrate: run DB migrations via API (protected by MIGRATE_TOKEN). Used by Docker/K8s to run migrations once.
		api.POST("/migrate", middleware.MigrateTokenAuth(), handlers.RunMigrate)

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
				users.GET("", middleware.CheckPermission(utils.PermUsersRead), userHandler.GetAllUsers)
				users.POST("", middleware.CheckPermission(utils.PermUsersWrite), userHandler.CreateUser)
				users.GET("/me", userHandler.GetMe) // Must be before /:id to avoid conflict
				users.GET("/:id", middleware.CheckPermission(utils.PermUsersRead), userHandler.GetUser)
				users.PUT("/:id", middleware.CheckPermission(utils.PermUsersWrite), userHandler.UpdateUser)
				users.DELETE("/:id", middleware.CheckPermission(utils.PermUsersWrite), userHandler.DeleteUser)
			}

			// DB User Management
			dbUsers := protected.Group("/db-users")
			{
				dbUsers.Use(middleware.CheckPermission(utils.PermUsersRead))
				dbUsers.GET("", dbUserHandler.GetDBUsers)
				dbUsers.PUT("/:id", middleware.CheckPermission(utils.PermUsersWrite), dbUserHandler.UpdateDBUserRole) // managed cred ID
				dbUsers.PUT("/:id/link", middleware.CheckPermission(utils.PermUsersWrite), dbUserHandler.LinkDBUser)  // db entity ID
			}

			// DB Credentials Management
			dbCredentials := protected.Group("/db-credentials")
			{
				dbCredentials.POST("/verify", dbUserHandler.VerifyCredentials)
			}

			// DB Role Management
			protected.GET("/db-roles", middleware.CheckPermission(utils.PermRolesManage), dbRoleHandler.GetDBRoles)

			// AI (BYOK)
			ai := protected.Group("/ai")
			{
				ai.POST("/generate-sql", aiHandler.GenerateSQL)
				ai.POST("/explain", aiHandler.ExplainQuery)
				ai.GET("/config", aiHandler.GetAIConfig)
				ai.PUT("/config", aiHandler.UpdateAIConfig)
				ai.GET("/usage", aiHandler.GetAIUsage)
			}

			// "Notify me when this is ready" for roadmap features (waitlist)
			protected.POST("/notify-me", featureNotifyHandler.Register)

			requests := protected.Group("/requests")
			{
				requests.Use(middleware.FeatureGate("advanced_approvals"))
				requests.Use(middleware.CheckPermission(utils.PermApprovalsManage))

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

			// Audit logs: community (permission-only). Export is premium.
			logs := protected.Group("/logs")
			logs.Use(middleware.CheckPermission(utils.PermLogsView))
			{
				logs.GET("", auditHandler.GetLogs)
				logs.POST("", auditHandler.CreateLog)
				logs.GET("/export", middleware.FeatureGate("audit_logs_export"), auditHandler.ExportLogs)
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
				instances.Use(middleware.CheckPermission(utils.PermInstancesRead))
				instances.GET("", instanceHandler.ListInstances)
				instances.GET("/:id/monitoring", instanceHandler.GetMonitoringLogs)
				instances.GET("/:id/databases", metaHandler.ListDatabases)
				instances.GET("/:id/databases/:dbName/tables", metaHandler.ListTables)
				instances.GET("/:id/schema", metaHandler.GetInstanceSchema)
				// Global table detail endpoint as per plan
				instances.GET("/tables/:tableId", metaHandler.GetTableDetails)
			}

			admin := protected.Group("/admin")
			{
				admin.Use(middleware.CheckPermission(utils.PermInstancesManage))
				admin.PUT("/ai-config", aiHandler.UpdateGlobalAIConfig)
				admin.GET("/ai/usage", aiHandler.GetAdminAIUsage)
				insts := admin.Group("/instances")
				{
					insts.GET("", instanceHandler.AdminListInstances)
					insts.POST("", instanceHandler.CreateInstance)
					insts.PUT("/:id", instanceHandler.UpdateInstance)
					insts.POST("/sync/:id", instanceHandler.SyncInstance)
				}
			}
		}

		// Build-Tag Plugin Pattern: Register premium routes dynamically.
		// If built with `-tags pro`, this points to `internal/api/provider_pro.go`.
		// If built normally (Community), this points to `internal/api/provider_community.go`.
		premiumDeps := &sessionapi.PremiumDeps{
			PermRepo:     permRepo,
			InstanceRepo: instanceRepo,
			AccessEngine: accessEngine,
			DB:           repository.DB,
			QueryService: queryService,
		}
		sessionapi.RegisterPremiumRoutes(protected, premiumDeps)
	}

	log.Printf("Starting server on port %s", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
