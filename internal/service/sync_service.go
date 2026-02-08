package service

import (
	"database/sql"
	"fmt"
	"log"
	"sessiondb/internal/models"
	"sessiondb/internal/repository"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

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

func (s *SyncService) SyncInstance(id uuid.UUID) error {
	instance, err := s.InstanceRepo.FindByID(id)
	if err != nil {
		return err
	}

	s.broadcast(id, "Initialization", "processing", 10, "Connecting to database...")

	// 1. Connect to target DB
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
		instance.Host, instance.Username, instance.Password, instance.Name, instance.Port)
	
	db, err := sql.Open(instance.Type, dsn)
	if err != nil {
		s.broadcast(id, "Initialization", "error", 10, "Connection failed: "+err.Error())
		return err
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		s.broadcast(id, "Initialization", "error", 10, "Ping failed: "+err.Error())
		return err
	}

	s.broadcast(id, "Cleanup", "processing", 20, "Clearing old metadata...")
	if err := s.MetaRepo.ClearInstanceMetadata(id); err != nil {
		return err
	}

	// 2. Fetch Tables
	s.broadcast(id, "Tables", "processing", 30, "Fetching tables and schemas...")
	tables, err := s.fetchTables(db, id)
	if err != nil {
		return err
	}
	if err := s.MetaRepo.SaveTables(tables); err != nil {
		return err
	}

	// 3. Fetch Columns
	s.broadcast(id, "Columns", "processing", 50, "Fetching column metadata...")
	for i, table := range tables {
		columns, err := s.fetchColumns(db, table.ID, table.Schema, table.Name)
		if err != nil {
			log.Printf("Error fetching columns for %s.%s: %v", table.Schema, table.Name, err)
			continue
		}
		if err := s.MetaRepo.SaveColumns(columns); err != nil {
			return err
		}
		if i%5 == 0 {
			prog := 50 + (i * 20 / len(tables))
			s.broadcast(id, "Columns", "processing", prog, fmt.Sprintf("Processing table %d/%d", i+1, len(tables)))
		}
	}

	// 4. Fetch Roles/Users
	s.broadcast(id, "Entities", "processing", 80, "Fetching roles and users...")
	entities, err := s.fetchEntities(db, id)
	if err != nil {
		return err
	}
	if err := s.MetaRepo.SaveEntities(entities); err != nil {
		return err
	}

	// 5. Fetch Privileges
	s.broadcast(id, "Privileges", "processing", 90, "Fetching privileges...")
	privs, err := s.fetchPrivileges(db, id)
	if err != nil {
		return err
	}
	if err := s.MetaRepo.SavePrivileges(privs); err != nil {
		return err
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
	s.UpdateChan <- SyncProgress{
		InstanceID: id,
		Step:       step,
		Status:     status,
		Percentage: percentage,
		Message:    msg,
	}
}

// Target-specific fetchers (Postgres as default)

func (s *SyncService) fetchTables(db *sql.DB, instanceID uuid.UUID) ([]models.DBTable, error) {
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
		if err := rows.Scan(&t.Schema, &t.Name, &t.Type); err != nil {
			return nil, err
		}
		t.ID = uuid.New() // Generate ID now so we can use it for columns
		tables = append(tables, t)
	}
	return tables, nil
}

func (s *SyncService) fetchColumns(db *sql.DB, tableID uuid.UUID, schema, table string) ([]models.DBColumn, error) {
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

func (s *SyncService) fetchEntities(db *sql.DB, instanceID uuid.UUID) ([]models.DBEntity, error) {
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
		if err := rows.Scan(&e.Name, &canLogin); err != nil {
			return nil, err
		}
		if canLogin { e.Type = "USER" } else { e.Type = "ROLE" }
		entities = append(entities, e)
	}
	return entities, nil
}

func (s *SyncService) fetchPrivileges(db *sql.DB, instanceID uuid.UUID) ([]models.DBPrivilege, error) {
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
		if err := rows.Scan(&p.Grantee, &p.Schema, &p.Table, &p.Privilege); err != nil {
			return nil, err
		}
		privs = append(privs, p)
	}
	return privs, nil
}
