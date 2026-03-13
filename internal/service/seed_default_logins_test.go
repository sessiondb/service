// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package service

import (
	"sessiondb/internal/config"
	"sessiondb/internal/models"
	"sessiondb/internal/repository"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupSeedTestDB opens an in-memory SQLite DB, creates roles and users tables with SQLite-compatible
// DDL (GORM AutoMigrate emits Postgres-specific defaults), and seeds one role with key "super_admin".
func setupSeedTestDB(t *testing.T) (*gorm.DB, *repository.UserRepository, *repository.RoleRepository) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	// Create tables without Postgres-specific defaults so SQLite accepts them
	for _, q := range []string{
		`CREATE TABLE roles (id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, name TEXT NOT NULL, key TEXT NOT NULL, description TEXT, is_system_role INTEGER DEFAULT 0)`,
		`CREATE TABLE users (id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, name TEXT NOT NULL, email TEXT NOT NULL UNIQUE, password_hash TEXT, role_id TEXT, status TEXT DEFAULT 'active', is_session_based INTEGER DEFAULT 0, session_expires_at DATETIME, last_login DATETIME, sso_id TEXT, rbac_permissions TEXT)`,
	} {
		if err := db.Exec(q).Error; err != nil {
			t.Fatalf("create table: %v", err)
		}
	}
	// Seed one role so FindByKey("super_admin") works
	r := models.Role{Name: "Super Admin", Key: "super_admin", IsSystemRole: true}
	if err := db.Create(&r).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}
	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	return db, userRepo, roleRepo
}

func TestSeedDefaultLogins_CreatesUserWhenTableEmpty(t *testing.T) {
	db, userRepo, roleRepo := setupSeedTestDB(t)
	cfg := &config.Config{
		DefaultLogins: []config.DefaultLogin{
			{Email: "admin@example.com", Password: "admin123", RoleKey: "super_admin"},
		},
	}

	err := SeedDefaultLogins(cfg, userRepo, roleRepo)
	if err != nil {
		t.Fatalf("SeedDefaultLogins: %v", err)
	}

	count, err := userRepo.Count()
	if err != nil {
		t.Fatalf("Count after seed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 user after seed, got count=%d", count)
	}
	var email string
	if err := db.Raw("SELECT email FROM users WHERE email = ?", "admin@example.com").Scan(&email).Error; err != nil {
		t.Fatalf("lookup user by email: %v", err)
	}
	if email != "admin@example.com" {
		t.Errorf("got email %q", email)
	}
	// Sanity: user has password hash and role_id set (raw select to avoid FindByEmail preloads)
	var row struct {
		PasswordHash string
		RoleID       uuid.UUID
	}
	if err := db.Raw("SELECT password_hash, role_id FROM users WHERE email = ?", "admin@example.com").Scan(&row).Error; err != nil {
		t.Fatalf("lookup user fields: %v", err)
	}
	if row.PasswordHash == "" {
		t.Error("expected PasswordHash to be set")
	}
	if row.RoleID == uuid.Nil {
		t.Error("expected RoleID to be set")
	}
}

func TestSeedDefaultLogins_DoesNotCreateWhenUserExists(t *testing.T) {
	_, userRepo, roleRepo := setupSeedTestDB(t)
	// Create one user so count > 0
	role, _ := roleRepo.FindByKey("super_admin")
	if role == nil {
		t.Fatal("need role for existing user")
	}
	existing := &models.User{
		Name: "Existing", Email: "existing@example.com", PasswordHash: "hash", RoleID: role.ID,
	}
	if err := userRepo.Create(existing); err != nil {
		t.Fatalf("create existing user: %v", err)
	}

	cfg := &config.Config{
		DefaultLogins: []config.DefaultLogin{
			{Email: "admin@example.com", Password: "admin123", RoleKey: "super_admin"},
		},
	}
	err := SeedDefaultLogins(cfg, userRepo, roleRepo)
	if err != nil {
		t.Fatalf("SeedDefaultLogins: %v", err)
	}

	count, err := userRepo.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 user after seed when one already existed, got count=%d", count)
	}
}
