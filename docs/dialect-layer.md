# Database Dialect Layer

**Tech doc** — Architecture and usage of the dialect abstraction for multi-database support.

## Purpose

The dialect layer provides a single interface for all database-type-specific behavior: connection strings (DSN), user provisioning (CREATE/DROP USER), permission SQL (GRANT/REVOKE), metadata scraping (tables, columns, entities, privileges), and health checks. Adding a new database (e.g. MSSQL, Oracle) requires implementing one type and registering it; no changes are needed in callers.

## Location

- **Interface and registry:** `internal/dialect/dialect.go`
- **Implementations:** `internal/dialect/postgres.go`, `internal/dialect/mysql.go`
- **Tests:** `internal/dialect/dialect_test.go`

## Interface

```go
type DatabaseDialect interface {
    Type() string
    DriverName() string
    BuildDSN(instance *models.DBInstance, dbName string) string
    BuildAdminDSN(instance *models.DBInstance) string
    BuildDSNForUser(instance *models.DBInstance, dbName, username, password string) string
    CreateUserSQL(username, password string) string
    DropUserSQL(username string) string
    GrantTableSQL(username, database, schema, table string, privileges []string) string
    GrantColumnSQL(username, database, schema, table string, columns []string, privileges []string) string
    RevokeTableSQL(username, database, schema, table string, privileges []string) string
    RevokeAllSQL(username string) string
    FetchDatabases(db *sql.DB, instanceID uuid.UUID) ([]string, error)
    FetchTables(db *sql.DB, instanceID uuid.UUID, dbName string) ([]models.DBTable, error)
    FetchColumns(db *sql.DB, tableID uuid.UUID, schema, table string) ([]models.DBColumn, error)
    FetchEntities(db *sql.DB, instanceID uuid.UUID, dbName string) ([]models.DBEntity, error)
    FetchPrivileges(db *sql.DB, instanceID uuid.UUID, dbName string) ([]models.DBPrivilege, error)
    FetchRoleMemberships(db *sql.DB, instanceID uuid.UUID) ([]models.DBRoleMembership, error)
    HealthCheckSQL() string
    FetchMetrics(db *sql.DB) ([]byte, error)
}
```

## Usage

Callers use the registry only:

```go
d, err := dialect.GetDialect(instance.Type)
if err != nil {
    return err // unsupported database type
}
dsn := d.BuildAdminDSN(instance)
db, err := sql.Open(d.DriverName(), dsn)
// ... use d for SQL generation and metadata
```

## Consumers

- **SyncService** — Uses dialect for DSN, FetchDatabases, FetchTables, FetchColumns, FetchEntities, FetchPrivileges, FetchRoleMemberships.
- **DBUserProvisioningService** — Uses dialect for admin DSN, CreateUserSQL, DropUserSQL, GrantTableSQL, RevokeTableSQL, RevokeAllSQL.
- **QueryService** — Uses dialect for DriverName and BuildDSNForUser when executing queries with user or admin credentials.
- **MonitoringService** — Uses dialect for BuildAdminDSN and DriverName for health checks.

## Adding a New Database

1. Add a new file, e.g. `internal/dialect/mssql.go`.
2. Implement `DatabaseDialect` (all methods).
3. In `internal/dialect/dialect.go`, register: `dialects["mssql"] = &MSSQLDialect{}`.
4. Ensure the Go driver for that database is imported somewhere (e.g. in a service that opens connections) so `sql.Open(d.DriverName(), dsn)` works.

No changes are required in SyncService, DBUserProvisioningService, QueryService, or MonitoringService.
