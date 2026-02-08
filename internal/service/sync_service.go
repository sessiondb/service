package service

import (
	"database/sql"
	"fmt"
	"log"
	"sessiondb/internal/models"
	"sessiondb/internal/repository"
	"time"

	"github.com/google/uuid"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

type DatabaseScraper interface {
	FetchTables(db *sql.DB, instanceID uuid.UUID, dbName string) ([]models.DBTable, error)
	FetchColumns(db *sql.DB, tableID uuid.UUID, schema, table string) ([]models.DBColumn, error)
	FetchEntities(db *sql.DB, instanceID uuid.UUID, dbName string) ([]models.DBEntity, error)
	FetchPrivileges(db *sql.DB, instanceID uuid.UUID, dbName string) ([]models.DBPrivilege, error)
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

	for dbIdx, dbName := range dbsToSync {
		currentPercentage := 20 + (dbIdx * 80 / len(dbsToSync))
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

		// 4. Fetch Roles/Users
		entities, err := scraper.FetchEntities(db, id, dbName)
		if err != nil {
			db.Close()
			return err
		}
		if err := s.MetaRepo.SaveEntities(entities); err != nil {
			db.Close()
			return err
		}

		// 5. Fetch Privileges
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
		e.InstanceID = instanceID
		e.Database = dbName
		if err := rows.Scan(&e.Name, &canLogin); err != nil {
			return nil, err
		}
		if canLogin { e.Type = "USER" } else { e.Type = "ROLE" }
		entities = append(entities, e)
	}
	return entities, nil
}

func (ps *PostgresScraper) FetchPrivileges(db *sql.DB, instanceID uuid.UUID, dbName string) ([]models.DBPrivilege, error) {
	rows, err := db.Query(`
		SELECT grantee, table_schema, table_name, privilege_type 
		FROM information_schema.table_privileges 
		WHERE table_schema NOT IN ('pg_catalog', 'information_schema')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var privs []models.DBPrivilege
	for rows.Next() {
		var p models.DBPrivilege
		p.InstanceID = instanceID
		p.Database = dbName
		if err := rows.Scan(&p.Grantee, &p.Schema, &p.Table, &p.Privilege); err != nil {
			return nil, err
		}
		privs = append(privs, p)
	}
	return privs, nil
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
	// MySQL users are global, not per-DB. But for cataloging, we'll associate them.
	rows, err := db.Query(`SELECT user, 'USER' FROM mysql.user`)
	if err != nil {
		return nil, nil
	}
	defer rows.Close()

	var entities []models.DBEntity
	for rows.Next() {
		var e models.DBEntity
		e.InstanceID = instanceID
		e.Database = dbName
		if err := rows.Scan(&e.Name, &e.Type); err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}
	return entities, nil
}

func (ms *MySQLScraper) FetchPrivileges(db *sql.DB, instanceID uuid.UUID, dbName string) ([]models.DBPrivilege, error) {
	rows, err := db.Query(`
		SELECT GRANTEE, TABLE_SCHEMA, TABLE_NAME, PRIVILEGE_TYPE 
		FROM information_schema.TABLE_PRIVILEGES 
		WHERE TABLE_SCHEMA = ?`, dbName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var privs []models.DBPrivilege
	for rows.Next() {
		var p models.DBPrivilege
		p.InstanceID = instanceID
		p.Database = dbName
		if err := rows.Scan(&p.Grantee, &p.Schema, &p.Table, &p.Privilege); err != nil {
			return nil, err
		}
		privs = append(privs, p)
	}
	return privs, nil
}
