//go:build pro

// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"sessiondb/internal/models"
)

// AlertRuleRepository handles persistence for AlertRule (Phase 5).
type AlertRuleRepository struct {
	DB *gorm.DB
}

// NewAlertRuleRepository returns a new AlertRuleRepository.
func NewAlertRuleRepository(db *gorm.DB) *AlertRuleRepository {
	return &AlertRuleRepository{DB: db}
}

// Create persists an AlertRule.
func (r *AlertRuleRepository) Create(rule *models.AlertRule) error {
	return r.DB.Create(rule).Error
}

// Update saves changes to an AlertRule.
func (r *AlertRuleRepository) Update(rule *models.AlertRule) error {
	return r.DB.Save(rule).Error
}

// FindByID loads an AlertRule by ID.
func (r *AlertRuleRepository) FindByID(id uuid.UUID) (*models.AlertRule, error) {
	var rule models.AlertRule
	if err := r.DB.First(&rule, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

// FindByTenantID returns all rules for the tenant.
func (r *AlertRuleRepository) FindByTenantID(tenantID uuid.UUID) ([]models.AlertRule, error) {
	var rules []models.AlertRule
	if err := r.DB.Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// FindEnabledByEventSource returns enabled rules for the event source (for evaluation).
func (r *AlertRuleRepository) FindEnabledByEventSource(eventSource string) ([]models.AlertRule, error) {
	var rules []models.AlertRule
	if err := r.DB.Where("event_source = ? AND is_enabled = ?", eventSource, true).Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// Delete soft-deletes an AlertRule.
func (r *AlertRuleRepository) Delete(id uuid.UUID) error {
	return r.DB.Delete(&models.AlertRule{}, "id = ?", id).Error
}
