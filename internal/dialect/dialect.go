// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package dialect

import (
	"database/sql"
	"fmt"
	"sessiondb/internal/models"

	"github.com/google/uuid"
)

// DatabaseDialect abstracts database-specific DSN building, user provisioning,
// permission SQL, and metadata scraping. Implement once per DB type (postgres, mysql, etc.).
type DatabaseDialect interface {
	Type() string
	DriverName() string

	// BuildDSN returns a DSN for connecting to a specific database using instance admin credentials. dbName empty = default discovery DB.
	BuildDSN(instance *models.DBInstance, dbName string) string
	// BuildAdminDSN returns a DSN for admin operations (user creation, grants). Uses admin credentials.
	BuildAdminDSN(instance *models.DBInstance) string
	// BuildDSNForGrant returns the DSN to use when executing GRANT for a table in targetDatabase (e.g. Postgres must connect to that DB; MySQL may use admin DSN).
	BuildDSNForGrant(instance *models.DBInstance, targetDatabase string) string
	// BuildDSNForUser returns a DSN for the given database using the provided username and password (e.g. for query execution with user creds).
	BuildDSNForUser(instance *models.DBInstance, dbName, username, password string) string

	// User provisioning
	CreateUserSQL(username, password string) string
	DropUserSQL(username string) string

	// Permission management. For GrantColumnSQL, columns empty means table-level grant.
	GrantTableSQL(username, database, schema, table string, privileges []string) string
	GrantColumnSQL(username, database, schema, table string, columns []string, privileges []string) string
	RevokeTableSQL(username, database, schema, table string, privileges []string) string
	RevokeAllSQL(username string) string

	// Metadata scraping
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

var dialects = map[string]DatabaseDialect{
	"postgres": &PostgresDialect{},
	"mysql":    &MySQLDialect{},
}

// GetDialect returns the dialect for the given database type, or an error if unsupported.
func GetDialect(dbType string) (DatabaseDialect, error) {
	d, ok := dialects[dbType]
	if !ok {
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
	return d, nil
}

// RegisterDialect registers a dialect for a type (e.g. for tests or future mssql).
func RegisterDialect(dbType string, d DatabaseDialect) {
	dialects[dbType] = d
}
