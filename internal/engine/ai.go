// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package engine

import (
	"context"

	"github.com/google/uuid"
)

// SchemaContext is the filtered schema (tables/columns) the user is allowed to see, for LLM context.
type SchemaContext struct {
	InstanceID uuid.UUID
	Database   string
	Tables     []TableSchema
}

// TableSchema describes one table and its allowed columns for the AI context.
type TableSchema struct {
	Schema  string
	Table   string
	Columns []string
}

// Intent classifies what kind of operation the user wants (e.g. query vs mutation).
type Intent string

const (
	IntentQuery     Intent = "query"
	IntentMutation  Intent = "mutation"
	IntentDDL       Intent = "ddl"
	IntentUserMgmt  Intent = "user_mgmt"
	IntentExplain   Intent = "explain"
	IntentUnknown   Intent = "unknown"
)

// AIEngine provides text-based SQL generation and optional execution with safety checks.
type AIEngine interface {
	// GenerateSQL produces SQL from a natural-language prompt and schema context.
	GenerateSQL(ctx context.Context, userID uuid.UUID, instanceID uuid.UUID, prompt string, schema *SchemaContext) (sql string, err error)
	// ClassifyIntent returns the intent of the prompt (e.g. query, mutation, DDL).
	ClassifyIntent(ctx context.Context, userID uuid.UUID, prompt string) (Intent, error)
	// ExplainQuery returns a short explanation of the given SQL.
	ExplainQuery(ctx context.Context, userID uuid.UUID, query string) (explanation string, err error)
	// RequiresApproval returns true if the generated SQL must be approved before execution for this instance/user.
	RequiresApproval(ctx context.Context, userID uuid.UUID, instanceID uuid.UUID, actionType string) (bool, error)
}
