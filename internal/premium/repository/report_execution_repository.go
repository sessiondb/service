//go:build pro

// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"sessiondb/internal/models"
)

// ReportExecutionRepository handles persistence for ReportExecution (Phase 6).
type ReportExecutionRepository struct {
	DB *gorm.DB
}

// NewReportExecutionRepository returns a new ReportExecutionRepository.
func NewReportExecutionRepository(db *gorm.DB) *ReportExecutionRepository {
	return &ReportExecutionRepository{DB: db}
}

// Create persists a ReportExecution.
func (r *ReportExecutionRepository) Create(ex *models.ReportExecution) error {
	return r.DB.Create(ex).Error
}

// Update saves changes to a ReportExecution.
func (r *ReportExecutionRepository) Update(ex *models.ReportExecution) error {
	return r.DB.Save(ex).Error
}

// FindByDefinitionID returns executions for a definition, most recent first.
func (r *ReportExecutionRepository) FindByDefinitionID(definitionID uuid.UUID, limit int) ([]models.ReportExecution, error) {
	if limit <= 0 {
		limit = 50
	}
	var list []models.ReportExecution
	if err := r.DB.Where("definition_id = ?", definitionID).Order("started_at DESC").Limit(limit).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
