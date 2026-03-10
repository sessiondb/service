//go:build pro

// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"sessiondb/internal/models"
)

// AlertEventRepository handles persistence for AlertEvent (Phase 5).
type AlertEventRepository struct {
	DB *gorm.DB
}

// NewAlertEventRepository returns a new AlertEventRepository.
func NewAlertEventRepository(db *gorm.DB) *AlertEventRepository {
	return &AlertEventRepository{DB: db}
}

// Create persists an AlertEvent.
func (r *AlertEventRepository) Create(ev *models.AlertEvent) error {
	return r.DB.Create(ev).Error
}

// ListByTenant returns events for the tenant with optional ruleID and status filter.
func (r *AlertEventRepository) ListByTenant(tenantID uuid.UUID, ruleID *uuid.UUID, status string, limit int) ([]models.AlertEvent, error) {
	q := r.DB.Where("tenant_id = ?", tenantID)
	if ruleID != nil {
		q = q.Where("rule_id = ?", *ruleID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if limit <= 0 {
		limit = 100
	}
	var events []models.AlertEvent
	if err := q.Order("created_at DESC").Limit(limit).Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}
