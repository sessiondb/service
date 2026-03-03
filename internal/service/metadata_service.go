// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package service

import (
	"sessiondb/internal/models"
	"sessiondb/internal/repository"

	"github.com/google/uuid"
)

type MetadataService struct {
	Repo *repository.MetadataRepository
}

func NewMetadataService(repo *repository.MetadataRepository) *MetadataService {
	return &MetadataService{Repo: repo}
}

func (s *MetadataService) GetDatabases(instanceID uuid.UUID) ([]string, error) {
	return s.Repo.GetDatabases(instanceID)
}

func (s *MetadataService) GetTables(instanceID uuid.UUID, database string) ([]models.DBTable, error) {
	return s.Repo.GetTables(instanceID, database)
}

type DatabaseSchema struct {
	Database string           `json:"database"`
	Tables   []models.DBTable `json:"tables"`
}

type InstanceSchemaResponse struct {
	InstanceID uuid.UUID        `json:"instanceId"`
	Databases  []DatabaseSchema `json:"databases"`
}

func (s *MetadataService) GetInstanceSchema(instanceID uuid.UUID) (*InstanceSchemaResponse, error) {
	tables, err := s.Repo.GetFullSchema(instanceID)
	if err != nil {
		return nil, err
	}

	// Group tables by database
	dbMap := make(map[string][]models.DBTable)
	for _, t := range tables {
		dbMap[t.Database] = append(dbMap[t.Database], t)
	}

	databases := make([]DatabaseSchema, 0, len(dbMap))
	for dbName, dbTables := range dbMap {
		databases = append(databases, DatabaseSchema{
			Database: dbName,
			Tables:   dbTables,
		})
	}

	return &InstanceSchemaResponse{
		InstanceID: instanceID,
		Databases:  databases,
	}, nil
}
func (s *MetadataService) GetTableDetails(tableID uuid.UUID) (*models.DBTable, error) {
	return s.Repo.GetTableByID(tableID)
}
