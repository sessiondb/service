//go:build pro

package alert

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"sessiondb/internal/engine"
	"sessiondb/internal/models"
	"sessiondb/internal/premium/repository"
)

// conditionThreshold describes a simple threshold condition (e.g. duration_ms > 5000).
type conditionThreshold struct {
	Metric string      `json:"metric"`
	Op     string      `json:"op"` // ">", ">=", "<", "<=", "=="
	Value  interface{} `json:"value"`
}

// Engine implements engine.AlertEngine for premium.
type Engine struct {
	RuleRepo  *repository.AlertRuleRepository
	EventRepo *repository.AlertEventRepository
}

// NewEngine returns a new premium AlertEngine.
func NewEngine(ruleRepo *repository.AlertRuleRepository, eventRepo *repository.AlertEventRepository) *Engine {
	return &Engine{RuleRepo: ruleRepo, EventRepo: eventRepo}
}

// Ensure Engine implements engine.AlertEngine.
var _ engine.AlertEngine = (*Engine)(nil)

// CreateRule persists an alert rule.
func (e *Engine) CreateRule(ctx context.Context, rule *models.AlertRule) error {
	return e.RuleRepo.Create(rule)
}

// UpdateRule updates an existing rule.
func (e *Engine) UpdateRule(ctx context.Context, rule *models.AlertRule) error {
	return e.RuleRepo.Update(rule)
}

// GetRule returns a rule by ID.
func (e *Engine) GetRule(ctx context.Context, ruleID uuid.UUID) (*models.AlertRule, error) {
	return e.RuleRepo.FindByID(ruleID)
}

// ListRules returns all rules for the tenant.
func (e *Engine) ListRules(ctx context.Context, tenantID uuid.UUID) ([]models.AlertRule, error) {
	return e.RuleRepo.FindByTenantID(tenantID)
}

// DeleteRule soft-deletes a rule.
func (e *Engine) DeleteRule(ctx context.Context, ruleID uuid.UUID) error {
	return e.RuleRepo.Delete(ruleID)
}

// EvaluateRules runs enabled rules for the event source and creates events on match.
func (e *Engine) EvaluateRules(ctx context.Context, eventSource string, payload map[string]interface{}) error {
	rules, err := e.RuleRepo.FindEnabledByEventSource(eventSource)
	if err != nil {
		return err
	}
	for i := range rules {
		rule := &rules[i]
		matched, title, desc := e.evaluateRule(rule, payload)
		if !matched {
			continue
		}
		ev := &models.AlertEvent{
			RuleID:      rule.ID,
			TenantID:    rule.TenantID,
			Severity:    rule.Severity,
			Title:       title,
			Description: desc,
			Status:      "open",
			Source:      eventSource,
		}
		if meta, _ := json.Marshal(payload); len(meta) > 0 {
			ev.Metadata = meta
		}
		if err := e.EventRepo.Create(ev); err != nil {
			return err
		}
	}
	return nil
}

// evaluateRule returns whether the payload matches the rule condition and suggested title/description.
func (e *Engine) evaluateRule(rule *models.AlertRule, payload map[string]interface{}) (matched bool, title, desc string) {
	var cond conditionThreshold
	if len(rule.Condition) == 0 {
		return false, "", ""
	}
	if err := json.Unmarshal(rule.Condition, &cond); err != nil {
		return false, "", ""
	}
	val, ok := payload[cond.Metric]
	if !ok {
		return false, "", ""
	}
	cmp := compareValue(val, cond.Value, cond.Op)
	if !cmp {
		return false, "", ""
	}
	title = rule.Name
	desc = fmt.Sprintf("Condition %s %s %v matched (got %v)", cond.Metric, cond.Op, cond.Value, val)
	return true, title, desc
}

// compareValue performs a simple comparison (numeric or string).
func compareValue(got, want interface{}, op string) bool {
	switch op {
	case ">":
		return toFloat(got) > toFloat(want)
	case ">=":
		return toFloat(got) >= toFloat(want)
	case "<":
		return toFloat(got) < toFloat(want)
	case "<=":
		return toFloat(got) <= toFloat(want)
	case "==":
		return fmt.Sprint(got) == fmt.Sprint(want)
	default:
		return false
	}
}

func toFloat(v interface{}) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int:
		return float64(x)
	case int64:
		return float64(x)
	default:
		return 0
	}
}

// ListEvents returns alert events for the tenant.
func (e *Engine) ListEvents(ctx context.Context, tenantID uuid.UUID, ruleID *uuid.UUID, status string) ([]models.AlertEvent, error) {
	return e.EventRepo.ListByTenant(tenantID, ruleID, status, 100)
}
