# TOML Config, Default Logins, and Request/Approval Flow — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add TOML default config (server + auth) with default platform logins seeded once on first run; extend approval requests with requested_items (instance/database/table/privileges); on approve, create Permission records and provision DB user with grants; add frontend request form (no-access screen + in-editor button) and admin refresh.

**Architecture:** Config loads .env then merges optional config.default.toml; default logins seeded in main after Migrate() when user table is empty. ApprovalRequest gains requested_items (JSONB); ApproveRequest in approval_service creates Permission(s), provisions DB user via existing DBUserProvisioningService, and calls GrantPermissions per item. Frontend: one shared form component; two entry points (no-access page, query editor button); admin list shows requested_items and has Refresh.

**Tech Stack:** Go (Viper, TOML), GORM, Gin; React/TypeScript (frontend), axios. See @testing-and-tdd, @documentation-standards.

**Design doc:** `docs/plans/2026-03-12-toml-approval-request-access-design.md`

---

## Phase 1: TOML config and default logins (backend)

### Task 1: Add TOML config file and struct

**Files:**
- Create: `config.default.toml`
- Modify: `internal/config/config.go`

**Step 1: Create example TOML**

Create `config.default.toml` at repo root:

```toml
[server]
port = "8080"
mode = "debug"

[auth]
# Optional: JWT defaults could go here later

[[auth.default_logins]]
email = "admin@example.com"
password = "admin123"
role_key = "super_admin"

[[auth.default_logins]]
email = "guest@example.com"
password = "guest123"
role_key = "analyst"
```

**Step 2: Add DefaultLogin struct and DefaultLogins to config**

In `internal/config/config.go` add:

```go
// DefaultLogin is a platform user to seed when no users exist (from TOML).
type DefaultLogin struct {
	Email    string `mapstructure:"email"`
	Password string `mapstructure:"password"`
	RoleKey  string `mapstructure:"role_key"`
}
```

Add to `Config` struct:

```go
DefaultLogins []DefaultLogin `mapstructure:"default_logins"`
```

Add a new struct for TOML-only shape (e.g. `tomlConfig`) or use a separate Viper read for the TOML file to populate `DefaultLogins`. Load TOML in `LoadConfig()`: after existing viper .env read, if `config.default.toml` exists (or path from env `CONFIG_TOML_PATH`), read it and unmarshal `[auth]` section into a struct that has `DefaultLogins []DefaultLogin`; set `config.DefaultLogins` from it. If TOML is missing, leave `DefaultLogins` nil. Ensure env still overrides server port/mode (existing behavior).

**Step 3: Unit test for LoadConfig with TOML**

Create or extend test: `internal/config/config_test.go`. Test that when a temp TOML exists with `[[auth.default_logins]]`, `LoadConfig()` returns config with `DefaultLogins` populated. Use `os.WriteFile` to create a temp TOML, set `viper.SetConfigFile` to it (or pass path), then load. Clean up temp file.

**Step 4: Run test**

Run: `go test ./internal/config/... -v -run DefaultLogins`  
Expected: PASS (or fix until pass).

**Step 5: Commit**

```bash
git add config.default.toml internal/config/config.go internal/config/config_test.go
git commit -m "config: add TOML default config and DefaultLogins struct"
```

---

### Task 2: Seed default logins once on first run

**Files:**
- Create: `internal/config/seed.go` (or `internal/service/seed_default_logins.go`)
- Modify: `cmd/server/main.go`
- Test: `internal/config/seed_test.go` or `internal/service/seed_test.go`

**Step 1: Write function that seeds default logins**

Create `SeedDefaultLogins(cfg *config.Config, userRepo *repository.UserRepository, roleRepo *repository.RoleRepository) error`. Logic: if `cfg.DefaultLogins == nil` or len 0, return nil. Query `userRepo` (or DB) for `SELECT COUNT(*) FROM users`; if count > 0, return nil. For each `cfg.DefaultLogins`: find user by email; if not found, find role by key (role_key), hash password (use same hashing as auth), create user with email, password hash, role_id. Use existing `User` model and `UserRepository.Create`. If role not found, skip that default login or return error (design: skip and log).

**Step 2: Write failing test**

In `internal/service/seed_default_logins_test.go` (or config): test that when user table is empty and DefaultLogins has one entry, after SeedDefaultLogins a user exists with that email. Use in-memory DB or testcontainer; ensure count was 0 before, then 1 after. Test that when at least one user already exists, SeedDefaultLogins does not create another user.

**Step 3: Run test**

Run: `go test ./internal/service/... -v -run SeedDefault`  
Expected: FAIL (undefined or wrong behavior) until implementation is done.

**Step 4: Implement SeedDefaultLogins**

Implement in `internal/service/seed_default_logins.go` (or under config). Use `userRepo` to count users (e.g. add `Count() (int64, error)` on UserRepository if not present) and create users. Hash password with same util as auth (e.g. bcrypt). Get role by key: add `FindByKey(key string)` to role repository if missing.

**Step 5: Run test again**

Run: `go test ./internal/service/... -v -run SeedDefault`  
Expected: PASS.

**Step 6: Call SeedDefaultLogins from main**

In `cmd/server/main.go`, after `repository.ConnectDB(cfg)` and after repositories are created (userRepo, roleRepo), call `service.SeedDefaultLogins(cfg, userRepo, roleRepo)`. If error, log and continue (or fail fast; design says log). Respect `SKIP_DEFAULT_LOGINS` env: if set to true/1/yes, skip calling SeedDefaultLogins.

**Step 7: Commit**

```bash
git add internal/service/seed_default_logins.go internal/service/seed_default_logins_test.go cmd/server/main.go internal/repository/user_repository.go internal/repository/role_repository.go
git commit -m "feat: seed default platform logins from TOML once on first run"
```

---

## Phase 2: Approval request items and approve side effects (backend)

### Task 3: Add requested_items to ApprovalRequest model and migration

**Files:**
- Modify: `internal/models/models.go`
- Migration: GORM AutoMigrate already runs; ensure new field is in Migrate() list (already have ApprovalRequest).

**Step 1: Add RequestedItem type and field**

In `internal/models/models.go`, add:

```go
// RequestedItem is one (instance, database, table, privileges) in an approval request.
type RequestedItem struct {
	InstanceID uuid.UUID `json:"instanceId"`
	Database   string    `json:"database"`
	Table      string    `json:"table"`
	Privileges []string  `json:"privileges"`
}
```

Add to `ApprovalRequest`:

```go
RequestedItems []byte `gorm:"type:jsonb" json:"requestedItems,omitempty"` // []RequestedItem
```

**Step 2: Run migration**

Start app or run migrate; ensure DB has new column. Optional: add a small test that creates an ApprovalRequest with RequestedItems and reads it back.

**Step 3: Commit**

```bash
git add internal/models/models.go
git commit -m "feat: add RequestedItems to ApprovalRequest model"
```

---

### Task 4: Approval API accept and return requested_items

**Files:**
- Modify: `internal/api/handlers/approval_handler.go`
- Modify: `internal/service/approval_service.go`

**Step 1: Update CreateRequest DTO and handler**

In `approval_handler.go`, change `CreateRequestDTO` to include:

```go
RequestedItems []models.RequestedItem `json:"requestedItems" binding:"required"`
```

In handler: validate at least one item; each item has non-empty InstanceID, Database, Table, Privileges. Check instance exists (instanceRepo.FindByID). Marshal `RequestedItems` to JSON and pass to service. Service stores in `ApprovalRequest.RequestedItems` ([]byte). Keep existing `Type`, `Description`, `Justification`.

**Step 2: Update GetRequests response**

In `GetRequests`, unmarshal `r.RequestedItems` into `[]models.RequestedItem` and include in response as `requestedItems`. Keep existing fields.

**Step 3: Update UpdateRequestStatus DTO**

For reject, accept optional `rejection_reason` in body (currently hardcoded "Rejected by user"). Use `req.RejectionReason` when provided.

**Step 4: Integration test**

Add or extend test: `internal/api/handlers/approval_handler_test.go` or `internal/service/approval_service_test.go`. Test CreateRequest with requested_items; test GetRequests returns requested_items. Use table-driven test.

**Step 5: Run tests**

Run: `go test ./internal/api/handlers/... ./internal/service/... -v -run Approval`  
Expected: PASS.

**Step 6: Commit**

```bash
git add internal/api/handlers/approval_handler.go internal/service/approval_service.go
git commit -m "feat: approval API accept and return requested_items"
```

---

### Task 5: Approve side effects — create permissions and provision DB user with grants

**Files:**
- Modify: `internal/service/approval_service.go`
- Modify: `cmd/server/main.go` (wire dependencies)

**Step 1: Define dependency for approval service**

ApprovalService needs: PermissionRepository, DBUserProvisioningService, InstanceRepository (to resolve instance), UserRepository (to load requester user). Add these to `NewApprovalService` and store in struct.

**Step 2: Implement ApplyApprovalSideEffects**

In `approval_service.go`, add `ApplyApprovalSideEffects(request *models.ApprovalRequest) error`. Unmarshal `request.RequestedItems` into `[]RequestedItem`. For each item: (1) Load requester user; (2) Load instance; (3) Create Permission for requester: UserID=requester.ID, InstanceID=item.InstanceID, Database=item.Database, Table=item.Table, Privileges=item.Privileges, Schema="" or "public"; use permRepo.Create. (4) Ensure DB user exists: call provisioningService.ProvisionDBUser(requester, instance). (5) Get credential for requester+instance; build Permission list (one Permission per item for this instance); call provisioningService.GrantPermissions(cred, perms). If any step fails, return error (caller will roll back or not update status).

**Step 3: Call from ApproveRequest**

In `ApproveRequest`, after updating status to approved and saving, call `ApplyApprovalSideEffects(request)`. If it returns error, revert request status to pending and return the error to the client (all-or-nothing).

**Step 4: Wire in main**

In `cmd/server/main.go`, when creating ApprovalService, pass permRepo, provisioningService, instanceRepo, userRepo. Ensure provisioningService and instanceRepo are already created.

**Step 5: Test**

Add test: when ApproveRequest is called, verify Permission records are created and GrantPermissions is invoked (mock or integration with test DB). Run: `go test ./internal/service/... -v -run Approve`  
Expected: PASS.

**Step 6: Commit**

```bash
git add internal/service/approval_service.go cmd/server/main.go
git commit -m "feat: on approve create permissions and provision DB user with grants"
```

---

## Phase 3: Frontend (request form, no-access screen, in-editor button, admin refresh)

**Note:** Frontend lives in the sessiondb frontend repo (see workspace rules). Paths below are relative to that repo (e.g. `src/`).

### Task 6: Shared Request DB Access form component

**Files (frontend repo):**
- Create: `src/components/RequestAccessForm.tsx` (or under `src/features/` / `src/pages/`)
- Create: `src/api/requests.ts` (axios helpers for POST /v1/requests, GET /v1/requests)

**Step 1: API helpers**

In `src/api/requests.ts`, add:
- `createRequest(body: { type: string; description: string; justification: string; requestedItems: RequestedItem[] })` → axios.post('/v1/requests', body)
- Type `RequestedItem = { instanceId: string; database: string; table: string; privileges: string[] }`

**Step 2: Form component**

Create `RequestAccessForm.tsx`: props include optional `initialInstanceId`, `initialDatabase`, `initialTable` (for prefill from query editor). State: instanceId, database, tables (array of strings), privileges (array of selected), description, justification. Fetch instances list (existing API). Dropdown for instance; text or dropdown for database; multi-select or dynamic list for tables; checkboxes for SELECT, INSERT, UPDATE, DELETE. Submit: build `requestedItems` (one item per table with same instance, database, privileges), call createRequest, onSuccess callback (close modal, show toast, refresh). Add JSDoc per project rules.

**Step 3: Manual test**

Run frontend; open form (e.g. from a temporary button); submit and verify POST /v1/requests with correct body. No automated test required for this task if project has no frontend test setup; otherwise add a simple render test.

**Step 4: Commit**

```bash
git add src/api/requests.ts src/components/RequestAccessForm.tsx
git commit -m "feat: add shared Request DB Access form component"
```

---

### Task 7: No-access screen (refresh screen)

**Files (frontend repo):**
- Create or modify: `src/pages/NoAccess.tsx` (or equivalent)
- Modify: `src/App.tsx` or router to show this when user has no instance access

**Step 1: No-access page**

Create a page that shows when the user has no permissions to any instance (e.g. after login, check permissions or instance list; if empty, show this page). Content: short message "You don't have access to any database yet" and a button "Request access". Click opens RequestAccessForm (modal or inline). After successful submit, show success message and a "Refresh" or "Check my access" button that refetches user permissions / instance list and redirects to query editor or home if they now have access.

**Step 2: Wire route**

In router, add route for no-access (e.g. `/no-access`) and logic to redirect to it when user has no instance access (e.g. from login or from a guard on query editor route).

**Step 3: Commit**

```bash
git add src/pages/NoAccess.tsx src/App.tsx
git commit -m "feat: add no-access screen with Request access and Refresh"
```

---

### Task 8: In-editor "Request access" button

**Files (frontend repo):**
- Modify: `src/pages/Query/Editor.tsx` (or where instance/database/table selector lives)

**Step 1: Add button and modal**

In the query editor, add a "Request access" (or "Request more access") button near the instance/database/table selector. On click, open RequestAccessForm in a modal with prefill: initialInstanceId, initialDatabase, initialTable from current editor context (selected instance, database, table). On submit success, optionally refetch instance/database/schema list so newly granted access appears.

**Step 2: Commit**

```bash
git add src/pages/Query/Editor.tsx
git commit -m "feat: add Request access button in query editor with context prefill"
```

---

### Task 9: Admin requests list and Refresh

**Files (frontend repo):**
- Modify: `src/pages/Admin/Approvals.tsx` (or Requests page)

**Step 1: Show requested_items**

Ensure GET /v1/requests response is used; display requestedItems for each request (instance name, database, table(s), privileges) in a table or expandable row. Use existing list UI; add columns or detail section for requested_items.

**Step 2: Add Refresh button**

Add a "Refresh" button that refetches GET /v1/requests and updates the list. Optional: periodic refresh (e.g. every 30s) or only on button click.

**Step 3: Rejection reason**

When rejecting, send optional rejection_reason from a text field in the reject modal/flow. Use UpdateRequestStatus with status "rejected" and rejectionReason in body.

**Step 4: Commit**

```bash
git add src/pages/Admin/Approvals.tsx
git commit -m "feat: admin requests list shows requested_items, Refresh button, rejection reason"
```

---

## Phase 4: Documentation and cleanup

### Task 10: Update docs and feature gate

**Files:**
- Modify: `docs/features/run-migrations-once.md` or create `docs/features/default-logins.md`
- Modify: `docs/features/access-control.md` or create `docs/features/request-approval-flow.md`
- Modify: `docs/api-documentation.md` (document POST /v1/requests body with requested_items, GET response)

**Step 1: Feature doc for default logins**

Document: TOML config.default.toml, [server] and [auth.default_logins], seed once on first run, SKIP_DEFAULT_LOGINS.

**Step 2: Feature doc for request/approval**

Document: Requested items (instance, database, table, privileges); create request; on approve, backend creates permissions and provisions DB user with grants; no-access screen and in-editor request button; admin list and refresh.

**Step 3: API doc**

Add/update POST /v1/requests and GET /v1/requests with requestedItems shape. Add PUT /v1/requests/:id rejection_reason.

**Step 4: Commit**

```bash
git add docs/
git commit -m "docs: TOML default logins and request/approval flow"
```

---

## Execution summary

- **Backend (this repo):** Tasks 1–5, 10 (config, seed, model, API, approve side effects, docs).
- **Frontend (frontend repo):** Tasks 6–9 (form, no-access screen, in-editor button, admin list/refresh).

Run backend tests: `go test ./...`  
Run frontend: `npm run dev` (or project script) and manually verify flows.
