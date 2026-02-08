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

func (s *MetadataService) GetTableDetails(tableID uuid.UUID) (*models.DBTable, error) {
	return s.Repo.GetTableByID(tableID)
}
