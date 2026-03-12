// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package service

import (
	"encoding/json"
	"sessiondb/internal/models"
	"sessiondb/internal/repository"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// mockProvisioner implements DBUserProvisioner for tests. It records calls and returns a dummy cred.
type mockProvisioner struct {
	ProvisionDBUserCalls int
	GrantPermissionsCalls int
	GrantPermissionsPerms [][]models.Permission
}

func (m *mockProvisioner) ProvisionDBUser(user *models.User, instance *models.DBInstance) (*models.DBUserCredential, error) {
	m.ProvisionDBUserCalls++
	return &models.DBUserCredential{UserID: user.ID, InstanceID: instance.ID, DBUsername: "testuser"}, nil
}

func (m *mockProvisioner) GrantPermissions(cred *models.DBUserCredential, permissions []models.Permission) error {
	m.GrantPermissionsCalls++
	m.GrantPermissionsPerms = append(m.GrantPermissionsPerms, permissions)
	return nil
}

// setupApprovalTestDB creates an in-memory SQLite DB with minimal tables for approval tests (SQLite-compatible DDL).
// Returns db, approvalRepo, permRepo, instanceRepo, userRepo, mockProv, and a valid requesterID (created user).
func setupApprovalTestDB(t *testing.T) (*gorm.DB, *repository.ApprovalRepository, *repository.PermissionRepository, *repository.InstanceRepository, *repository.UserRepository, *mockProvisioner, uuid.UUID) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	// SQLite-compatible tables (no gen_random_uuid / uuid type)
	for _, q := range []string{
		`CREATE TABLE roles (id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, name TEXT NOT NULL, key TEXT NOT NULL, description TEXT, is_system_role INTEGER DEFAULT 0)`,
		`CREATE TABLE users (id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, name TEXT NOT NULL, email TEXT NOT NULL, password_hash TEXT, role_id TEXT, status TEXT DEFAULT 'active', is_session_based INTEGER DEFAULT 0, session_expires_at DATETIME, last_login DATETIME, sso_id TEXT, rbac_permissions TEXT)`,
		`CREATE TABLE approval_requests (id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, type TEXT NOT NULL, requester_id TEXT, target_user_id TEXT, description TEXT, justification TEXT, requested_permissions TEXT, requested_items TEXT, status TEXT DEFAULT 'pending', reviewed_by TEXT, reviewed_at DATETIME, approved_permissions TEXT, rejection_reason TEXT, expires_at DATETIME)`,
		`CREATE TABLE permissions (id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, role_id TEXT, user_id TEXT, instance_id TEXT, database TEXT NOT NULL, schema TEXT, "table" TEXT NOT NULL, column TEXT, privileges TEXT, type TEXT, expires_at DATETIME, granted_by TEXT, schedule_cron TEXT)`,
		`CREATE TABLE db_instances (id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, name TEXT NOT NULL, host TEXT NOT NULL, port INTEGER NOT NULL, type TEXT NOT NULL, username TEXT NOT NULL, password TEXT NOT NULL, status TEXT, last_sync DATETIME, is_prod INTEGER DEFAULT 0, monitoring_enabled INTEGER DEFAULT 0, alert_email TEXT)`,
		`CREATE TABLE db_user_credentials (id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, user_id TEXT NOT NULL, instance_id TEXT NOT NULL, db_username TEXT NOT NULL, db_password TEXT NOT NULL, role TEXT, expires_at DATETIME, status TEXT)`,
		`CREATE TABLE query_tabs (id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, user_id TEXT, name TEXT, query TEXT, is_active INTEGER)`,
		`CREATE TABLE saved_scripts (id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, user_id TEXT, name TEXT, description TEXT, query TEXT, is_public INTEGER)`,
	} {
		if err := db.Exec(q).Error; err != nil {
			t.Fatalf("create table: %v", err)
		}
	}
	roleID := "11111111-1111-1111-1111-111111111111"
	userID := "22222222-2222-2222-2222-222222222222"
	instanceID := "33333333-3333-3333-3333-333333333333"
	if err := db.Exec(`INSERT INTO roles (id, name, key, is_system_role) VALUES (?, ?, ?, ?)`, roleID, "Admin", "admin", 1).Error; err != nil {
		t.Fatalf("insert role: %v", err)
	}
	if err := db.Exec(`INSERT INTO users (id, name, email, role_id, status) VALUES (?, ?, ?, ?, ?)`, userID, "Requester", "req@test.com", roleID, "active").Error; err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if err := db.Exec(`INSERT INTO db_instances (id, name, host, port, type, username, password, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		instanceID, "TestInstance", "localhost", 5432, "postgres", "admin", "secret", "online").Error; err != nil {
		t.Fatalf("insert instance: %v", err)
	}
	approvalRepo := repository.NewApprovalRepository(db)
	permRepo := repository.NewPermissionRepository(db)
	instanceRepo := repository.NewInstanceRepository(db)
	userRepo := repository.NewUserRepository(db)
	mockProv := &mockProvisioner{}
	return db, approvalRepo, permRepo, instanceRepo, userRepo, mockProv, uuid.MustParse(userID)
}

func TestApprovalService_CreateRequest_StoresRequestedItems(t *testing.T) {
	_, approvalRepo, permRepo, instanceRepo, userRepo, mockProv, requesterID := setupApprovalTestDB(t)
	svc := NewApprovalService(approvalRepo, permRepo, mockProv, instanceRepo, userRepo)

	requestedItems := []models.RequestedItem{
		{
			InstanceID: uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			Database:   "public",
			Table:      "users",
			Privileges: []string{"SELECT", "INSERT"},
		},
	}
	requestedItemsJSON, err := json.Marshal(requestedItems)
	if err != nil {
		t.Fatalf("marshal requested items: %v", err)
	}

	req, err := svc.CreateRequest(requesterID, "PERM_UPGRADE", "Need access", "For report", nil, requestedItemsJSON)
	if err != nil {
		t.Fatalf("CreateRequest: %v", err)
	}
	if req.ID == uuid.Nil {
		t.Error("expected non-nil request ID")
	}
	if len(req.RequestedItems) == 0 {
		t.Error("expected RequestedItems to be stored")
	}
	var decoded []models.RequestedItem
	if err := json.Unmarshal(req.RequestedItems, &decoded); err != nil {
		t.Fatalf("unmarshal stored requested items: %v", err)
	}
	if len(decoded) != 1 {
		t.Fatalf("decoded requested items length = %d, want 1", len(decoded))
	}
	if decoded[0].Database != "public" || decoded[0].Table != "users" {
		t.Errorf("decoded item: database=%q table=%q", decoded[0].Database, decoded[0].Table)
	}
}

func TestApprovalService_GetAllRequests_ReturnsRequestedItems(t *testing.T) {
	_, approvalRepo, permRepo, instanceRepo, userRepo, mockProv, requesterID := setupApprovalTestDB(t)
	svc := NewApprovalService(approvalRepo, permRepo, mockProv, instanceRepo, userRepo)

	requestedItems := []models.RequestedItem{
		{
			InstanceID: uuid.New(),
			Database:   "analytics",
			Table:      "events",
			Privileges: []string{"SELECT"},
		},
	}
	requestedItemsJSON, _ := json.Marshal(requestedItems)

	_, err := svc.CreateRequest(requesterID, "TEMP_USER", "Temp access", "Audit", []byte("[]"), requestedItemsJSON)
	if err != nil {
		t.Fatalf("CreateRequest: %v", err)
	}

	all, err := svc.GetAllRequests()
	if err != nil {
		t.Fatalf("GetAllRequests: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("GetAllRequests length = %d, want 1", len(all))
	}
	r := all[0]
	if len(r.RequestedItems) == 0 {
		t.Fatal("GetAllRequests: expected requestedItems to be returned")
	}
	var items []models.RequestedItem
	if err := json.Unmarshal(r.RequestedItems, &items); err != nil {
		t.Fatalf("unmarshal requested items from GetAllRequests: %v", err)
	}
	if len(items) != 1 || items[0].Database != "analytics" || items[0].Table != "events" {
		t.Errorf("requestedItems = %+v", items)
	}
}

// TestApprovalService_ApproveRequest_SideEffects_CreatesPermissionsAndGrants verifies that ApplyApprovalSideEffects (used by ApproveRequest)
// creates Permission records and invokes ProvisionDBUser and GrantPermissions via the provisioner.
// This covers the side-effects logic invoked by ApproveRequest when the request has RequestedItems.
func TestApprovalService_ApproveRequest_SideEffects_CreatesPermissionsAndGrants(t *testing.T) {
	_, approvalRepo, permRepo, instanceRepo, userRepo, mockProv, requesterID := setupApprovalTestDB(t)
	svc := NewApprovalService(approvalRepo, permRepo, mockProv, instanceRepo, userRepo)

	instanceID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	requestedItems := []models.RequestedItem{
		{InstanceID: instanceID, Database: "public", Table: "users", Privileges: []string{"SELECT", "INSERT"}},
	}
	requestedItemsJSON, err := json.Marshal(requestedItems)
	if err != nil {
		t.Fatalf("marshal requested items: %v", err)
	}
	reviewerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	request := &models.ApprovalRequest{
		RequesterID:    requesterID,
		ReviewedBy:     &reviewerID,
		RequestedItems: requestedItemsJSON,
	}

	err = svc.ApplyApprovalSideEffects(request)
	if err != nil {
		t.Fatalf("ApplyApprovalSideEffects: %v", err)
	}

	if mockProv.ProvisionDBUserCalls != 1 {
		t.Errorf("ProvisionDBUserCalls = %d, want 1", mockProv.ProvisionDBUserCalls)
	}
	if mockProv.GrantPermissionsCalls != 1 {
		t.Errorf("GrantPermissionsCalls = %d, want 1", mockProv.GrantPermissionsCalls)
	}
	if len(mockProv.GrantPermissionsPerms) != 1 || len(mockProv.GrantPermissionsPerms[0]) != 1 {
		t.Errorf("GrantPermissionsPerms = %+v", mockProv.GrantPermissionsPerms)
	}

	perms, err := permRepo.FindByUserID(requesterID)
	if err != nil {
		t.Fatalf("FindByUserID: %v", err)
	}
	if len(perms) != 1 {
		t.Fatalf("permissions count = %d, want 1", len(perms))
	}
	if perms[0].Database != "public" || perms[0].Table != "users" {
		t.Errorf("permission = %+v", perms[0])
	}
}

