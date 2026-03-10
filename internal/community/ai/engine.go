// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package ai

import (
	"context"
	"sessiondb/internal/apierrors"
	"sessiondb/internal/engine"
	"sessiondb/internal/models"
	"sessiondb/internal/repository"
	"sessiondb/internal/utils"
	"strings"

	"github.com/google/uuid"
)

// Engine implements engine.AIEngine for community (BYOK).
type Engine struct {
	AIConfigRepo   *repository.AIConfigRepository
	AccessEngine   engine.AccessEngine
	MetaRepo       *repository.MetadataRepository
	UserRepo       *repository.UserRepository
}

// NewEngine returns a new community AI engine.
func NewEngine(
	aiConfigRepo *repository.AIConfigRepository,
	accessEngine engine.AccessEngine,
	metaRepo *repository.MetadataRepository,
	userRepo *repository.UserRepository,
) *Engine {
	return &Engine{
		AIConfigRepo: aiConfigRepo,
		AccessEngine: accessEngine,
		MetaRepo:     metaRepo,
		UserRepo:     userRepo,
	}
}

// GenerateSQL produces SQL from a natural-language prompt using the user's BYOK config and schema filtered by permissions.
func (e *Engine) GenerateSQL(ctx context.Context, userID, instanceID uuid.UUID, prompt string, schema *engine.SchemaContext) (string, error) {
	cfg, err := e.AIConfigRepo.GetUserAIConfig(userID)
	if err != nil || cfg == nil {
		return "", errNoAIConfig
	}
	apiKey, err := utils.DecryptPassword(cfg.APIKey)
	if err != nil {
		return "", errInvalidAIConfig
	}
	baseURL := ""
	if cfg.BaseURL != nil {
		baseURL = *cfg.BaseURL
	}
	provider := NewOpenAICompatibleProvider(baseURL, apiKey, cfg.ModelName)
	if schema == nil {
		schema, err = e.buildSchemaContext(ctx, userID, instanceID)
		if err != nil {
			return "", err
		}
	}
	return provider.GenerateSQL(ctx, prompt, schema)
}

// ClassifyIntent returns intent from prompt using simple heuristics (no LLM call).
func (e *Engine) ClassifyIntent(ctx context.Context, userID uuid.UUID, prompt string) (engine.Intent, error) {
	lower := strings.ToLower(strings.TrimSpace(prompt))
	if strings.Contains(lower, "explain") || strings.Contains(lower, "what does") {
		return engine.IntentExplain, nil
	}
	if strings.Contains(lower, "create table") || strings.Contains(lower, "drop table") ||
		strings.Contains(lower, "alter table") || strings.Contains(lower, "create user") ||
		strings.Contains(lower, "drop user") {
		return engine.IntentDDL, nil
	}
	if strings.Contains(lower, "insert") || strings.Contains(lower, "update") ||
		strings.Contains(lower, "delete") {
		return engine.IntentMutation, nil
	}
	return engine.IntentQuery, nil
}

// ExplainQuery returns a short explanation of the SQL using the user's BYOK config.
func (e *Engine) ExplainQuery(ctx context.Context, userID uuid.UUID, query string) (string, error) {
	cfg, err := e.AIConfigRepo.GetUserAIConfig(userID)
	if err != nil || cfg == nil {
		return "", errNoAIConfig
	}
	apiKey, err := utils.DecryptPassword(cfg.APIKey)
	if err != nil {
		return "", errInvalidAIConfig
	}
	baseURL := ""
	if cfg.BaseURL != nil {
		baseURL = *cfg.BaseURL
	}
	provider := NewOpenAICompatibleProvider(baseURL, apiKey, cfg.ModelName)
	return provider.ExplainQuery(ctx, query)
}

// RequiresApproval returns true if the action type requires approval for this user on the instance.
func (e *Engine) RequiresApproval(ctx context.Context, userID uuid.UUID, instanceID uuid.UUID, actionType string) (bool, error) {
	policies, err := e.AIConfigRepo.GetAIExecutionPolicies(instanceID)
	if err != nil || len(policies) == 0 {
		return true, nil // default: require approval when no policy
	}
	user, err := e.UserRepo.FindByID(userID)
	if err != nil {
		return true, err
	}
	roleName := ""
	if user.Role.Name != "" {
		roleName = user.Role.Name
	}
	for _, p := range policies {
		if p.ActionType != actionType {
			continue
		}
		if !p.RequireApproval {
			return false, nil
		}
		for _, r := range p.AllowedRoles {
			if r == roleName {
				return false, nil
			}
		}
		return true, nil
	}
	return true, nil
}

// buildSchemaContext returns schema for the instance filtered by user's effective permissions.
func (e *Engine) buildSchemaContext(ctx context.Context, userID, instanceID uuid.UUID) (*engine.SchemaContext, error) {
	perms, err := e.AccessEngine.GetEffectivePermissions(ctx, userID, instanceID)
	if err != nil {
		return nil, err
	}
	tables, err := e.MetaRepo.GetFullSchema(instanceID)
	if err != nil {
		return nil, err
	}
	allowed := make(map[string]map[string]struct{}) // "schema.table" -> set of columns
	for _, p := range perms {
		if !permHasPrivilege(&p, "SELECT") {
			continue
		}
		key := p.Schema + "." + p.Table
		if p.Table == "*" {
			key = "*"
		}
		if allowed[key] == nil {
			allowed[key] = make(map[string]struct{})
		}
		if p.Column == "" {
			allowed[key]["*"] = struct{}{}
		} else {
			allowed[key][p.Column] = struct{}{}
		}
	}
	var out engine.SchemaContext
	out.InstanceID = instanceID
	for _, t := range tables {
		key := t.Schema + "." + t.Name
		wild := "*"
		cols, hasTable := allowed[key]
		wildCols := allowed[wild]
		if !hasTable && wildCols == nil {
			continue
		}
		var colList []string
		if wildCols != nil && len(wildCols) > 0 {
			for _, c := range t.Columns {
				colList = append(colList, c.Name)
			}
		} else if cols != nil {
			if _, all := cols["*"]; all {
				for _, c := range t.Columns {
					colList = append(colList, c.Name)
				}
			} else {
				for _, c := range t.Columns {
					if _, ok := cols[c.Name]; ok {
						colList = append(colList, c.Name)
					}
				}
			}
		}
		if len(colList) > 0 {
			out.Tables = append(out.Tables, engine.TableSchema{Schema: t.Schema, Table: t.Name, Columns: colList})
		}
	}
	return &out, nil
}

func permHasPrivilege(p *models.Permission, privilege string) bool {
	for _, pr := range p.Privileges {
		if pr == privilege || pr == "ALL" || pr == "ALL PRIVILEGES" {
			return true
		}
	}
	return false
}

var (
	errNoAIConfig     = apierrors.NewAppError(400, apierrors.CodeInvalidRequest, "Configure your AI provider API key in settings to use this feature")
	errInvalidAIConfig = apierrors.NewAppError(400, apierrors.CodeInvalidRequest, "Invalid or expired API key")
)
