package service

import (
	"sessiondb/internal/models"
	"sessiondb/internal/repository"
	"time"

	"github.com/google/uuid"
)

type InstanceService struct {
	Repo *repository.InstanceRepository
}

func NewInstanceService(repo *repository.InstanceRepository) *InstanceService {
	return &InstanceService{Repo: repo}
}

func (s *InstanceService) CreateInstance(name, host string, port int, instanceType, username, password string) (*models.DBInstance, error) {
	instance := &models.DBInstance{
		Name:     name,
		Host:     host,
		Port:     port,
		Type:     instanceType,
		Username: username,
		Password: password, // In production, encrypt this
		Status:   "offline",
	}
	if err := s.Repo.Create(instance); err != nil {
		return nil, err
	}
	return instance, nil
}

func (s *InstanceService) GetAllInstances() ([]models.DBInstance, error) {
	return s.Repo.FindAll()
}

func (s *InstanceService) GetInstanceByID(id uuid.UUID) (*models.DBInstance, error) {
	return s.Repo.FindByID(id)
}

func (s *InstanceService) UpdateInstance(id uuid.UUID, updates map[string]interface{}) (*models.DBInstance, error) {
	instance, err := s.Repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// Dynamic update based on map
	for k, v := range updates {
		switch k {
		case "name":
			instance.Name = v.(string)
		case "host":
			instance.Host = v.(string)
		case "port":
			instance.Port = int(v.(float64)) // JSON numbers are float64
		case "type":
			instance.Type = v.(string)
		case "username":
			instance.Username = v.(string)
		case "password":
			instance.Password = v.(string) // Encrypt in prod
		case "status":
			instance.Status = v.(string)
		}
	}

	if err := s.Repo.Update(instance); err != nil {
		return nil, err
	}
	return instance, nil
}

func (s *InstanceService) TriggerSync(id uuid.UUID) error {
	instance, err := s.Repo.FindByID(id)
	if err != nil {
		return err
	}

	// Mock sync start
	now := time.Now()
	instance.LastSync = &now
	instance.Status = "online"
	
	return s.Repo.Update(instance)
}
