# BYOK, AI SQL Display, and User Creation — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** (1) Admin-level BYOK with user-override and token-usage dashboard; (2) Fix AI-generated SQL display (strip markdown code fences); (3) User creation with optional “send credentials by email.”

**Architecture:** BYOK: global AI config + resolve user → admin; record usage per call. AI display: strip ```sql/``` in backend and frontend. User flow: optional mail service and sendCredentialsEmail on create.

**Tech Stack:** Go (backend), React/TypeScript (frontend), GORM/Postgres, optional SMTP for email.

---

### Task 1: Backend — stripSQLCodeFence helper and test

**Files:**
- Create: `internal/community/ai/sql_fence.go`
- Create: `internal/community/ai/sql_fence_test.go`

**Step 1: Write the failing test**

In `internal/community/ai/sql_fence_test.go`:

```go
package ai

import "testing"

func TestStripSQLCodeFence(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "SELECT 1", "SELECT 1"},
		{"sql block", "```sql\nSELECT 1\n```", "SELECT 1"},
		{"generic block", "```\nSELECT 1\n```", "SELECT 1"},
		{"with newlines", "  \n```sql\nSELECT * FROM t\n```  \n", "SELECT * FROM t"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripSQLCodeFence(tt.in)
			if got != tt.want {
				t.Errorf("StripSQLCodeFence() = %q, want %q", got, tt.want)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/mouli.b/Documents/personal/projects/servers/sessiondb && go test ./internal/community/ai/ -v -run StripSQLCodeFence`

Expected: FAIL (undefined: StripSQLCodeFence)

**Step 3: Write minimal implementation**

Create `internal/community/ai/sql_fence.go`:

```go
package ai

import (
	"regexp"
	"strings"
)

var sqlBlockRe = regexp.MustCompile(`(?s)^\s*(?:` + "`" + `{3}\s*sql\s*\n?|` + "`" + `{3}\s*\n?)(.*?)` + "`" + `{3}\s*$`)

func StripSQLCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if m := sqlBlockRe.FindStringSubmatch(s); len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	return s
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/community/ai/ -v -run StripSQLCodeFence`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/community/ai/sql_fence.go internal/community/ai/sql_fence_test.go
git commit -m "feat(ai): add StripSQLCodeFence for generated SQL"
```

---

### Task 2: Backend — Wire StripSQLCodeFence in provider

**Files:**
- Modify: `internal/community/ai/provider.go` (around line 109)

**Step 1: Write the failing test**

In `internal/community/ai/provider_test.go` (create if missing), add a test that calls a helper returning fenced SQL and asserts unwrapped output. Alternatively, extend `sql_fence_test.go` with one more case and rely on Task 1; then this task has no new test (integration: manual call to Generate SQL in UI). For strict TDD, add in `internal/community/ai/provider_test.go`:

```go
func TestGenerateSQL_StripsCodeFence(t *testing.T) {
	// Mock HTTP to return content with ```sql\nSELECT 1\n```; assert provider returns "SELECT 1"
	// If no mock in repo, skip and do Step 3 only; verify manually.
	t.Skip("integration: verify via API or mock in follow-up")
}
```

**Step 2: Run test**

Run: `go test ./internal/community/ai/ -v -run TestGenerateSQL`

Expected: SKIP or PASS.

**Step 3: Write minimal implementation**

In `internal/community/ai/provider.go`, in `chat()`, change the return line from:

```go
return out.Choices[0].Message.Content, nil
```

to:

```go
return StripSQLCodeFence(out.Choices[0].Message.Content), nil
```

**Step 4: Run all ai package tests**

Run: `go test ./internal/community/ai/ -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/community/ai/provider.go
git commit -m "feat(ai): strip markdown code fence from generated SQL"
```

---

### Task 3: Frontend — stripSqlFromMarkdown and use in Editor

**Files:**
- Create: `src/utils/sqlMarkdown.ts`
- Modify: `src/pages/Query/Editor.tsx` (handleGenerateSubmit, ~line 303)

**Step 1: Write the failing test (optional; repo may not have frontend unit tests for utils)**

If `src/utils/` has no test setup, skip to Step 3 and verify manually. Otherwise create `src/utils/sqlMarkdown.test.ts`:

```ts
import { stripSqlFromMarkdown } from './sqlMarkdown';
expect(stripSqlFromMarkdown('```sql\nSELECT 1\n```')).toBe('SELECT 1');
```

**Step 2: Run test (if any)**

Run: `npm test -- sqlMarkdown` or `npm run build`

Expected: FAIL (module not found) or build error until Step 3.

**Step 3: Write minimal implementation**

Create `src/utils/sqlMarkdown.ts`:

```ts
/**
 * Strips markdown code fence (```sql or ```) from AI-generated SQL so the editor shows plain SQL.
 */
export function stripSqlFromMarkdown(sql: string): string {
  let s = (sql ?? '').trim();
  const sqlMatch = /^\s*```\s*sql\s*\n?([\s\S]*?)```\s*$/i.exec(s);
  if (sqlMatch) return sqlMatch[1].trim();
  const genericMatch = /^\s*```\s*\n?([\s\S]*?)```\s*$/.exec(s);
  if (genericMatch) return genericMatch[1].trim();
  return s;
}
```

In `src/pages/Query/Editor.tsx`, add import and change:

```ts
import { stripSqlFromMarkdown } from '../../utils/sqlMarkdown';
// ...
if (activeTab) {
  handleQueryChange(stripSqlFromMarkdown(data.sql));
}
```

**Step 4: Verify**

Run: `npm run build`

Expected: build succeeds. Manually: Generate SQL with AI, confirm editor shows plain SQL without ``` or backslash.

**Step 5: Commit**

```bash
git add src/utils/sqlMarkdown.ts src/pages/Query/Editor.tsx
git commit -m "fix(ui): strip markdown from AI SQL in editor"
```

---

### Task 4: Backend — GlobalAIConfig model and migration

**Files:**
- Modify: `internal/models/models.go` (after UserAIConfig, ~line 261)
- Modify: `internal/repository/db.go` (AutoMigrate list)

**Step 1: Write the failing test**

In `internal/repository/ai_config_repository_test.go` (create if missing) or a new test file:

```go
func TestGetGlobalAIConfig_Empty(t *testing.T) {
	db := setupTestDB(t) // or use existing test DB helper
	repo := NewAIConfigRepository(db)
	cfg, err := repo.GetGlobalAIConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg != nil {
		t.Errorf("expected nil when no global config")
	}
}
```

If no test DB helper exists, skip test and add model + migrate; verify with `go build ./...`.

**Step 2: Run test**

Run: `go test ./internal/repository/ -run GetGlobalAIConfig -v`

Expected: FAIL (GetGlobalAIConfig undefined).

**Step 3: Write minimal implementation**

In `internal/models/models.go`, after `UserAIConfig` struct, add:

```go
// GlobalAIConfig stores the organization-wide AI API key (admin-set). Single row per deployment.
type GlobalAIConfig struct {
	Base
	ProviderType string  `gorm:"not null" json:"providerType"`
	APIKey       string  `gorm:"not null" json:"-"`
	BaseURL      *string `json:"baseUrl,omitempty"`
	ModelName    string  `json:"modelName"`
}
```

In `internal/repository/db.go`, add `&models.GlobalAIConfig{}` to the `AutoMigrate` list.

In `internal/repository/ai_config_repository.go`, add:

```go
func (r *AIConfigRepository) GetGlobalAIConfig() (*models.GlobalAIConfig, error) {
	var cfg models.GlobalAIConfig
	err := r.DB.First(&cfg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &cfg, nil
}

func (r *AIConfigRepository) SaveGlobalAIConfig(cfg *models.GlobalAIConfig) error {
	return r.DB.Save(cfg).Error
}
```

Add imports: `"errors"`, `"gorm.io/gorm"` if not present.

**Step 4: Run tests and build**

Run: `go build ./...` and `go test ./internal/repository/ -run AIConfig -v` (or full repo test).

Expected: PASS / build OK.

**Step 5: Commit**

```bash
git add internal/models/models.go internal/repository/db.go internal/repository/ai_config_repository.go
git commit -m "feat(ai): add GlobalAIConfig model and repo"
```

---

### Task 5: Backend — AI engine resolve user then global config

**Files:**
- Modify: `internal/community/ai/engine.go` (GenerateSQL and ExplainQuery)

**Step 1: Write the failing test**

In `internal/community/ai/engine_test.go` (create if missing):

```go
func TestEngine_GenerateSQL_UsesGlobalWhenNoUserConfig(t *testing.T) {
	// Mock AIConfigRepo: GetUserAIConfig returns nil, GetGlobalAIConfig returns valid config.
	// Call GenerateSQL and assert provider is called (or assert no errNoAIConfig).
	t.Skip("requires mock; implement when engine is refactored for injectable deps")
}
```

Or skip and do Step 3; verify manually: remove user AI config, set global, call Generate SQL.

**Step 2: Run test**

Run: `go test ./internal/community/ai/ -v`

Expected: SKIP or existing tests PASS.

**Step 3: Write minimal implementation**

In `internal/community/ai/engine.go`, add a helper:

```go
func (e *Engine) getAIConfigForUser(ctx context.Context, userID uuid.UUID) (providerType string, apiKey string, baseURL string, modelName string, useGlobal bool, err error) {
	uc, err := e.AIConfigRepo.GetUserAIConfig(userID)
	if err == nil && uc != nil && uc.APIKey != "" {
		dec, err := utils.DecryptPassword(uc.APIKey)
		if err != nil {
			return "", "", "", "", false, errInvalidAIConfig
		}
		base := ""
		if uc.BaseURL != nil {
			base = *uc.BaseURL
		}
		return uc.ProviderType, dec, base, uc.ModelName, false, nil
	}
	gc, err := e.AIConfigRepo.GetGlobalAIConfig()
	if err != nil || gc == nil || gc.APIKey == "" {
		return "", "", "", "", false, errNoAIConfig
	}
	dec, err := utils.DecryptPassword(gc.APIKey)
	if err != nil {
		return "", "", "", "", false, errInvalidAIConfig
	}
	base := ""
	if gc.BaseURL != nil {
		base = *gc.BaseURL
	}
	return gc.ProviderType, dec, base, gc.ModelName, true, nil
}
```

Then in `GenerateSQL`, replace the block that gets user config with a call to `getAIConfigForUser` and build the provider from the returned values. Do the same for `ExplainQuery`. Ensure both paths set a variable or context so the handler can know “used global” for the response (e.g. return a struct with SQL + UsedGlobal, or set it in context). For minimal change, only resolve order is required; “source” in GET /ai/config can be added in Task 6.

**Step 4: Run tests**

Run: `go test ./internal/community/ai/ ./cmd/server/ -v` and `go build ./...`

Expected: PASS.

**Step 5: Commit**

```bash
git add internal/community/ai/engine.go
git commit -m "feat(ai): resolve AI config user first then global"
```

---

### Task 6: Backend — GET /ai/config source and PUT /admin/ai-config

**Files:**
- Modify: `internal/api/handlers/ai_handler.go`
- Modify: `cmd/server/main.go` (register PUT /admin/ai-config with admin permission)

**Step 1:** No new unit test; handler tests can be added later. Proceed to implementation.

**Step 2:** In `GetAIConfig`, after loading user config and/or global config, return JSON with `source: "user"` or `source: "global"` and `configured: true/false`.

**Step 3:** Add handler `UpdateGlobalAIConfig` that reads body (same shape as UpdateAIConfig), encrypts API key, calls `AIConfigRepo.SaveGlobalAIConfig`. Register in main under admin group: `admin.PUT("/ai-config", aiHandler.UpdateGlobalAIConfig)` with `CheckPermission(utils.PermInstancesManage)` or a dedicated admin permission.

**Step 4:** Run `go build ./...` and manual test: GET /v1/ai/config returns source; PUT /v1/admin/ai-config (as admin) saves global config.

**Step 5: Commit**

```bash
git add internal/api/handlers/ai_handler.go cmd/server/main.go
git commit -m "feat(api): GET ai/config source and PUT admin/ai-config"
```

---

### Task 7: Backend — AITokenUsage model and RecordUsage

**Files:**
- Modify: `internal/models/models.go`
- Modify: `internal/repository/db.go`
- Create: `internal/repository/ai_usage_repository.go`

**Step 1:** Add model `AITokenUsage` with: UserID (uuid), UsedGlobal (bool), Timestamp (time.Time), InputTokens, OutputTokens (int), Model, Feature (string). Add to AutoMigrate. Create repo with `RecordUsage(usage *models.AITokenUsage) error` and `GetUsageByUser(userID uuid.UUID, from, to time.Time) ([]models.AITokenUsage, error)`.

**Step 2:** In AI engine, after successful GenerateSQL/ExplainQuery, call usageRepo.RecordUsage with userID, usedGlobal, feature name, and token counts (0 if provider doesn’t return). Inject AITokenUsageRepository into engine (or into handler and record from handler).

**Step 3:** Add `GET /ai/usage` (current user) and `GET /admin/ai/usage` (admin, aggregated). Implement in handler and main.

**Step 4:** Build and run tests.

**Step 5: Commit**

```bash
git add internal/models/models.go internal/repository/db.go internal/repository/ai_usage_repository.go internal/community/ai/engine.go internal/api/handlers/ai_handler.go cmd/server/main.go
git commit -m "feat(ai): token usage tracking and usage API"
```

---

### Task 8: Frontend — Admin AI config and “Using your/org key” badge

**Files:**
- Modify: `src/api/ai.ts` (add updateGlobalAIConfig, getAIConfig response type with source)
- Modify: `src/pages/Admin/SettingsAIConfig.tsx` (admin section for org key, badge for source)

**Step 1:** Extend `getAIConfig` response type with `source?: 'user' | 'global'`. Add `updateGlobalAIConfig(params)` calling `PUT /admin/ai-config`.

**Step 2:** In SettingsAIConfig, if user is admin (e.g. role from AuthContext), render “Organization AI config” form and “Your AI config” form. Show badge: “Using your key” when source === 'user', “Using organization key” when source === 'global'.

**Step 3:** Wire admin form submit to `updateGlobalAIConfig`. Ensure Query page can show same badge (e.g. from getAIConfig).

**Step 4:** `npm run build` and manual check.

**Step 5: Commit**

```bash
git add src/api/ai.ts src/pages/Admin/SettingsAIConfig.tsx
git commit -m "feat(ui): admin AI config and source badge"
```

---

### Task 9: Frontend — AI usage dashboard and badges

**Files:**
- Modify: `src/api/ai.ts` (getAIUsage, getAdminAIUsage)
- Create or modify: `src/pages/Admin/` (AI usage section or page with table/cards and badges)

**Step 1:** Add `getAIUsage()` and `getAdminAIUsage()` in ai.ts.

**Step 2:** Add admin UI that fetches `getAdminAIUsage()` and displays global + per-user usage (e.g. “X requests this month”). Add optional badge for current user on Query page from `getAIUsage()`.

**Step 3:** Build and manual test.

**Step 4: Commit**

```bash
git add src/api/ai.ts src/pages/Admin/*
git commit -m "feat(ui): AI usage dashboard and user badge"
```

---

### Task 10: Backend — Mail service and sendCredentialsEmail on user create

**Files:**
- Create: `internal/service/mail_service.go`
- Modify: `internal/config/config.go`
- Modify: `internal/api/handlers/user_handler.go` (CreateUser body and post-create email)

**Step 1:** Add config struct fields: MailEnabled, MailFrom, SMTPHost, SMTPPort, SMTPUser, SMTPPassword (or SendGrid API key). Load from env.

**Step 2:** In `mail_service.go`, implement `SendCredentialsEmail(to, userName, tempPassword, loginURL string) error`. If MailEnabled is false, return nil. Otherwise send via SMTP (or SendGrid). No-op implementation is acceptable for first pass (log and return nil).

**Step 3:** In CreateUser request struct add `SendCredentialsEmail bool`. After Service.Create, if SendCredentialsEmail && mail enabled, call mailService.SendCredentialsEmail(createdUser.Email, createdUser.Name, req.Password, appURL). Do not log req.Password. Return 201; optionally add response field `emailSent: true/false`.

**Step 4:** Build and test.

**Step 5: Commit**

```bash
git add internal/config/config.go internal/service/mail_service.go internal/api/handlers/user_handler.go
git commit -m "feat(users): optional send credentials by email on create"
```

---

### Task 11: Frontend — “Send credentials by email” checkbox

**Files:**
- Modify: `src/pages/Admin/UserModal.tsx`
- Modify: `src/api/` (users create payload type and call)

**Step 1:** Add checkbox state “Send credentials to user by email” and include `sendCredentialsEmail: boolean` in create user API payload.

**Step 2:** On success, show message: “User created. Credentials have been sent by email.” or “User created. Email could not be sent; share credentials manually.” if backend returns emailSent: false.

**Step 3:** Build and manual test.

**Step 4: Commit**

```bash
git add src/pages/Admin/UserModal.tsx src/api/*
git commit -m "feat(ui): send credentials by email on user create"
```

---

### Task 12: Documentation and existing-user semantics

**Files:**
- Create: `docs/features/ai-query.md`
- Create: `docs/features/user-creation-and-credentials.md`
- Modify: `internal/service/db_user_provisioning_service.go` or `user_service.go` (comments only)

**Step 1:** In ai-query.md document: BYOK admin vs user, resolve order, usage dashboard, and that generated SQL is stripped of markdown.

**Step 2:** In user-creation-and-credentials.md document: admin creates user → platform + DB users; optional email; “existing user” = platform user already exists; provisioning on new instance = create DB user if none (no catalog crawl unless future “link DB user” feature).

**Step 3:** Add short comments in provisioning and user service where relevant.

**Step 4: Commit**

```bash
git add docs/features/*.md internal/service/*.go
git commit -m "docs: BYOK and user creation flows"
```

---

## Execution order (recommended)

1. Task 1 → Task 2 → Task 3 (Bug #2: AI SQL display).
2. Task 4 → Task 5 → Task 6 (BYOK: global config and resolve).
3. Task 7 → Task 8 → Task 9 (Usage tracking and UI).
4. Task 10 → Task 11 (Mail and send-credentials UI).
5. Task 12 (Docs and comments).

---

## Summary

| Area        | Tasks   | Deliverables                                      |
|------------|---------|---------------------------------------------------|
| Bug #2 SQL | 1, 2, 3 | StripSQLCodeFence + provider + frontend strip    |
| Bug #1 BYOK| 4–9     | GlobalAIConfig, resolve order, usage, dashboard   |
| Bug #3 User| 10, 11, 12 | Mail service, sendCredentialsEmail, checkbox, docs |
