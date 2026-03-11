// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package dialect

import (
	"strings"
	"sessiondb/internal/models"
	"testing"
)

func TestGetDialect_Supported(t *testing.T) {
	tests := []struct {
		dbType string
	}{
		{"postgres"},
		{"mysql"},
	}
	for _, tt := range tests {
		t.Run(tt.dbType, func(t *testing.T) {
			d, err := GetDialect(tt.dbType)
			if err != nil {
				t.Fatalf("GetDialect(%q) err = %v", tt.dbType, err)
			}
			if d.Type() != tt.dbType {
				t.Errorf("Type() = %q, want %q", d.Type(), tt.dbType)
			}
			if d.DriverName() == "" {
				t.Error("DriverName() is empty")
			}
		})
	}
}

func TestGetDialect_Unsupported(t *testing.T) {
	_, err := GetDialect("mssql")
	if err == nil {
		t.Fatal("GetDialect(\"mssql\") expected error")
	}
}

func TestPostgresDialect_SQL(t *testing.T) {
	d := &PostgresDialect{}
	inst := &models.DBInstance{Host: "localhost", Port: 5432, Username: "u", Password: "p"}

	if d.BuildDSN(inst, "") != "host=localhost port=5432 user=u password=p dbname=postgres sslmode=disable" {
		t.Errorf("BuildDSN(inst, \"\") wrong default dbname")
	}
	if d.BuildDSN(inst, "mydb") != "host=localhost port=5432 user=u password=p dbname=mydb sslmode=disable" {
		t.Errorf("BuildDSN(inst, \"mydb\") wrong")
	}
	if d.BuildAdminDSN(inst) != "host=localhost port=5432 user=u password=p dbname=postgres sslmode=disable" {
		t.Errorf("BuildAdminDSN wrong")
	}

	create := d.CreateUserSQL("jane", "secret")
	if create != "CREATE USER jane WITH PASSWORD 'secret'" {
		t.Errorf("CreateUserSQL = %q", create)
	}
	drop := d.DropUserSQL("jane")
	if drop != "DROP USER jane" {
		t.Errorf("DropUserSQL = %q", drop)
	}

	grantT := d.GrantTableSQL("jane", "mydb", "public", "orders", []string{"SELECT", "INSERT"})
	if grantT != "GRANT SELECT, INSERT ON TABLE public.orders TO jane" {
		t.Errorf("GrantTableSQL = %q", grantT)
	}
	grantC := d.GrantColumnSQL("jane", "mydb", "public", "orders", []string{"id", "name"}, []string{"SELECT"})
	if grantC != "GRANT SELECT (id, name) ON TABLE public.orders TO jane" {
		t.Errorf("GrantColumnSQL = %q", grantC)
	}
	revoke := d.RevokeTableSQL("jane", "mydb", "public", "orders", []string{"SELECT"})
	if revoke != "REVOKE SELECT ON TABLE public.orders FROM jane" {
		t.Errorf("RevokeTableSQL = %q", revoke)
	}
	revokeAll := d.RevokeAllSQL("jane")
	if revokeAll != "REVOKE ALL ON ALL TABLES IN SCHEMA public FROM jane" {
		t.Errorf("RevokeAllSQL = %q", revokeAll)
	}

	if d.HealthCheckSQL() != "SELECT 1" {
		t.Errorf("HealthCheckSQL = %q", d.HealthCheckSQL())
	}
	dsnUser := d.BuildDSNForUser(inst, "mydb", "jane", "secret")
	if dsnUser != "host=localhost port=5432 user=jane password=secret dbname=mydb sslmode=disable" {
		t.Errorf("BuildDSNForUser = %q", dsnUser)
	}
}

func TestMySQLDialect_SQL(t *testing.T) {
	d := &MySQLDialect{}
	inst := &models.DBInstance{Host: "localhost", Port: 3306, Username: "u", Password: "p"}

	dsn := d.BuildDSN(inst, "mydb")
	if dsn != "u:p@tcp(localhost:3306)/mydb?parseTime=true" {
		t.Errorf("BuildDSN(inst, \"mydb\") = %q", dsn)
	}
	if d.BuildAdminDSN(inst) != "u:p@tcp(localhost:3306)/mysql?parseTime=true" {
		t.Errorf("BuildAdminDSN wrong")
	}

	create := d.CreateUserSQL("jane", "secret")
	if create != "CREATE USER 'jane'@'%' IDENTIFIED BY 'secret'" {
		t.Errorf("CreateUserSQL = %q", create)
	}
	drop := d.DropUserSQL("jane")
	if drop != "DROP USER 'jane'@'%'" {
		t.Errorf("DropUserSQL = %q", drop)
	}

	grantT := d.GrantTableSQL("jane", "mydb", "", "orders", []string{"SELECT"})
	if grantT != "GRANT SELECT ON mydb.orders TO 'jane'@'%'" {
		t.Errorf("GrantTableSQL = %q", grantT)
	}
	revokeAll := d.RevokeAllSQL("jane")
	if revokeAll != "REVOKE ALL PRIVILEGES ON *.* FROM 'jane'@'%'" {
		t.Errorf("RevokeAllSQL = %q", revokeAll)
	}
}

// TestMySQLDialect_BuildDSNForUser_normalizeUsername ensures 'user'@'%' is normalized to short name for DSN.
func TestMySQLDialect_BuildDSNForUser_normalizeUsername(t *testing.T) {
	d := &MySQLDialect{}
	inst := &models.DBInstance{Host: "localhost", Port: 3306}

	// MySQL stores usernames as 'mouli'@'%'; DSN must use short name only
	dsn := d.BuildDSNForUser(inst, "", "'mouli'@'%'", "secret")
	if dsn != "mouli:secret@tcp(localhost:3306)/?parseTime=true" {
		t.Errorf("BuildDSNForUser with 'mouli'@'%%' = %q, want short username in DSN", dsn)
	}

	// Password with special chars is URL-encoded
	dsn2 := d.BuildDSNForUser(inst, "mydb", "mouli", "p@ss#word")
	if !contains(t, dsn2, "mouli:") || !contains(t, dsn2, "@tcp(localhost:3306)/mydb") {
		t.Errorf("BuildDSNForUser with special chars in password = %q", dsn2)
	}
}

func contains(t *testing.T, s, sub string) bool {
	t.Helper()
	return strings.Contains(s, sub)
}

func TestPostgresDialect_FetchDatabases_NoDB(t *testing.T) {
	// Cannot test against real DB in unit test; just ensure interface is satisfied
	var _ DatabaseDialect = (*PostgresDialect)(nil)
}

func TestRegisterDialect(t *testing.T) {
	fake := &PostgresDialect{}
	RegisterDialect("test_fake", fake)
	defer func() {
		delete(dialects, "test_fake")
	}()
	d, err := GetDialect("test_fake")
	if err != nil {
		t.Fatalf("GetDialect after Register: %v", err)
	}
	if d.Type() != "postgres" {
		t.Errorf("registered dialect Type = %q", d.Type())
	}
}

// Ensure dialect is usable with a nil instance for SQL-only tests (no DSN)
func TestDialect_SQLWithEmptyInstance(t *testing.T) {
	inst := &models.DBInstance{}
	_ = inst
	pg := &PostgresDialect{}
	_ = pg.GrantTableSQL("u", "db", "public", "t", []string{"SELECT"})
	my := &MySQLDialect{}
	_ = my.GrantTableSQL("u", "db", "", "t", []string{"SELECT"})
}
