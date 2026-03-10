// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package dialect

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sessiondb/internal/models"
	"sessiondb/internal/utils"
	"strings"

	"github.com/google/uuid"
)

type PostgresDialect struct{}

func (d *PostgresDialect) Type() string { return "postgres" }
func (d *PostgresDialect) DriverName() string { return "postgres" }

func (d *PostgresDialect) BuildDSN(instance *models.DBInstance, dbName string) string {
	if dbName == "" {
		dbName = "postgres"
	}
	return buildPostgresDSN(instance.Host, instance.Port, instance.Username, instance.Password, dbName)
}

func (d *PostgresDialect) BuildAdminDSN(instance *models.DBInstance) string {
	return buildPostgresDSN(instance.Host, instance.Port, instance.Username, instance.Password, "postgres")
}

func (d *PostgresDialect) BuildDSNForUser(instance *models.DBInstance, dbName, username, password string) string {
	if dbName == "" {
		dbName = "postgres"
	}
	return buildPostgresDSN(instance.Host, instance.Port, username, password, dbName)
}

func buildPostgresDSN(host string, port int, user, password, dbName string) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbName)
}

func (d *PostgresDialect) CreateUserSQL(username, password string) string {
	// Escape single quotes in password for safety
	password = strings.ReplaceAll(password, "'", "''")
	return "CREATE USER " + username + " WITH PASSWORD '" + password + "'"
}

func (d *PostgresDialect) DropUserSQL(username string) string {
	return "DROP USER " + username
}

func (d *PostgresDialect) GrantTableSQL(username, database, schema, table string, privileges []string) string {
	privStr := strings.Join(privileges, ", ")
	if schema == "" {
		schema = "public"
	}
	tbl := schema + "." + table
	if table == "*" {
		return "GRANT CONNECT ON DATABASE " + database + " TO " + username + "; GRANT " + privStr + " ON ALL TABLES IN SCHEMA " + schema + " TO " + username
	}
	return "GRANT " + privStr + " ON TABLE " + tbl + " TO " + username
}

func (d *PostgresDialect) GrantColumnSQL(username, database, schema, table string, columns []string, privileges []string) string {
	if len(columns) == 0 {
		return d.GrantTableSQL(username, database, schema, table, privileges)
	}
	if schema == "" {
		schema = "public"
	}
	tbl := schema + "." + table
	privStr := strings.Join(privileges, ", ")
	colList := strings.Join(columns, ", ")
	return "GRANT " + privStr + " (" + colList + ") ON TABLE " + tbl + " TO " + username
}

func (d *PostgresDialect) RevokeTableSQL(username, database, schema, table string, privileges []string) string {
	privStr := strings.Join(privileges, ", ")
	if schema == "" {
		schema = "public"
	}
	if table == "*" {
		return "REVOKE " + privStr + " ON ALL TABLES IN SCHEMA " + schema + " FROM " + username
	}
	return "REVOKE " + privStr + " ON TABLE " + schema + "." + table + " FROM " + username
}

func (d *PostgresDialect) RevokeAllSQL(username string) string {
	return "REVOKE ALL ON ALL TABLES IN SCHEMA public FROM " + username
}

func (d *PostgresDialect) FetchDatabases(db *sql.DB, _ uuid.UUID) ([]string, error) {
	rows, err := db.Query(`SELECT datname FROM pg_database WHERE datistemplate = false AND datname NOT IN ('postgres')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var dbs []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		dbs = append(dbs, name)
	}
	return dbs, nil
}

func (d *PostgresDialect) FetchTables(db *sql.DB, instanceID uuid.UUID, dbName string) ([]models.DBTable, error) {
	rows, err := db.Query(`SELECT table_schema, table_name, table_type FROM information_schema.tables WHERE table_schema NOT IN ('pg_catalog', 'information_schema')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tables []models.DBTable
	for rows.Next() {
		var t models.DBTable
		t.InstanceID = instanceID
		t.Database = dbName
		if err := rows.Scan(&t.Schema, &t.Name, &t.Type); err != nil {
			return nil, err
		}
		t.ID = uuid.New()
		tables = append(tables, t)
	}
	return tables, nil
}

func (d *PostgresDialect) FetchColumns(db *sql.DB, tableID uuid.UUID, schema, table string) ([]models.DBColumn, error) {
	rows, err := db.Query(`SELECT column_name, data_type, is_nullable, column_default FROM information_schema.columns WHERE table_schema = $1 AND table_name = $2`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cols []models.DBColumn
	for rows.Next() {
		var c models.DBColumn
		c.TableID = tableID
		var nullable string
		if err := rows.Scan(&c.Name, &c.DataType, &nullable, &c.DefaultValue); err != nil {
			return nil, err
		}
		c.IsNullable = nullable == "YES"
		cols = append(cols, c)
	}
	return cols, nil
}

func (d *PostgresDialect) FetchEntities(db *sql.DB, instanceID uuid.UUID, dbName string) ([]models.DBEntity, error) {
	rows, err := db.Query(`SELECT rolname, rolcanlogin FROM pg_roles`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entities []models.DBEntity
	for rows.Next() {
		var e models.DBEntity
		var canLogin bool
		var originalName string
		if err := rows.Scan(&originalName, &canLogin); err != nil {
			return nil, err
		}
		e.ID = uuid.New()
		e.InstanceID = instanceID
		e.Database = dbName
		e.DBKey = originalName
		if canLogin {
			e.Type = "USER"
			e.Name = originalName
		} else {
			e.Type = "ROLE"
			e.Name = utils.ToPascalCase(originalName)
		}
		entities = append(entities, e)
	}
	return entities, nil
}

func (d *PostgresDialect) FetchPrivileges(db *sql.DB, instanceID uuid.UUID, dbName string) ([]models.DBPrivilege, error) {
	query := `
		SELECT grantee, table_schema, table_name, privilege_type, is_grantable
		FROM information_schema.table_privileges
		WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
		UNION ALL
		SELECT grantee, object_schema AS table_schema, '*' AS table_name, privilege_type, is_grantable
		FROM information_schema.role_usage_grants
		WHERE object_schema NOT IN ('pg_catalog', 'information_schema')
		UNION ALL
		SELECT rolname AS grantee, '*' AS table_schema, '*' AS table_name, 'ALL' AS privilege_type, 'YES' AS is_grantable
		FROM pg_roles WHERE rolsuper = true
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var privs []models.DBPrivilege
	for rows.Next() {
		var p models.DBPrivilege
		var isGrantable string
		p.InstanceID = instanceID
		p.Database = dbName
		if err := rows.Scan(&p.Grantee, &p.Schema, &p.Table, &p.Privilege, &isGrantable); err != nil {
			return nil, err
		}
		p.IsGrantable = isGrantable == "YES"
		privs = append(privs, p)
	}
	return privs, nil
}

func (d *PostgresDialect) FetchRoleMemberships(db *sql.DB, instanceID uuid.UUID) ([]models.DBRoleMembership, error) {
	rows, err := db.Query(`SELECT r.rolname AS role_name, m.rolname AS member_name FROM pg_auth_members am JOIN pg_roles r ON am.roleid = r.oid JOIN pg_roles m ON am.member = m.oid`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var memberships []models.DBRoleMembership
	for rows.Next() {
		var m models.DBRoleMembership
		m.ID = uuid.New()
		m.InstanceID = instanceID
		if err := rows.Scan(&m.RoleName, &m.MemberName); err != nil {
			return nil, err
		}
		memberships = append(memberships, m)
	}
	return memberships, nil
}

func (d *PostgresDialect) HealthCheckSQL() string {
	return "SELECT 1"
}

func (d *PostgresDialect) FetchMetrics(db *sql.DB) ([]byte, error) {
	return json.Marshal(map[string]interface{}{})
}
