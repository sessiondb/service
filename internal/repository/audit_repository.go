package repository

import (
	"sessiondb/internal/models"

	"gorm.io/gorm"
)

type AuditRepository struct {
	DB *gorm.DB
}

func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{DB: db}
}

func (r *AuditRepository) Log(log *models.AuditLog) error {
	return r.DB.Create(log).Error
}

func (r *AuditRepository) FindAll() ([]models.AuditLog, error) {
	var logs = make([]models.AuditLog, 0)
	err := r.DB.Preload("User").Order("timestamp desc").Limit(100).Find(&logs).Error
	return logs, err
}
