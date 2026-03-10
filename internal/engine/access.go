// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package engine

import (
	"context"
	"sessiondb/internal/models"

	"github.com/google/uuid"
)

// AccessEngine resolves data-level access (instance/database/schema/table/column).
// System RBAC (users:read, etc.) is enforced by middleware; this engine handles what data a user can query.
type AccessEngine interface {
	// CheckAccess returns true if the user may use the given privilege on the given resource.
	// Column may be empty for table-level check. Privilege is e.g. "SELECT", "INSERT", "UPDATE", "DELETE".
	CheckAccess(ctx context.Context, userID, instanceID uuid.UUID, database, schema, table, column, privilege string) (allowed bool, err error)
	// GetEffectivePermissions returns all permissions that apply to the user for the instance (direct + role).
	// Used by schema context builder and provisioning sync.
	GetEffectivePermissions(ctx context.Context, userID, instanceID uuid.UUID) ([]models.Permission, error)
}
