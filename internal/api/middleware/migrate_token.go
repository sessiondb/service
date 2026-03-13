// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// MigrateTokenAuth returns a middleware that allows the request only if X-Migrate-Token
// or Authorization: Bearer <token> matches the MIGRATE_TOKEN env var.
// If MIGRATE_TOKEN is unset, the migrate endpoint is disabled (503).
func MigrateTokenAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		expected := os.Getenv("MIGRATE_TOKEN")
		if expected == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "migrate endpoint disabled (MIGRATE_TOKEN not set)"})
			c.Abort()
			return
		}
		token := strings.TrimSpace(c.GetHeader("X-Migrate-Token"))
		if token == "" {
			auth := c.GetHeader("Authorization")
			if strings.HasPrefix(strings.TrimSpace(auth), "Bearer ") {
				token = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(auth), "Bearer "))
			}
		}
		if token != expected {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid migrate token"})
			c.Abort()
			return
		}
		c.Next()
	}
}
