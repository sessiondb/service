// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package engine

import (
	"context"

	"github.com/google/uuid"
	"sessiondb/internal/models"
)

// ReportEngine (Phase 6 – premium) runs report definitions and stores execution results.
type ReportEngine interface {
	// CreateDefinition persists a report definition.
	CreateDefinition(ctx context.Context, def *models.ReportDefinition) error
	// UpdateDefinition updates an existing definition.
	UpdateDefinition(ctx context.Context, def *models.ReportDefinition) error
	// GetDefinition returns a definition by ID.
	GetDefinition(ctx context.Context, id uuid.UUID) (*models.ReportDefinition, error)
	// ListDefinitions returns all definitions for the tenant.
	ListDefinitions(ctx context.Context, tenantID uuid.UUID) ([]models.ReportDefinition, error)
	// DeleteDefinition soft-deletes a definition.
	DeleteDefinition(ctx context.Context, id uuid.UUID) error
	// RunReport runs the report by definition ID; updates ReportExecution and optional ResultURL.
	RunReport(ctx context.Context, definitionID uuid.UUID) (*models.ReportExecution, error)
	// ListExecutions returns executions for a definition.
	ListExecutions(ctx context.Context, definitionID uuid.UUID) ([]models.ReportExecution, error)
}
