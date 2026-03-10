// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package repository

import (
	"fmt"
	"log"
	"sessiondb/internal/config"
	"sessiondb/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDB(cfg *config.Config) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.Port,
		cfg.Database.SSLMode,
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Connected to database")
}

func Migrate() {
	if DB == nil {
		log.Fatal("Database not connected")
	}

	// Enable uuid-ossp extension
	DB.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";")
	DB.Exec("CREATE EXTENSION IF NOT EXISTS \"pgcrypto\";")

	// --- Run deduplication BEFORE AutoMigrate creates the unique index.
	// Keeps the oldest active row per (instance_id, name, type) and hard-deletes all other duplicates.
	// Scoped to non-soft-deleted rows only so cleared-instance rows don't interfere.
	dedupeSQL := `
		DELETE FROM db_entities
		WHERE deleted_at IS NULL
		AND id NOT IN (
			SELECT DISTINCT ON (instance_id, name, type) id
			FROM db_entities
			WHERE deleted_at IS NULL
			ORDER BY instance_id, name, type, created_at ASC
		)
	`
	if res := DB.Exec(dedupeSQL); res.Error != nil {
		log.Printf("Warning: deduplication of db_entities skipped (table may not exist yet): %v", res.Error)
	} else {
		log.Printf("db_entities deduplication: removed %d duplicate rows", res.RowsAffected)
	}

	// Pre-migration: Ensure roles has db_key if it exists, or just populate it if it already failed half-way
	DB.Exec(`ALTER TABLE roles ADD COLUMN IF NOT EXISTS db_key TEXT`)
	DB.Exec(`UPDATE roles SET db_key = LOWER(REPLACE(name, ' ', '_')) WHERE db_key IS NULL`)

	err := DB.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Permission{},
		&models.ApprovalRequest{},

		&models.AuditLog{},
		&models.SavedScript{},
		&models.QueryTab{},
		&models.DBUserCredential{},
		&models.DBInstance{},
		&models.DBMonitoringLog{},
		&models.DBTable{},
		&models.DBColumn{},
		&models.DBEntity{},
		&models.DBPrivilege{},
		&models.DBRoleMembership{},
		&models.UserAIConfig{},
		&models.AIExecutionPolicy{},
	)
	if err != nil {
		log.Fatalf("Database migration failed: %v", err)
	}

	// Remove permissions that have no instance_id (legacy); only instance-scoped permissions are valid.
	if res := DB.Unscoped().Where("instance_id IS NULL").Delete(&models.Permission{}); res.Error != nil {
		log.Printf("Warning: cleanup of permissions without instance_id failed: %v", res.Error)
	} else if res.RowsAffected > 0 {
		log.Printf("Removed %d permission(s) with no instance_id", res.RowsAffected)
	}

	SeedRoles()
	log.Println("Database migration completed")
}

func SeedRoles() {
	roles := []struct {
		Name  string
		DBKey string
	}{
		{"Super Admin", "super_admin"},
		{"Developer", "developer"},
		{"Analyst", "analyst"},
	}
	for _, r := range roles {
		var count int64
		DB.Model(&models.Role{}).Where("name = ?", r.Name).Count(&count)
		if count == 0 {
			role := models.Role{
				Name:         r.Name,
				DBKey:        r.DBKey,
				Description:  "Default role",
				IsSystemRole: true,
			}
			DB.Create(&role)
			log.Printf("Seeded role: %s", r.Name)
		}
	}
}
