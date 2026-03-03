// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package repository

import (
	"sessiondb/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MonitoringRepository struct {
	db *gorm.DB
}

func NewMonitoringRepository(db *gorm.DB) *MonitoringRepository {
	return &MonitoringRepository{db: db}
}

func (r *MonitoringRepository) CreateLog(log *models.DBMonitoringLog) error {
	return r.db.Create(log).Error
}

func (r *MonitoringRepository) GetLatestLog(instanceID uuid.UUID) (*models.DBMonitoringLog, error) {
	var log models.DBMonitoringLog
	err := r.db.Where("instance_id = ?", instanceID).Order("created_at desc").First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (r *MonitoringRepository) GetLogs(instanceID uuid.UUID, limit int) ([]models.DBMonitoringLog, error) {
	var logs []models.DBMonitoringLog
	err := r.db.Where("instance_id = ?", instanceID).Order("created_at desc").Limit(limit).Find(&logs).Error
	return logs, err
}
