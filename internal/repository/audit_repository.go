// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package repository

import (
	"sessiondb/internal/models"

	"github.com/google/uuid"
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

func (r *AuditRepository) FindByUserAndAction(userID uuid.UUID, action string) ([]models.AuditLog, error) {
	var logs = make([]models.AuditLog, 0)
	err := r.DB.Where("user_id = ? AND action = ?", userID, action).Order("timestamp desc").Limit(100).Find(&logs).Error
	return logs, err
}
