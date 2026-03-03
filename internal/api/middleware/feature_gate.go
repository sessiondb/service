// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// FeatureGate is a middleware that checks if a premium open-core feature is enabled
// for the current tenant, reading from the JWT claims populated by AuthMiddleware.
func FeatureGate(featureName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		featuresInterface, exists := c.Get("tenant_features")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "Tenant features not found in token. Please log in again."})
			c.Abort()
			return
		}

		featuresMap, ok := featuresInterface.(map[string]interface{})
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid tenant features format in token"})
			c.Abort()
			return
		}

		featureData, featureExists := featuresMap[featureName]
		if !featureExists {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Upgrade required to access this feature.",
				"code":  "plan_upgrade_required",
			})
			c.Abort()
			return
		}

		featureMapAsserted, mapOk := featureData.(map[string]interface{})
		if !mapOk {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid feature flag definition"})
			c.Abort()
			return
		}

		enabled, ok := featureMapAsserted["enabled"].(bool)

		if !ok || !enabled {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Upgrade required to access this feature.",
				"code":  "plan_upgrade_required",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
