// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

//go:build pro

package api

import (
	"log"

	"sessiondb/internal/premium/repository"
	"sessiondb/internal/premium/service"
	"sessiondb/internal/premium/utils"

	"github.com/gin-gonic/gin"
)

// RegisterPremiumRoutes explicitly imports and connects the premium functionality.
// Because of the `//go:build pro` tag, this file only compiles when the user runs:
// `go build -tags pro`
func RegisterPremiumRoutes(router *gin.RouterGroup) {
	log.Println("[INFO] PRO Edition loaded. Registering premium routes...")

	// Initialize the isolated Pro modules
	rulesService := service.NewEnhancedRuleEnforcement()
	credsService := service.NewAutoCredsExpiry()
	insightsRepo := repository.NewQueryInsights()
	metricsRepo := repository.NewDBMetrics()
	altersRepo := repository.NewDBAlters()
	ttlUtils := utils.NewTTLBasedAccess()

	// Suppress unused variable panics for now while they are just stubs
	_ = rulesService
	_ = credsService
	_ = insightsRepo
	_ = metricsRepo
	_ = altersRepo
	_ = ttlUtils

	// In the future, we will mount our pro routers here using the above services, e.g.:
	// router.GET("/metrics", middleware.RequireFeature("metrics"), premiumMetricsHandler.GetMetrics)
}
