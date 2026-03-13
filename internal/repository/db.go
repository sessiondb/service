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

// migrateRolesKey ensures roles.key exists and is backfilled before AutoMigrate (avoids NOT NULL on new column).
func migrateRolesKey() {
	// Skip if roles table does not exist yet (e.g. fresh install; AutoMigrate will create it).
	var exists bool
	if err := DB.Raw(`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'roles')`).Scan(&exists).Error; err != nil || !exists {
		return
	}
	// Rename db_key to key if present.
	DB.Exec(`DO $$ BEGIN IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'roles' AND column_name = 'db_key') THEN ALTER TABLE roles RENAME COLUMN db_key TO key; END IF; END $$`)
	// Add key column if missing (nullable so existing rows are valid).
	DB.Exec(`ALTER TABLE roles ADD COLUMN IF NOT EXISTS key TEXT`)
	// Backfill from name.
	DB.Exec(`UPDATE roles SET key = LOWER(REPLACE(TRIM(name), ' ', '_')) WHERE key IS NULL OR key = ''`)
	// Now safe to set NOT NULL so GORM's AutoMigrate won't try to add the column with NOT NULL.
	DB.Exec(`ALTER TABLE roles ALTER COLUMN key SET NOT NULL`)
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

	// Migrate roles.key BEFORE AutoMigrate so existing rows get backfilled (avoid NOT NULL on new column).
	migrateRolesKey()

	// Ensure approval_requests has requested_items and rejection_reason (for DBs created before these columns existed).
	DB.Exec(`ALTER TABLE approval_requests ADD COLUMN IF NOT EXISTS requested_items JSONB`)
	DB.Exec(`ALTER TABLE approval_requests ADD COLUMN IF NOT EXISTS rejection_reason TEXT`)

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
		&models.GlobalAIConfig{},
		&models.AITokenUsage{},
		&models.AIExecutionPolicy{},
		&models.CredentialSession{},
		&models.AlertRule{},
		&models.AlertEvent{},
		&models.ReportDefinition{},
		&models.ReportExecution{},
		&models.FeatureNotifyRequest{},
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
		Name string
		Key  string
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
				Key:          r.Key,
				Description:  "Default role",
				IsSystemRole: true,
			}
			DB.Create(&role)
			log.Printf("Seeded role: %s", r.Name)
		}
	}
}
