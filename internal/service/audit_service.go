package service

import (
	"sessiondb/internal/models"
	"sessiondb/internal/repository"
	"time"

	"github.com/google/uuid"
)

type AuditService struct {
	AuditRepo *repository.AuditRepository
}

func NewAuditService(auditRepo *repository.AuditRepository) *AuditService {
	return &AuditService{AuditRepo: auditRepo}
}

func (s *AuditService) LogAction(userID uuid.UUID, action, resource, resourceType, status, errorMessage string) {
	log := &models.AuditLog{
		Timestamp:    time.Now(),
		UserID:       userID,
		Action:       action,
		Resource:     resource,
		ResourceType: resourceType,
		Status:       status,
		ErrorMessage: errorMessage,
	}
	// Fire and forget
	go s.AuditRepo.Log(log)
}

func (s *AuditService) LogQuery(userID, instanceID uuid.UUID, instanceName, query, dbName, status, errMsg, ipAddress, userAgent string, durationMs int64) {
	log := &models.AuditLog{
		Timestamp:    time.Now(),
		UserID:       userID,
		Action:       "QUERY_EXECUTE",
		Resource:     "database",
		ResourceType: "query",
		InstanceID:   &instanceID,
		InstanceName: instanceName,
		Database:     dbName,
		Query:        query,
		Status:       status,
		ErrorMessage: errMsg,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		DurationMs:   durationMs,
	}
	go s.AuditRepo.Log(log)
}

func (s *AuditService) Create(log *models.AuditLog) error {
	return s.AuditRepo.Log(log)
}

func (s *AuditService) GetLogs() ([]models.AuditLog, error) {
	return s.AuditRepo.FindAll()
}

func (s *AuditService) GetQueryHistory(userID uuid.UUID) ([]models.AuditLog, error) {
	return s.AuditRepo.FindByUserAndAction(userID, "QUERY_EXECUTE")
}
