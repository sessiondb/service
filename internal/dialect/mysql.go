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

const mysqlUserHostFmt = "'%s'@'%s'"

type MySQLDialect struct{}

func (d *MySQLDialect) Type() string       { return "mysql" }
func (d *MySQLDialect) DriverName() string { return "mysql" }

func (d *MySQLDialect) BuildDSN(instance *models.DBInstance, dbName string) string {
	base := fmt.Sprintf("%s:%s@tcp(%s:%d)/", instance.Username, instance.Password, instance.Host, instance.Port)
	if dbName != "" {
		return base + dbName + "?parseTime=true"
	}
	return base + "?parseTime=true"
}

func (d *MySQLDialect) BuildAdminDSN(instance *models.DBInstance) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/mysql?parseTime=true",
		instance.Username, instance.Password, instance.Host, instance.Port)
}

// BuildDSNForUser builds a DSN with the given username and password. User and password are
// URL-encoded so special characters (@ : / etc.) do not break the connection string.
func (d *MySQLDialect) BuildDSNForUser(instance *models.DBInstance, dbName, username, password string) string {
	validUser := strings.Split(username, "@")[0]
	validUser = strings.Trim(validUser, "'\"")
	base := fmt.Sprintf("%s:%s@tcp(%s:%d)/", validUser, password, instance.Host, instance.Port)
	if dbName != "" {
		return base + dbName + "?parseTime=true"
	}
	return base + "?parseTime=true"
}

func (d *MySQLDialect) CreateUserSQL(username, password string) string {
	password = strings.ReplaceAll(password, "'", "''")
	return fmt.Sprintf("CREATE USER '%s'@'%%' IDENTIFIED BY '%s'", username, password)
}

func (d *MySQLDialect) DropUserSQL(username string) string {
	// MySQL user may be stored as 'user'@'%'
	if strings.Contains(username, "@") {
		return fmt.Sprintf("DROP USER %s", username)
	}
	return fmt.Sprintf("DROP USER '%s'@'%%'", username)
}

func (d *MySQLDialect) GrantTableSQL(username, database, schema, table string, privileges []string) string {
	userPart := username
	if !strings.Contains(username, "@") {
		userPart = "'" + username + "'@'%'"
	}
	privStr := strings.Join(privileges, ", ")
	if table == "*" {
		return fmt.Sprintf("GRANT %s ON %s.* TO %s", privStr, database, userPart)
	}
	tbl := database + "." + table
	return fmt.Sprintf("GRANT %s ON %s TO %s", privStr, tbl, userPart)
}

func (d *MySQLDialect) GrantColumnSQL(username, database, schema, table string, columns []string, privileges []string) string {
	if len(columns) == 0 {
		return d.GrantTableSQL(username, database, schema, table, privileges)
	}
	userPart := username
	if !strings.Contains(username, "@") {
		userPart = "'" + username + "'@'%'"
	}
	privStr := strings.Join(privileges, ", ")
	colList := strings.Join(columns, ", ")
	tbl := database + "." + table
	return fmt.Sprintf("GRANT %s (%s) ON %s TO %s", privStr, colList, tbl, userPart)
}

func (d *MySQLDialect) RevokeTableSQL(username, database, schema, table string, privileges []string) string {
	userPart := username
	if !strings.Contains(username, "@") {
		userPart = "'" + username + "'@'%'"
	}
	privStr := strings.Join(privileges, ", ")
	if table == "*" {
		return fmt.Sprintf("REVOKE %s ON %s.* FROM %s", privStr, database, userPart)
	}
	tbl := database + "." + table
	return fmt.Sprintf("REVOKE %s ON %s FROM %s", privStr, tbl, userPart)
}

func (d *MySQLDialect) RevokeAllSQL(username string) string {
	userPart := username
	if !strings.Contains(username, "@") {
		userPart = "'" + username + "'@'%'"
	}
	return "REVOKE ALL PRIVILEGES ON *.* FROM " + userPart
}

func (d *MySQLDialect) FetchDatabases(db *sql.DB, _ uuid.UUID) ([]string, error) {
	rows, err := db.Query(`SELECT SCHEMA_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')`)
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

func (d *MySQLDialect) FetchTables(db *sql.DB, instanceID uuid.UUID, dbName string) ([]models.DBTable, error) {
	rows, err := db.Query(`SELECT TABLE_SCHEMA, TABLE_NAME, TABLE_TYPE FROM information_schema.TABLES WHERE TABLE_SCHEMA = ?`, dbName)
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

func (d *MySQLDialect) FetchColumns(db *sql.DB, tableID uuid.UUID, schema, table string) ([]models.DBColumn, error) {
	rows, err := db.Query(`SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COLUMN_DEFAULT FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?`, schema, table)
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

func (d *MySQLDialect) FetchEntities(db *sql.DB, instanceID uuid.UUID, dbName string) ([]models.DBEntity, error) {
	roleSet := make(map[string]bool)
	roleRows, err := db.Query(`SELECT DISTINCT FROM_USER FROM mysql.role_edges`)
	if err == nil {
		defer roleRows.Close()
		for roleRows.Next() {
			var roleName string
			if err := roleRows.Scan(&roleName); err == nil {
				roleSet[roleName] = true
			}
		}
	}
	rows, err := db.Query(`SELECT user, host FROM mysql.user`)
	if err != nil {
		return nil, nil
	}
	defer rows.Close()
	var entities []models.DBEntity
	for rows.Next() {
		var e models.DBEntity
		var user, host string
		if err := rows.Scan(&user, &host); err != nil {
			return nil, err
		}
		fullGrantee := fmt.Sprintf(mysqlUserHostFmt, user, host)
		e.ID = uuid.New()
		e.InstanceID = instanceID
		e.Database = dbName
		e.DBKey = fullGrantee
		if roleSet[user] {
			e.Type = "ROLE"
			e.Name = utils.ToPascalCase(user)
		} else {
			e.Type = "USER"
			e.Name = fullGrantee
		}
		entities = append(entities, e)
	}
	return entities, nil
}

func (d *MySQLDialect) FetchPrivileges(db *sql.DB, instanceID uuid.UUID, dbName string) ([]models.DBPrivilege, error) {
	query := `
		SELECT GRANTEE, TABLE_SCHEMA, TABLE_NAME, PRIVILEGE_TYPE, IS_GRANTABLE
		FROM information_schema.TABLE_PRIVILEGES WHERE TABLE_SCHEMA = ?
		UNION ALL
		SELECT GRANTEE, TABLE_SCHEMA, '*' AS TABLE_NAME, PRIVILEGE_TYPE, IS_GRANTABLE
		FROM information_schema.SCHEMA_PRIVILEGES WHERE TABLE_SCHEMA = ?
		UNION ALL
		SELECT GRANTEE, '*' AS TABLE_SCHEMA, '*' AS TABLE_NAME, PRIVILEGE_TYPE, IS_GRANTABLE
		FROM information_schema.USER_PRIVILEGES
	`
	rows, err := db.Query(query, dbName, dbName)
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

func (d *MySQLDialect) FetchRoleMemberships(db *sql.DB, instanceID uuid.UUID) ([]models.DBRoleMembership, error) {
	rows, err := db.Query(`SELECT FROM_USER, FROM_HOST, TO_USER, TO_HOST FROM mysql.role_edges`)
	if err != nil {
		return nil, nil
	}
	defer rows.Close()
	var memberships []models.DBRoleMembership
	for rows.Next() {
		var m models.DBRoleMembership
		var fromUser, fromHost, toUser, toHost string
		if err := rows.Scan(&fromUser, &fromHost, &toUser, &toHost); err != nil {
			return nil, err
		}
		m.ID = uuid.New()
		m.InstanceID = instanceID
		m.RoleName = fmt.Sprintf(mysqlUserHostFmt, fromUser, fromHost)
		m.MemberName = fmt.Sprintf(mysqlUserHostFmt, toUser, toHost)
		memberships = append(memberships, m)
	}
	return memberships, nil
}

func (d *MySQLDialect) HealthCheckSQL() string {
	return "SELECT 1"
}

func (d *MySQLDialect) FetchMetrics(db *sql.DB) ([]byte, error) {
	return json.Marshal(map[string]interface{}{})
}
