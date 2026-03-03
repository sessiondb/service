// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package api

import "github.com/gin-gonic/gin"

// PremiumService defines the contract that the Pro version will honor
// if it is successfully compiled into the binary.
type PremiumService interface {
	RegisterRoutes(router *gin.RouterGroup)
}
