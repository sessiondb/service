// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

//go:build pro

package api

import (
	"context"
	"log"

	"sessiondb/internal/premium/alert"
	"sessiondb/internal/premium/report"
	"sessiondb/internal/premium/repository"
	"sessiondb/internal/premium/service"
	"sessiondb/internal/premium/session"
	"sessiondb/internal/premium/utils"

	"github.com/gin-gonic/gin"
)

// RegisterPremiumRoutes explicitly imports and connects the premium functionality.
// Because of the `//go:build pro` tag, this file only compiles when the user runs:
// `go build -tags pro`
func RegisterPremiumRoutes(router *gin.RouterGroup, deps *PremiumDeps) {
	log.Println("[INFO] PRO Edition loaded. Registering premium routes...")

	if deps == nil {
		return
	}

	// Initialize the isolated Pro modules
	rulesService := service.NewEnhancedRuleEnforcement()
	credsService := service.NewAutoCredsExpiry()
	insightsRepo := repository.NewQueryInsights()
	metricsRepo := repository.NewDBMetrics()
	altersRepo := repository.NewDBAlters()
	ttlUtils := utils.NewTTLBasedAccess()

	_ = rulesService
	_ = credsService
	_ = insightsRepo
	_ = metricsRepo
	_ = altersRepo
	_ = ttlUtils

	// Phase 4: Session Engine
	credSessionRepo := repository.NewCredentialSessionRepository(deps.DB)
	sessionEngine := session.NewEngine(credSessionRepo, deps.InstanceRepo, deps.AccessEngine)
	sessionHandler := session.NewHandler(sessionEngine)
	sessions := router.Group("/sessions")
	{
		sessions.POST("/start", sessionHandler.Start)
		sessions.POST("/:id/end", sessionHandler.End)
		sessions.GET("/active", sessionHandler.GetActive)
	}

	// Phase 5: Alert Engine
	alertRuleRepo := repository.NewAlertRuleRepository(deps.DB)
	alertEventRepo := repository.NewAlertEventRepository(deps.DB)
	alertEngine := alert.NewEngine(alertRuleRepo, alertEventRepo)
	alertHandler := alert.NewHandler(alertEngine)
	if deps.QueryService != nil {
		deps.QueryService.SetOnQueryExecuted(func(ctx context.Context, eventSource string, payload map[string]interface{}) {
			_ = alertEngine.EvaluateRules(ctx, eventSource, payload)
		})
	}
	alerts := router.Group("/alerts")
	{
		alerts.POST("/rules", alertHandler.CreateRule)
		alerts.GET("/rules", alertHandler.ListRules)
		alerts.GET("/rules/:id", alertHandler.GetRule)
		alerts.PUT("/rules/:id", alertHandler.UpdateRule)
		alerts.DELETE("/rules/:id", alertHandler.DeleteRule)
		alerts.GET("/events", alertHandler.ListEvents)
	}

	// Phase 6: Report Engine
	reportDefRepo := repository.NewReportDefinitionRepository(deps.DB)
	reportExecRepo := repository.NewReportExecutionRepository(deps.DB)
	reportEngine := report.NewEngine(reportDefRepo, reportExecRepo)
	reportHandler := report.NewHandler(reportEngine)
	reports := router.Group("/reports")
	{
		reports.POST("/definitions", reportHandler.CreateDefinition)
		reports.GET("/definitions", reportHandler.ListDefinitions)
		reports.GET("/definitions/:id", reportHandler.GetDefinition)
		reports.PUT("/definitions/:id", reportHandler.UpdateDefinition)
		reports.DELETE("/definitions/:id", reportHandler.DeleteDefinition)
		reports.POST("/definitions/:id/run", reportHandler.RunReport)
		reports.GET("/definitions/:id/executions", reportHandler.ListExecutions)
	}
}
