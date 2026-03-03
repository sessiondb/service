// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

//go:build !pro

package api

import (
	"log"

	"github.com/gin-gonic/gin"
)

// RegisterPremiumRoutes acts as a no-op safety net for the Community Edition.
// Since the 'pro' build tag is absent, this file is loaded instead of the pro provider.
func RegisterPremiumRoutes(router *gin.RouterGroup) {
	log.Println("[INFO] Community Edition loaded. Premium routes are disabled.")
	// Do nothing. This ensures the Go program cleanly compiles.
}
