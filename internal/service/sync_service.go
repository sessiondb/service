package service

import (
	"database/sql"
	"fmt"
	"log"
	"sessiondb/internal/models"
	"sessiondb/internal/repository"
	"sessiondb/internal/utils"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type DatabaseScraper interface {
	FetchTables(db *sql.DB, instanceID uuid.UUID, dbName string) ([]models.DBTable, error)
	FetchColumns(db *sql.DB, tableID uuid.UUID, schema, table string) ([]models.DBColumn, error)
	FetchEntities(db *sql.DB, instanceID uuid.UUID, dbName string) ([]models.DBEntity, error)
	FetchPrivileges(db *sql.DB, instanceID uuid.UUID, dbName string) ([]models.DBPrivilege, error)
	FetchRoleMemberships(db *sql.DB, instanceID uuid.UUID) ([]models.DBRoleMembership, error)
	FetchDatabases(db *sql.DB) ([]string, error)
	GetDSN(instance *models.DBInstance, dbName string) string
}

type SyncService struct {
	InstanceRepo *repository.InstanceRepository
	MetaRepo     *repository.MetadataRepository
	UpdateChan   chan SyncProgress
}

type SyncProgress struct {
	InstanceID uuid.UUID `json:"instanceId"`
	Step       string    `json:"step"`
	Status     string    `json:"status"`
	Percentage int       `json:"percentage"`
	Message    string    `json:"message"`
}

func NewSyncService(instRepo *repository.InstanceRepository, metaRepo *repository.MetadataRepository) *SyncService {
	return &SyncService{
		InstanceRepo: instRepo,
		MetaRepo:     metaRepo,
		UpdateChan:   make(chan SyncProgress, 100),
	}
}

func (s *SyncService) SyncInstance(id uuid.UUID, targetDB string) error {
	instance, err := s.InstanceRepo.FindByID(id)
	if err != nil {
		return err
	}

	s.broadcast(id, "Initialization", "processing", 10, "Connecting to database...")

	// 1. Get Scraper and Connect for Discovery
	var scraper DatabaseScraper
	switch instance.Type {
	case "mysql":
		scraper = &MySQLScraper{}
	case "postgres":
		scraper = &PostgresScraper{}
	default:
		s.broadcast(id, "Initialization", "error", 10, "Unsupported database type: "+instance.Type)
		return fmt.Errorf("unsupported database type: %s", instance.Type)
	}

	// Connect to default DB for discovery
	rootDsn := scraper.GetDSN(instance, "")
	rootDb, err := sql.Open(instance.Type, rootDsn)
	if err != nil {
		s.broadcast(id, "Initialization", "error", 10, "Connection failed: "+err.Error())
		return err
	}
	defer rootDb.Close()

	if err := rootDb.Ping(); err != nil {
		s.broadcast(id, "Initialization", "error", 10, "Ping failed: "+err.Error())
		return err
	}

	var dbsToSync []string
	if targetDB != "" {
		dbsToSync = []string{targetDB}
	} else {
		s.broadcast(id, "Discovery", "processing", 15, "Discovering databases...")
		dbsToSync, err = scraper.FetchDatabases(rootDb)
		if err != nil {
			s.broadcast(id, "Discovery", "error", 15, "Discovery failed: "+err.Error())
			return err
		}
	}

	s.broadcast(id, "Cleanup", "processing", 20, "Clearing old metadata...")
	if err := s.MetaRepo.ClearInstanceMetadata(id); err != nil {
		return err
	}

	// Fetch cluster-level entities (users/roles) once using the root connection.
	// pg_roles and mysql.user are cluster-wide, not per-database, so calling
	// this inside the per-DB loop would cause duplicate inserts.
	s.broadcast(id, "Users", "processing", 22, "Fetching database users and roles...")
	entities, err := scraper.FetchEntities(rootDb, id, "")
	if err != nil {
		s.broadcast(id, "Users", "error", 22, "Failed to fetch entities: "+err.Error())
		return err
	}
	log.Printf("[Sync %s] Fetched %d entities", id, len(entities))
	if err := s.MetaRepo.SaveEntities(entities); err != nil {
		s.broadcast(id, "Users", "error", 22, "Failed to save entities: "+err.Error())
		log.Printf("[Sync %s] SaveEntities error: %v", id, err)
		return err
	}
	log.Printf("[Sync %s] Saved %d entities successfully", id, len(entities))

	// Fetch and save role memberships
	s.broadcast(id, "Users", "processing", 23, "Fetching role hierarchies...")
	memberships, err := scraper.FetchRoleMemberships(rootDb, id)
	if err != nil {
		s.broadcast(id, "Users", "error", 23, "Failed to fetch role memberships: "+err.Error())
		return err
	}
	if err := s.MetaRepo.SaveRoleMemberships(memberships); err != nil {
		return err
	}

	for dbIdx, dbName := range dbsToSync {
		currentPercentage := 25 + (dbIdx * 75 / len(dbsToSync))
		s.broadcast(id, dbName, "processing", currentPercentage, "Starting sync for database: "+dbName)

		// Connect to specific DB
		dbDsn := scraper.GetDSN(instance, dbName)
		db, err := sql.Open(instance.Type, dbDsn)
		if err != nil {
			log.Printf("Failed to connect to %s: %v", dbName, err)
			continue
		}

		// 2. Fetch Tables
		tables, err := scraper.FetchTables(db, id, dbName)
		if err != nil {
			db.Close()
			return err
		}
		if err := s.MetaRepo.SaveTables(tables); err != nil {
			db.Close()
			return err
		}

		// 3. Fetch Columns
		for _, table := range tables {
			columns, err := scraper.FetchColumns(db, table.ID, table.Schema, table.Name)
			if err != nil {
				log.Printf("Error fetching columns for %s.%s: %v", table.Schema, table.Name, err)
				continue
			}
			if err := s.MetaRepo.SaveColumns(columns); err != nil {
				db.Close()
				return err
			}
		}

		// 4. Fetch Privileges
		privs, err := scraper.FetchPrivileges(db, id, dbName)
		if err != nil {
			db.Close()
			return err
		}
		if err := s.MetaRepo.SavePrivileges(privs); err != nil {
			db.Close()
			return err
		}
		db.Close()
	}

	// Finish
	now := time.Now()
	instance.LastSync = &now
	instance.Status = "online"
	s.InstanceRepo.Update(instance)

	s.broadcast(id, "Completion", "success", 100, "Sync completed successfully!")
	return nil
}

func (s *SyncService) broadcast(id uuid.UUID, step, status string, percentage int, msg string) {
	log.Printf("[Sync %s] %s: %s (%d%%) - %s", id, step, status, percentage, msg)
	s.UpdateChan <- SyncProgress{
		InstanceID: id,
		Step:       step,
		Status:     status,
		Percentage: percentage,
		Message:    msg,
	}
}

// --- Postgres Scraper ---

type PostgresScraper struct{}

func (ps *PostgresScraper) GetDSN(instance *models.DBInstance, dbName string) string {
	if dbName == "" {
		dbName = "postgres" // Default for discovery
	}
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
		instance.Host, instance.Username, instance.Password, dbName, instance.Port)
}

func (ps *PostgresScraper) FetchDatabases(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`
		SELECT datname FROM pg_database 
		WHERE datistemplate = false AND datname NOT IN ('postgres')`)
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

func (ps *PostgresScraper) FetchTables(db *sql.DB, instanceID uuid.UUID, dbName string) ([]models.DBTable, error) {
	rows, err := db.Query(`
		SELECT table_schema, table_name, table_type 
		FROM information_schema.tables 
		WHERE table_schema NOT IN ('pg_catalog', 'information_schema')`)
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

func (ps *PostgresScraper) FetchColumns(db *sql.DB, tableID uuid.UUID, schema, table string) ([]models.DBColumn, error) {
	rows, err := db.Query(`
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns 
		WHERE table_schema = $1 AND table_name = $2`, schema, table)
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

func (ps *PostgresScraper) FetchEntities(db *sql.DB, instanceID uuid.UUID, dbName string) ([]models.DBEntity, error) {
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

func (ps *PostgresScraper) FetchPrivileges(db *sql.DB, instanceID uuid.UUID, dbName string) ([]models.DBPrivilege, error) {
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
		FROM pg_roles
		WHERE rolsuper = true
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

func (ps *PostgresScraper) FetchRoleMemberships(db *sql.DB, instanceID uuid.UUID) ([]models.DBRoleMembership, error) {
	rows, err := db.Query(`
		SELECT r.rolname as role_name, m.rolname as member_name
		FROM pg_auth_members am
		JOIN pg_roles r ON am.roleid = r.oid
		JOIN pg_roles m ON am.member = m.oid`)
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

// --- MySQL Scraper ---

type MySQLScraper struct{}

func (ms *MySQLScraper) GetDSN(instance *models.DBInstance, dbName string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		instance.Username, instance.Password, instance.Host, instance.Port, dbName)
}

func (ms *MySQLScraper) FetchDatabases(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`
		SELECT SCHEMA_NAME FROM information_schema.SCHEMATA 
		WHERE SCHEMA_NAME NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')`)
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

func (ms *MySQLScraper) FetchTables(db *sql.DB, instanceID uuid.UUID, dbName string) ([]models.DBTable, error) {
	rows, err := db.Query(`
		SELECT TABLE_SCHEMA, TABLE_NAME, TABLE_TYPE 
		FROM information_schema.TABLES 
		WHERE TABLE_SCHEMA = ?`, dbName)
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

func (ms *MySQLScraper) FetchColumns(db *sql.DB, tableID uuid.UUID, schema, table string) ([]models.DBColumn, error) {
	rows, err := db.Query(`
		SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COLUMN_DEFAULT
		FROM information_schema.COLUMNS 
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?`, schema, table)
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

func (ms *MySQLScraper) FetchEntities(db *sql.DB, instanceID uuid.UUID, dbName string) ([]models.DBEntity, error) {
	// First, discover which users are actually roles by checking role_edges.
	// In MySQL 8.0+, roles are entries in mysql.user that appear as FROM_USER in role_edges.
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
	// role_edges may not exist on older MySQL — that's fine, roleSet stays empty.

	// MySQL users are global, not per-DB.
	rows, err := db.Query(`SELECT user, host FROM mysql.user`)
	if err != nil {
		return nil, nil // Fallback for limited permission users
	}
	defer rows.Close()

	var entities []models.DBEntity
	for rows.Next() {
		var e models.DBEntity
		var user, host string
		if err := rows.Scan(&user, &host); err != nil {
			return nil, err
		}

		fullGrantee := fmt.Sprintf("'%s'@'%s'", user, host)

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

func (ms *MySQLScraper) FetchPrivileges(db *sql.DB, instanceID uuid.UUID, dbName string) ([]models.DBPrivilege, error) {
	query := `
		SELECT GRANTEE, TABLE_SCHEMA, TABLE_NAME, PRIVILEGE_TYPE, IS_GRANTABLE
		FROM information_schema.TABLE_PRIVILEGES 
		WHERE TABLE_SCHEMA = ?
		UNION ALL
		SELECT GRANTEE, TABLE_SCHEMA, '*' AS TABLE_NAME, PRIVILEGE_TYPE, IS_GRANTABLE
		FROM information_schema.SCHEMA_PRIVILEGES
		WHERE TABLE_SCHEMA = ?
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

func (ms *MySQLScraper) FetchRoleMemberships(db *sql.DB, instanceID uuid.UUID) ([]models.DBRoleMembership, error) {
	// mysql.role_edges tracks role grants
	rows, err := db.Query(`SELECT FROM_USER, FROM_HOST, TO_USER, TO_HOST FROM mysql.role_edges`)
	if err != nil {
		// Table might not exist in older MySQL versions
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
		m.RoleName = fmt.Sprintf("'%s'@'%s'", fromUser, fromHost)
		m.MemberName = fmt.Sprintf("'%s'@'%s'", toUser, toHost)

		memberships = append(memberships, m)
	}
	return memberships, nil
}
