// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package access

import (
	"context"
	"sessiondb/internal/engine"
	"sessiondb/internal/models"
	"sessiondb/internal/repository"

	"github.com/google/uuid"
)

// Engine implements engine.AccessEngine for community: column-to-instance access resolution.
type Engine struct {
	PermRepo *repository.PermissionRepository
}

// NewEngine returns a new community AccessEngine.
func NewEngine(permRepo *repository.PermissionRepository) *Engine {
	return &Engine{PermRepo: permRepo}
}

// CheckAccess returns true if the user has the requested privilege on the given resource.
// It resolves effective permissions (direct + role) and checks the most specific matching grant.
func (e *Engine) CheckAccess(ctx context.Context, userID, instanceID uuid.UUID, database, schema, table, column, privilege string) (bool, error) {
	perms, err := e.PermRepo.FindByUserAndInstance(userID, instanceID)
	if err != nil {
		return false, err
	}
	for _, p := range perms {
		if !permMatchesResource(&p, instanceID, database, schema, table, column) {
			continue
		}
		if permHasPrivilege(&p, privilege) {
			return true, nil
		}
	}
	return false, nil
}

// GetEffectivePermissions returns all permissions that apply to the user for the instance.
func (e *Engine) GetEffectivePermissions(ctx context.Context, userID, instanceID uuid.UUID) ([]models.Permission, error) {
	return e.PermRepo.FindByUserAndInstance(userID, instanceID)
}

// permMatchesResource returns true if the permission applies to the given resource.
// More specific (column) takes precedence; we accept if the grant covers the resource at any level.
func permMatchesResource(p *models.Permission, instanceID uuid.UUID, database, schema, table, column string) bool {
	if p.InstanceID != nil && *p.InstanceID != instanceID {
		return false
	}
	if p.Database != "*" && p.Database != database {
		return false
	}
	if p.Schema != "" && p.Schema != schema {
		return false
	}
	if p.Table != "*" && p.Table != table {
		return false
	}
	if p.Column != "" && p.Column != column {
		return false
	}
	return true
}

// permHasPrivilege returns true if the permission grants the requested privilege (e.g. SELECT).
func permHasPrivilege(p *models.Permission, privilege string) bool {
	for _, pr := range p.Privileges {
		if pr == privilege || pr == "ALL" || pr == "ALL PRIVILEGES" {
			return true
		}
	}
	return false
}

// Ensure Engine implements engine.AccessEngine.
var _ engine.AccessEngine = (*Engine)(nil)
