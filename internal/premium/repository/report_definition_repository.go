//go:build pro

// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"sessiondb/internal/models"
)

// ReportDefinitionRepository handles persistence for ReportDefinition (Phase 6).
type ReportDefinitionRepository struct {
	DB *gorm.DB
}

// NewReportDefinitionRepository returns a new ReportDefinitionRepository.
func NewReportDefinitionRepository(db *gorm.DB) *ReportDefinitionRepository {
	return &ReportDefinitionRepository{DB: db}
}

// Create persists a ReportDefinition.
func (r *ReportDefinitionRepository) Create(def *models.ReportDefinition) error {
	return r.DB.Create(def).Error
}

// Update saves changes to a ReportDefinition.
func (r *ReportDefinitionRepository) Update(def *models.ReportDefinition) error {
	return r.DB.Save(def).Error
}

// FindByID loads a ReportDefinition by ID.
func (r *ReportDefinitionRepository) FindByID(id uuid.UUID) (*models.ReportDefinition, error) {
	var def models.ReportDefinition
	if err := r.DB.First(&def, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &def, nil
}

// FindByTenantID returns all definitions for the tenant.
func (r *ReportDefinitionRepository) FindByTenantID(tenantID uuid.UUID) ([]models.ReportDefinition, error) {
	var defs []models.ReportDefinition
	if err := r.DB.Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&defs).Error; err != nil {
		return nil, err
	}
	return defs, nil
}

// Delete soft-deletes a ReportDefinition.
func (r *ReportDefinitionRepository) Delete(id uuid.UUID) error {
	return r.DB.Delete(&models.ReportDefinition{}, "id = ?", id).Error
}
