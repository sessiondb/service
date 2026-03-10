//go:build pro

package report

import (
	"context"
	"time"

	"github.com/google/uuid"
	"sessiondb/internal/engine"
	"sessiondb/internal/models"
	"sessiondb/internal/premium/repository"
)

// Engine implements engine.ReportEngine for premium (on-demand run; optional scheduler later).
type Engine struct {
	DefRepo  *repository.ReportDefinitionRepository
	ExecRepo *repository.ReportExecutionRepository
}

// NewEngine returns a new premium ReportEngine.
func NewEngine(defRepo *repository.ReportDefinitionRepository, execRepo *repository.ReportExecutionRepository) *Engine {
	return &Engine{DefRepo: defRepo, ExecRepo: execRepo}
}

// Ensure Engine implements engine.ReportEngine.
var _ engine.ReportEngine = (*Engine)(nil)

// CreateDefinition persists a report definition.
func (e *Engine) CreateDefinition(ctx context.Context, def *models.ReportDefinition) error {
	return e.DefRepo.Create(def)
}

// UpdateDefinition updates an existing definition.
func (e *Engine) UpdateDefinition(ctx context.Context, def *models.ReportDefinition) error {
	return e.DefRepo.Update(def)
}

// GetDefinition returns a definition by ID.
func (e *Engine) GetDefinition(ctx context.Context, id uuid.UUID) (*models.ReportDefinition, error) {
	return e.DefRepo.FindByID(id)
}

// ListDefinitions returns all definitions for the tenant.
func (e *Engine) ListDefinitions(ctx context.Context, tenantID uuid.UUID) ([]models.ReportDefinition, error) {
	return e.DefRepo.FindByTenantID(tenantID)
}

// DeleteDefinition soft-deletes a definition.
func (e *Engine) DeleteDefinition(ctx context.Context, id uuid.UUID) error {
	return e.DefRepo.Delete(id)
}

// RunReport runs the report by definition ID: creates ReportExecution, marks completed with placeholder result.
func (e *Engine) RunReport(ctx context.Context, definitionID uuid.UUID) (*models.ReportExecution, error) {
	def, err := e.DefRepo.FindByID(definitionID)
	if err != nil || def == nil {
		return nil, err
	}
	now := time.Now()
	ex := &models.ReportExecution{
		DefinitionID: definitionID,
		Status:      "running",
		StartedAt:   now,
	}
	if err := e.ExecRepo.Create(ex); err != nil {
		return nil, err
	}
	// On-demand: no actual query execution yet; just mark completed with a placeholder result URL.
	// Full implementation would run DataSources queries, write CSV/JSON to storage, set ResultURL.
	completed := now
	ex.Status = "completed"
	ex.CompletedAt = &completed
	ex.ResultURL = "/reports/download/" + ex.ID.String()
	if err := e.ExecRepo.Update(ex); err != nil {
		return nil, err
	}
	def.LastRunAt = &completed
	_ = e.DefRepo.Update(def)
	return ex, nil
}

// ListExecutions returns executions for a definition.
func (e *Engine) ListExecutions(ctx context.Context, definitionID uuid.UUID) ([]models.ReportExecution, error) {
	return e.ExecRepo.FindByDefinitionID(definitionID, 50)
}
