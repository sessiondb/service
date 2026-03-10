// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package engine

import (
	"context"

	"github.com/google/uuid"
	"sessiondb/internal/models"
)

// AlertEngine (Phase 5 – premium) evaluates rules and creates alert events.
type AlertEngine interface {
	// CreateRule persists an alert rule for the tenant.
	CreateRule(ctx context.Context, rule *models.AlertRule) error
	// UpdateRule updates an existing rule.
	UpdateRule(ctx context.Context, rule *models.AlertRule) error
	// GetRule returns a rule by ID.
	GetRule(ctx context.Context, ruleID uuid.UUID) (*models.AlertRule, error)
	// ListRules returns all rules for the tenant.
	ListRules(ctx context.Context, tenantID uuid.UUID) ([]models.AlertRule, error)
	// DeleteRule soft-deletes a rule.
	DeleteRule(ctx context.Context, ruleID uuid.UUID) error
	// EvaluateRules runs enabled rules for the given event source and payload; creates AlertEvents on match.
	EvaluateRules(ctx context.Context, eventSource string, payload map[string]interface{}) error
	// ListEvents returns alert events for the tenant with optional filter (ruleID, status).
	ListEvents(ctx context.Context, tenantID uuid.UUID, ruleID *uuid.UUID, status string) ([]models.AlertEvent, error)
}
