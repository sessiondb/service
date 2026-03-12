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

// setupApprovalTestDB creates an in-memory SQLite DB with minimal tables for approval tests (SQLite-compatible DDL).
// Returns db, approvalRepo, and a valid requesterID (created user).
func setupApprovalTestDB(t *testing.T) (*gorm.DB, *repository.ApprovalRepository, uuid.UUID) {
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
	} {
		if err := db.Exec(q).Error; err != nil {
			t.Fatalf("create table: %v", err)
		}
	}
	roleID := "11111111-1111-1111-1111-111111111111"
	userID := "22222222-2222-2222-2222-222222222222"
	if err := db.Exec(`INSERT INTO roles (id, name, key, is_system_role) VALUES (?, ?, ?, ?)`, roleID, "Admin", "admin", 1).Error; err != nil {
		t.Fatalf("insert role: %v", err)
	}
	if err := db.Exec(`INSERT INTO users (id, name, email, role_id, status) VALUES (?, ?, ?, ?, ?)`, userID, "Requester", "req@test.com", roleID, "active").Error; err != nil {
		t.Fatalf("insert user: %v", err)
	}
	approvalRepo := repository.NewApprovalRepository(db)
	return db, approvalRepo, uuid.MustParse(userID)
}

func TestApprovalService_CreateRequest_StoresRequestedItems(t *testing.T) {
	_, approvalRepo, requesterID := setupApprovalTestDB(t)
	svc := NewApprovalService(approvalRepo)

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
	_, approvalRepo, requesterID := setupApprovalTestDB(t)
	svc := NewApprovalService(approvalRepo)

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

