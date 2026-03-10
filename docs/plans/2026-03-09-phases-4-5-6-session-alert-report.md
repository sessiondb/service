# Phases 4, 5, 6 (Session, Alert, Report Engines) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement Session Engine (Phase 4), Alert Engine (Phase 5), and Report Engine (Phase 6) as premium-only features, following the full platform design (`docs/plans/2026-03-09-sessiondb-full-platform-design.md`).

**Architecture:** Engine interfaces live in `internal/engine/` (session.go, alert.go, report.go). Premium implementations live in `internal/premium/` under `session/`, `alert/`, `report/`. New models (CredentialSession, AlertRule, AlertEvent, ReportDefinition, ReportExecution) go in `internal/models/models.go`. Repositories can live in `internal/repository/` (shared) or `internal/premium/repository/` (pro-only). Routes are registered in `internal/api/provider_pro.go` (//go:build pro). Stub packages `internal/premium/repository`, `service`, `utils` must exist so `provider_pro.go` compiles.

**Tech Stack:** Go, GORM, Gin, PostgreSQL (metadata), existing dialect layer and access engine.

---

## Phase 4: Session Engine (Premium)

### 4.1 Models and migration
- **Modify:** `internal/models/models.go` — add `CredentialSession` (UserID, InstanceID, SessionType, DBUsername, CredentialID, StartedAt, ExpiresAt, EndedAt, Status).
- **Modify:** `internal/repository/db.go` — AutoMigrate `CredentialSession` (and optionally guard with build tag or always migrate for simplicity).

### 4.2 Engine interface
- **Create:** `internal/engine/session.go` — define `SessionEngine` interface: StartSession(ctx, userID, instanceID, ttl) (CredentialSession, password, error), EndSession(ctx, sessionID) error, GetActiveSession(ctx, userID, instanceID) (*CredentialSession, error).

### 4.3 Premium implementation
- **Create:** `internal/premium/session/engine.go` (//go:build pro) — implement SessionEngine: ephemeral mode creates DB user via dialect, grants from Permission repo, returns CredentialSession + password; EndSession drops user via dialect and updates record.
- **Create:** `internal/repository/credential_session_repository.go` — CRUD for CredentialSession (Create, FindByID, FindActiveByUserAndInstance, Update).
- **Create:** `internal/api/handlers/session_handler.go` (//go:build pro) or in premium — POST /sessions/start, POST /sessions/:id/end, GET /sessions/active.
- **Modify:** `internal/api/provider_pro.go` — wire SessionEngine, session handler, register routes under e.g. `/sessions`.

### 4.4 Query pipeline integration
- **Modify:** `internal/service/query_service.go` — when executing, if user has an active CredentialSession for the instance, use that DB username/password (or session-scoped credential) for the connection. Populate AuditLog.SessionID and DBUsername from session when present.

---

## Phase 5: Alert Engine (Premium)

### 5.1 Models and migration
- **Modify:** `internal/models/models.go` — add `AlertRule` (TenantID, CreatedBy, Name, Description, EventSource, Condition jsonb, Severity, IsEnabled, Channels jsonb), `AlertEvent` (RuleID, TenantID, Severity, Title, Description, Metadata jsonb, Status, Source).
- **Modify:** `internal/repository/db.go` — AutoMigrate AlertRule, AlertEvent.

### 5.2 Engine interface
- **Create:** `internal/engine/alert.go` — define `AlertEngine` interface: CreateRule(ctx, rule) error, EvaluateRules(ctx, eventSource, payload) error, ListRules(ctx, tenantID) ([]AlertRule, error), ListEvents(ctx, tenantID, filter) ([]AlertEvent, error).

### 5.3 Premium implementation
- **Create:** `internal/premium/alert/engine.go` (//go:build pro) — implement AlertEngine; EvaluateRules evaluates Condition (threshold, etc.) and creates AlertEvent on match.
- **Create:** `internal/repository/alert_rule_repository.go`, `alert_event_repository.go` — CRUD.
- **Create:** handlers for CRUD rules and list events; register in provider_pro.go under `/alerts`.

### 5.4 Integration
- **Modify:** Query execution / audit logging path — after significant events, call AlertEngine.EvaluateRules (or enqueue) so rule conditions can fire. Start with a simple hook (e.g. on query execution complete).

---

## Phase 6: Report Engine (Premium)

### 6.1 Models and migration
- **Modify:** `internal/models/models.go` — add `ReportDefinition` (TenantID, CreatedBy, Name, Description, DataSources, Filters, Schedule cron, DeliveryChannels, Format, IsEnabled, LastRunAt), `ReportExecution` (DefinitionID, Status, StartedAt, CompletedAt, ResultURL, Error).
- **Modify:** `internal/repository/db.go` — AutoMigrate ReportDefinition, ReportExecution.

### 6.2 Engine interface
- **Create:** `internal/engine/report.go` — define `ReportEngine` interface: CreateDefinition(ctx, def) error, RunReport(ctx, definitionID) error, ListDefinitions(ctx, tenantID) ([]ReportDefinition, error), ListExecutions(ctx, definitionID) ([]ReportExecution, error).

### 6.3 Premium implementation
- **Create:** `internal/premium/report/engine.go` (//go:build pro) — implement ReportEngine; RunReport runs data queries, generates output (e.g. CSV/JSON), stores ResultURL, updates ReportExecution status.
- **Create:** repositories for ReportDefinition, ReportExecution; handlers for CRUD and run; register in provider_pro.go under `/reports`.
- **Background worker:** Cron or scheduler that finds due ReportDefinition by Schedule and calls RunReport. Can be a goroutine in main or a separate process; start with “run on demand” only if simpler.

---

## Prerequisite: Premium stubs

So `go build -tags pro` succeeds, ensure:
- **Create:** `internal/premium/repository/stubs.go` (//go:build pro) — types QueryInsights, DBMetrics, DBAlters with NewQueryInsights(), NewDBMetrics(), NewDBAlters() returning stub structs.
- **Create:** `internal/premium/service/stubs.go` (//go:build pro) — NewEnhancedRuleEnforcement(), NewAutoCredsExpiry() returning stub structs.
- **Create:** `internal/premium/utils/stubs.go` (//go:build pro) — NewTTLBasedAccess() returning stub.

---

## Summary checklist

| Phase | Deliverables |
|-------|--------------|
| Stubs | premium/repository, service, utils so provider_pro.go compiles |
| 4 Session | CredentialSession model, SessionEngine interface + premium impl, repo, handler, routes, query pipeline session awareness |
| 5 Alert | AlertRule, AlertEvent models, AlertEngine interface + premium impl, repos, handlers, routes, event hook |
| 6 Report | ReportDefinition, ReportExecution models, ReportEngine interface + premium impl, repos, handlers, routes, run-on-demand |

---

## Testing

- Unit tests for engine implementations and repositories (in premium packages with //go:build pro).
- Manual: build with `go build -tags pro`, start server, call new endpoints with auth.
