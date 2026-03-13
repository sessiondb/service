// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package handlers

import (
	"log"
	"net/http"
	"sessiondb/internal/repository"

	"github.com/gin-gonic/gin"
)

// RunMigrate runs database migrations and returns 200 on success.
// Must be protected by MigrateTokenAuth middleware (caller sets MIGRATE_TOKEN env).
func RunMigrate(c *gin.Context) {
	if repository.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not connected"})
		return
	}
	repository.Migrate()
	log.Println("Migration completed via API")
	c.JSON(http.StatusOK, gin.H{"status": "migrated"})
}
