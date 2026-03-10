# SessionDB Full Platform Design

**Date:** 2026-03-09
**Author:** Sai Mouli Bandari
**Status:** Approved

## 1. Overview

SessionDB is an open-core online database management tool providing teams with a unified interface to query databases safely, with granular access control, audit logging, and AI-powered query assistance. The platform ships as two tiers:

- **Community Edition** — open-source, self-hosted, full-featured database management
- **Premium Edition** — proprietary add-ons for enterprise security, compliance, and AI features

### Core Feature Map

| Feature | Community | Premium |
|---|---|---|
| Query panel (multi-DB execution) | Yes | Yes |
| User management (CRUD, roles) | Yes | Yes |
| Audit logging | Yes | Yes |
| Access control (column-to-instance) | Yes | Yes |
| AI query generation & executor (BYOK) | Yes | Yes |
| Session-based credentials | — | Yes |
| Time-based access (column/table TTL) | — | Yes |
| Advanced alerting (custom rules + AI) | — | Yes |
| Query reporting (scheduled + AI) | — | Yes |
| Voice-based prompting | — | Yes |
| Pluggable notification channels | WebSocket only | Email + Webhook + extensible |

## 2. Architecture: Modular Engine Pattern

The system is organized into engines, each with a community and premium implementation. Engines communicate through well-defined Go interfaces. Premium code compiles only with the `pro` build tag and is stripped from community distributions at the filesystem level (CI deletes `internal/premium/`) and compiler level (build tags).

```
┌─────────────────────────────────────────────────────────┐
│                    Gin HTTP Layer                        │
│  (Routes, Middleware: Auth, RBAC, FeatureGate)          │
├─────────────┬───────────┬───────────┬───────────────────┤
│  Access     │    AI     │  Session  │   Notification    │
│  Engine     │  Engine   │  Engine   │   Engine          │
├─────────────┼───────────┼───────────┼───────────────────┤
│  Alert      │  Report   │  Query    │   Existing        │
│  Engine     │  Engine   │  (exists) │   Services        │
├─────────────┴───────────┴───────────┴───────────────────┤
│              Database Dialect Layer                      │
├─────────────────────────────────────────────────────────┤
│              Repository Layer (GORM)                     │
├─────────────────────────────────────────────────────────┤
│      PostgreSQL (metadata)  │  Target DBs (Pg/MySQL/+)  │
└─────────────────────────────────────────────────────────┘
```

### Directory Structure

```
internal/
├── api/
│   ├── handlers/            # existing + new handler files
│   ├── middleware/           # existing (auth, rbac, feature gate)
│   ├── registry.go          # PremiumService interface
│   ├── provider_community.go
│   └── provider_pro.go
├── dialect/                  # NEW: database dialect abstraction
│   ├── dialect.go            # DatabaseDialect interface + registry
│   ├── postgres.go           # PostgresDialect
│   ├── mysql.go              # MySQLDialect
│   └── (future: mssql.go, oracle.go, etc.)
├── engine/                   # NEW: engine interfaces
│   ├── access.go
│   ├── ai.go
│   ├── session.go
│   ├── alert.go
│   ├── report.go
│   └── notification.go
├── community/                # NEW: community engine implementations
│   ├── access/
│   ├── ai/
│   └── notification/
├── premium/                  # existing folder (empty in community)
│   ├── access/               # time-based access extensions
│   ├── ai/                   # platform tokens + voice
│   ├── session/              # ephemeral + vault-style creds
│   ├── alert/                # rule engine + AI anomaly detection
│   ├── report/               # scheduled reports + AI insights
│   └── notification/         # email + webhook channels
├── models/                   # existing + new models
├── repository/               # existing + new repos
├── service/                  # existing services
└── utils/                    # existing utilities
```

## 3. Database Dialect Layer

Replaces the current scattered `if/else` dialect handling (across `db_user_provisioning_service.go`, `query_service.go`, `monitoring_service.go`, `sync_service.go`) with a single interface.

### Interface

```go
type DatabaseDialect interface {
    Type() string
    DriverName() string
    BuildDSN(instance *models.DBInstance, dbName string) string
    BuildAdminDSN(instance *models.DBInstance) string

    // User provisioning
    CreateUserSQL(username, password string) string
    DropUserSQL(username string) string

    // Permission management
    GrantTableSQL(username, database, schema, table string, privileges []string) string
    GrantColumnSQL(username, database, schema, table string, columns []string, privileges []string) string
    RevokeTableSQL(username, database, schema, table string, privileges []string) string
    RevokeAllSQL(username string) string

    // Metadata scraping (absorbs existing DatabaseScraper)
    FetchDatabases(db *sql.DB, instanceID uuid.UUID) ([]string, error)
    FetchTables(db *sql.DB, instanceID uuid.UUID, dbName string) ([]models.DBTable, error)
    FetchColumns(db *sql.DB, tableID uuid.UUID, schema, table string) ([]models.DBColumn, error)
    FetchEntities(db *sql.DB, instanceID uuid.UUID, dbName string) ([]models.DBEntity, error)
    FetchPrivileges(db *sql.DB, instanceID uuid.UUID, dbName string) ([]models.DBPrivilege, error)
    FetchRoleMemberships(db *sql.DB, instanceID uuid.UUID) ([]models.DBRoleMembership, error)

    // Monitoring
    HealthCheckSQL() string
    FetchMetrics(db *sql.DB) ([]byte, error)
}
```

### Registry

```go
var dialects = map[string]DatabaseDialect{
    "postgres": &PostgresDialect{},
    "mysql":    &MySQLDialect{},
}

func GetDialect(dbType string) (DatabaseDialect, error) {
    d, ok := dialects[dbType]
    if !ok {
        return nil, fmt.Errorf("unsupported database type: %s", dbType)
    }
    return d, nil
}
```

### Adding a New Database

Create one file (e.g., `internal/dialect/mssql.go`), implement all interface methods, register in the map. No other files need changes.

## 4. Access Engine

### Two Permission Layers

| Layer | Purpose | Enforcement |
|---|---|---|
| **System RBAC** | Platform actions (manage users, view logs, manage instances) | `rbac.go` + JWT claims + `CheckPermission` middleware |
| **Data Access** | What data a user can query (instances/databases/tables/columns) | Extended `Permission` model + Access Engine + DB-level grants |

### Extended Permission Model

The existing `Permission` model is extended (not replaced) with additional granularity:

```go
type Permission struct {
    Base
    RoleID     *uuid.UUID     `gorm:"index" json:"roleId,omitempty"`
    UserID     *uuid.UUID     `gorm:"index" json:"userId,omitempty"`
    InstanceID uuid.UUID      `gorm:"index;not null" json:"instanceId"`
    Database   string         `gorm:"not null" json:"database"`
    Schema     string         `json:"schema,omitempty"`
    Table      string         `gorm:"not null" json:"table"`
    Column     string         `json:"column,omitempty"`
    Privileges pq.StringArray `gorm:"type:text[]" json:"privileges"`
    Type       string         `gorm:"default:'permanent'" json:"type"`
    ExpiresAt  *time.Time     `json:"expiresAt,omitempty"`
    GrantedBy  uuid.UUID      `gorm:"type:uuid" json:"grantedBy"`
    // Premium: time-window access
    ScheduleCron *string      `json:"scheduleCron,omitempty"`
}
```

New fields: `InstanceID`, `Schema`, `Column`, `GrantedBy`, `ScheduleCron`.

### Access Control Flow (3 Layers, Defense in Depth)

**Layer 1 — Database-Level Column Grants (security enforcement, 100% accurate):**

When permissions are created/modified, the provisioning service syncs grants to the target database:

- Instance/database/table-level → `GRANT ... ON table TO user` via dialect
- Column-level → `GRANT SELECT (col1, col2) ON table TO user` via dialect

The database itself rejects any query that touches unauthorized columns, regardless of query complexity (JOINs, CTEs, subqueries, etc.).

**Layer 2 — Application Pre-Check (UX, best-effort ~80-90%):**

Before forwarding to the database, a lightweight SQL parser extracts referenced tables/columns and checks against `Permission` entries. Catches common violations with friendly error messages ("No READ on orders.salary"). For complex queries the parser cannot fully analyze, defers to Layer 1.

**Layer 3 — Premium Query Rewriting (convenience, optional):**

Automatically rewrites queries to only include allowed columns (e.g., `SELECT *` → `SELECT col1, col2, col3`). Best-effort; falls back to Layer 1 if rewrite is too complex.

### Permission Sync Flow

```
Permission CRUD API called
  → Save to metadata DB
  → If table-level or above: call dialect.GrantTableSQL() on target DB
  → If column-level: call dialect.GrantColumnSQL() on target DB
  → If column-only change: no DB-level change (app layer + existing grants handle it)
```

### Premium: Time-Based Access

- `ExpiresAt` — grant auto-revokes after timestamp
- `ScheduleCron` — recurring access windows (e.g., "business hours only")
- Background worker checks expired/scheduled grants and runs REVOKE via dialect

### Existing Code Impact

| Component | Change |
|---|---|
| `Permission` model | Add fields: `InstanceID`, `Schema`, `Column`, `GrantedBy`, `ScheduleCron` |
| `db_user_provisioning_service.go` | Refactor to use dialect interface; add column-level grant support |
| `role_handler.go` / `role_service.go` | Keep as-is (system RBAC unchanged) |
| `user_handler.go` / `user_service.go` | Keep as-is |
| `check_permission.go` middleware | Keep as-is (system RBAC) |
| Query execution pipeline | Add Access Engine pre-check before execution |

## 5. AI Engine

### Community: BYOK Text-Based Query & Executor

**Flow:**

```
User prompt → AI Engine
  → Schema Context Builder (filters by user's access grants)
  → LLM Provider (pluggable interface)
  → Generated SQL
  → Safety Gate (configurable per-instance)
  → Access Engine check
  → Query Executor (existing)
  → Results
```

### AI Provider Interface

```go
type AIProvider interface {
    GenerateSQL(ctx context.Context, prompt string, schema SchemaContext) (string, error)
    ClassifyIntent(ctx context.Context, prompt string) (Intent, error)
    ExplainQuery(ctx context.Context, query string) (string, error)
}
```

Supports any OpenAI-compatible API endpoint. Community users bring their own keys; premium users get platform-provided tokens.

### Schema Context Builder

Pulls table/column metadata from `DBTable`/`DBColumn` models, filtered by the user's `Permission` entries. The LLM only "sees" tables and columns the user is authorized to access.

### Safety Gate

Per-instance configurable policies for destructive operations:

```go
type AIExecutionPolicy struct {
    Base
    InstanceID      uuid.UUID      `gorm:"index;not null"`
    ActionType      string         // "SELECT", "INSERT", "UPDATE", "DELETE", "DDL", "USER_MGMT"
    RequireApproval bool           `gorm:"default:true"`
    AllowedRoles    pq.StringArray `gorm:"type:text[]"`
}
```

Admins configure which operation types auto-execute vs. require approval, per instance. E.g., auto-execute SELECTs on dev, require approval for everything on production.

### BYOK Key Storage

```go
type UserAIConfig struct {
    Base
    UserID       uuid.UUID `gorm:"uniqueIndex;not null"`
    ProviderType string    // "openai", "anthropic", "custom"
    APIKey       string    // AES-256 encrypted (reuses existing crypto utility)
    BaseURL      *string   // for custom OpenAI-compatible endpoints
    ModelName    string
}
```

### Premium Additions

- **Platform-provided tokens** — system uses its own keys, no `UserAIConfig` needed
- **Voice prompting** — speech-to-text → text prompt → same AI Engine pipeline
- **Enhanced context** — query history analysis for smarter suggestions

## 6. Session Engine (Premium Only)

Two credential modes with complete audit traceability.

### Mode 1: Ephemeral DB Users

```
User starts session
  → Create DB user: sdb_eph_{userID}_{timestamp}
  → Grant permissions matching user's Permission entries (via dialect)
  → User executes queries through ephemeral user
  → Session ends (user action or TTL expiry)
  → DROP USER via dialect
  → CredentialSession record links ephemeral user → SessionDB user + time window
```

### Mode 2: Vault-Style Leases

```
User requests credentials
  → Check out DB credentials from pool
  → Record lease: (UserID, CredentialID, CheckedOutAt, TTL)
  → User executes queries with leased credentials
  → TTL expires → revoke lease, rotate password
  → Audit trail: every lease mapped to exact user + time range
```

### New Models

```go
type CredentialSession struct {
    Base
    UserID       uuid.UUID  `gorm:"index;not null"`
    InstanceID   uuid.UUID  `gorm:"index;not null"`
    SessionType  string     // "ephemeral", "lease"
    DBUsername   string
    CredentialID *uuid.UUID // for lease mode
    StartedAt    time.Time
    ExpiresAt    time.Time
    EndedAt      *time.Time
    Status       string     // "active", "expired", "revoked"
}
```

### Audit Traceability

The existing `AuditLog` model has `SessionID` and `DBUsername` fields. These are populated automatically during query execution, creating the complete trace: "User X had DB user Y from time A to time B, and ran these queries during that window."

## 7. Alert Engine (Premium Only)

Fully custom rule engine where users define their own rules on any metric/event the system captures.

### Models

```go
type AlertRule struct {
    Base
    TenantID    uuid.UUID `gorm:"index;not null"`
    CreatedBy   uuid.UUID
    Name        string
    Description string
    EventSource string    // "query", "access", "instance", "session", "audit"
    Condition   []byte    `gorm:"type:jsonb"` // structured rule definition
    Severity    string    // "info", "warning", "critical"
    IsEnabled   bool      `gorm:"default:true"`
    Channels    []byte    `gorm:"type:jsonb"` // notification channel config
}

type AlertEvent struct {
    Base
    RuleID      *uuid.UUID `gorm:"index"` // nil for AI-generated alerts
    TenantID    uuid.UUID  `gorm:"index;not null"`
    Severity    string
    Title       string
    Description string
    Metadata    []byte     `gorm:"type:jsonb"`
    Status      string     // "open", "acknowledged", "resolved"
    Source      string     // "rule", "ai"
}
```

### Rule Condition Format

```json
{
  "type": "threshold",
  "metric": "query.duration_ms",
  "operator": "gt",
  "value": 5000,
  "window": "5m",
  "min_occurrences": 3
}
```

### AI Component

Runs anomaly detection on captured metrics (query patterns, access patterns, performance). Generates `AlertEvent` entries with `Source: "ai"` when unusual behavior is detected.

## 8. Report Engine (Premium Only)

Scheduled custom reports where users define data sources, filters, schedules, and delivery channels.

### Models

```go
type ReportDefinition struct {
    Base
    TenantID         uuid.UUID `gorm:"index;not null"`
    CreatedBy        uuid.UUID
    Name             string
    Description      string
    DataSources      []byte    `gorm:"type:jsonb"` // which metrics/logs to include
    Filters          []byte    `gorm:"type:jsonb"` // date ranges, users, instances
    Schedule         string    // cron expression
    DeliveryChannels []byte    `gorm:"type:jsonb"` // email, webhook, dashboard
    Format           string    // "pdf", "csv", "json", "dashboard"
    IsEnabled        bool      `gorm:"default:true"`
    LastRunAt        *time.Time
}

type ReportExecution struct {
    Base
    DefinitionID uuid.UUID `gorm:"index;not null"`
    Status       string    // "running", "completed", "failed"
    StartedAt    time.Time
    CompletedAt  *time.Time
    ResultURL    *string
    Error        *string
}
```

### Background Worker

A cron-based worker picks up due `ReportDefinition` entries, executes data queries, generates reports in the requested format, and delivers via configured notification channels.

### AI Component

Can auto-generate report definitions from prompts ("weekly summary of PII table access") and add AI-written insights/summaries to report outputs.

## 9. Notification Engine

### Interface

```go
type NotificationChannel interface {
    Send(ctx context.Context, message NotificationMessage) error
    Type() string
}
```

### Community

Ships with WebSocket notifications (existing `notification_hub.go`).

### Premium

Adds pluggable channels:

```go
type NotificationConfig struct {
    Base
    TenantID    uuid.UUID `gorm:"index;not null"`
    ChannelType string    // "email", "webhook", "slack" (extensible)
    Config      []byte    `gorm:"type:jsonb"` // SMTP settings, webhook URL, etc.
    IsEnabled   bool      `gorm:"default:true"`
}
```

Ships with email + webhook. Users can extend by implementing the `NotificationChannel` interface.

## 10. Implementation Priority

Recommended build order based on dependencies:

| Phase | What | Why First |
|---|---|---|
| **Phase 1** | Database Dialect Layer | Foundation — unblocks all other engines, cleans up existing code |
| **Phase 2** | Access Engine (community) | Core value prop — column-to-instance access control |
| **Phase 3** | AI Engine (community BYOK) | Key differentiator — text-based query generation + executor |
| **Phase 4** | Session Engine (premium) | Builds on Access Engine + Dialect Layer |
| **Phase 5** | Alert Engine (premium) | Builds on audit logging infrastructure |
| **Phase 6** | Report Engine (premium) | Builds on audit + alert data |
| **Phase 7** | Notification Engine (premium channels) | Consumed by Alert + Report engines |
| **Phase 8** | Voice prompting (premium) | Extension of AI Engine |

## 11. Key Design Decisions

1. **Engine interfaces in `internal/engine/`** expose no premium logic — they are contracts (method signatures) only.
2. **Premium code protection** is three-fold: Go build tags, `.gitignore.community`, and CI pipeline deletion.
3. **Column-level security** relies on database-level GRANT enforcement (100% accurate) with application-level pre-check for UX.
4. **AI provider architecture** is pluggable via a common interface supporting any OpenAI-compatible endpoint.
5. **Database extensibility** is handled by the Dialect interface — one file per database type, zero changes elsewhere.
6. **Session credential traceability** maps every ephemeral/leased DB user back to the SessionDB user via `CredentialSession` records linked to `AuditLog` entries.
