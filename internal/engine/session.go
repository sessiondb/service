// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package engine

import (
	"context"
	"time"

	"github.com/google/uuid"
	"sessiondb/internal/models"
)

// SessionEngine (Phase 4 – premium) manages ephemeral DB users and vault-style credential leases with full audit traceability.
type SessionEngine interface {
	// StartSession creates an ephemeral DB user (or checks out a lease), grants permissions from the user's Permission set, and returns the session and temporary password.
	StartSession(ctx context.Context, userID, instanceID uuid.UUID, ttl time.Duration) (*models.CredentialSession, string, error)
	// EndSession ends the session: drops the ephemeral user (or revokes the lease) and updates the CredentialSession record.
	EndSession(ctx context.Context, sessionID uuid.UUID) error
	// GetActiveSession returns the active CredentialSession for the user on the given instance, if any.
	GetActiveSession(ctx context.Context, userID, instanceID uuid.UUID) (*models.CredentialSession, error)
	// GetSessionByID returns a session by ID (for ownership checks).
	GetSessionByID(ctx context.Context, sessionID uuid.UUID) (*models.CredentialSession, error)
}
