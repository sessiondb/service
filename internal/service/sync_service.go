// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package service

import (
	"database/sql"
	"log"
	"sessiondb/internal/dialect"
	"sessiondb/internal/repository"
	"time"

	"github.com/google/uuid"
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

func (s *SyncService) SyncInstance(id uuid.UUID, targetDB string) error {
	instance, err := s.InstanceRepo.FindByID(id)
	if err != nil {
		return err
	}

	s.broadcast(id, "Initialization", "processing", 10, "Connecting to database...")

	d, err := dialect.GetDialect(instance.Type)
	if err != nil {
		s.broadcast(id, "Initialization", "error", 10, err.Error())
		return err
	}

	rootDsn := d.BuildDSN(instance, "")
	rootDb, err := sql.Open(d.DriverName(), rootDsn)
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
		dbsToSync, err = d.FetchDatabases(rootDb, id)
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
	s.broadcast(id, "Users", "processing", 22, "Fetching database users and roles...")
	entities, err := d.FetchEntities(rootDb, id, "")
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

	s.broadcast(id, "Users", "processing", 23, "Fetching role hierarchies...")
	memberships, err := d.FetchRoleMemberships(rootDb, id)
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

		dbDsn := d.BuildDSN(instance, dbName)
		db, err := sql.Open(d.DriverName(), dbDsn)
		if err != nil {
			log.Printf("Failed to connect to %s: %v", dbName, err)
			continue
		}

		tables, err := d.FetchTables(db, id, dbName)
		if err != nil {
			db.Close()
			return err
		}
		if err := s.MetaRepo.SaveTables(tables); err != nil {
			db.Close()
			return err
		}

		for _, table := range tables {
			columns, err := d.FetchColumns(db, table.ID, table.Schema, table.Name)
			if err != nil {
				log.Printf("Error fetching columns for %s.%s: %v", table.Schema, table.Name, err)
				continue
			}
			if err := s.MetaRepo.SaveColumns(columns); err != nil {
				db.Close()
				return err
			}
		}

		privs, err := d.FetchPrivileges(db, id, dbName)
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

