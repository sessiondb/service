//go:build pro

package alert

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"sessiondb/internal/apierrors"
	"sessiondb/internal/engine"
	"sessiondb/internal/models"
)

// CreateRuleBody is the request body for creating an alert rule (Condition/Channels as JSON).
type CreateRuleBody struct {
	Name        string          `json:"name" binding:"required"`
	Description string          `json:"description"`
	EventSource string          `json:"eventSource" binding:"required"`
	Condition   json.RawMessage `json:"condition"`
	Severity    string          `json:"severity"`
	IsEnabled   *bool           `json:"isEnabled"`
	Channels    json.RawMessage `json:"channels"`
}

// Handler handles premium alert HTTP endpoints.
type Handler struct {
	AlertEngine engine.AlertEngine
}

// NewHandler returns a new alert Handler.
func NewHandler(alertEngine engine.AlertEngine) *Handler {
	return &Handler{AlertEngine: alertEngine}
}

// TenantIDFromContext returns tenant ID; for now we use a placeholder (e.g. system tenant) until multi-tenant is wired.
func TenantIDFromContext(c *gin.Context) uuid.UUID {
	// TODO: when tenant context exists, use c.Get("tenantID")
	return uuid.Nil
}

// CreateRule creates an alert rule.
// POST /alerts/rules
func (h *Handler) CreateRule(c *gin.Context) {
	tenantID := TenantIDFromContext(c)
	userID := c.MustGet("userID").(uuid.UUID)
	var body CreateRuleBody
	if err := c.ShouldBindJSON(&body); err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, err.Error()))
		return
	}
	rule := models.AlertRule{
		TenantID:    tenantID,
		CreatedBy:   userID,
		Name:        body.Name,
		Description: body.Description,
		EventSource: body.EventSource,
		Condition:   body.Condition,
		Channels:   body.Channels,
		Severity:    body.Severity,
		IsEnabled:   true,
	}
	if body.Severity == "" {
		rule.Severity = "medium"
	}
	if body.IsEnabled != nil {
		rule.IsEnabled = *body.IsEnabled
	}
	if err := h.AlertEngine.CreateRule(c.Request.Context(), &rule); err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusInternalServerError, apierrors.CodeInternalError, err.Error()))
		return
	}
	c.JSON(http.StatusCreated, rule)
}

// ListRules lists alert rules for the tenant.
// GET /alerts/rules
func (h *Handler) ListRules(c *gin.Context) {
	tenantID := TenantIDFromContext(c)
	rules, err := h.AlertEngine.ListRules(c.Request.Context(), tenantID)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusInternalServerError, apierrors.CodeInternalError, err.Error()))
		return
	}
	c.JSON(http.StatusOK, rules)
}

// GetRule returns one rule by ID.
// GET /alerts/rules/:id
func (h *Handler) GetRule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, "Invalid rule ID"))
		return
	}
	rule, err := h.AlertEngine.GetRule(c.Request.Context(), id)
	if err != nil || rule == nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusNotFound, apierrors.CodeNotFound, "Rule not found"))
		return
	}
	c.JSON(http.StatusOK, rule)
}

// UpdateRuleBody is the request body for updating an alert rule.
type UpdateRuleBody struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	EventSource string          `json:"eventSource"`
	Condition   json.RawMessage `json:"condition"`
	Severity    string          `json:"severity"`
	IsEnabled   *bool           `json:"isEnabled"`
	Channels    json.RawMessage `json:"channels"`
}

// UpdateRule updates a rule.
// PUT /alerts/rules/:id
func (h *Handler) UpdateRule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, "Invalid rule ID"))
		return
	}
	rule, err := h.AlertEngine.GetRule(c.Request.Context(), id)
	if err != nil || rule == nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusNotFound, apierrors.CodeNotFound, "Rule not found"))
		return
	}
	var body UpdateRuleBody
	if err := c.ShouldBindJSON(&body); err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, err.Error()))
		return
	}
	if body.Name != "" {
		rule.Name = body.Name
	}
	if body.Description != "" || body.Name != "" {
		rule.Description = body.Description
	}
	if body.EventSource != "" {
		rule.EventSource = body.EventSource
	}
	if len(body.Condition) > 0 {
		rule.Condition = body.Condition
	}
	if body.Severity != "" {
		rule.Severity = body.Severity
	}
	if body.IsEnabled != nil {
		rule.IsEnabled = *body.IsEnabled
	}
	if len(body.Channels) > 0 {
		rule.Channels = body.Channels
	}
	if err := h.AlertEngine.UpdateRule(c.Request.Context(), rule); err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusInternalServerError, apierrors.CodeInternalError, err.Error()))
		return
	}
	c.JSON(http.StatusOK, rule)
}

// DeleteRule deletes a rule.
// DELETE /alerts/rules/:id
func (h *Handler) DeleteRule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, "Invalid rule ID"))
		return
	}
	if err := h.AlertEngine.DeleteRule(c.Request.Context(), id); err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusInternalServerError, apierrors.CodeInternalError, err.Error()))
		return
	}
	c.Status(http.StatusNoContent)
}

// ListEvents lists alert events (query: ruleId, status).
// GET /alerts/events
func (h *Handler) ListEvents(c *gin.Context) {
	tenantID := TenantIDFromContext(c)
	var ruleID *uuid.UUID
	if s := c.Query("ruleId"); s != "" {
		id, err := uuid.Parse(s)
		if err == nil {
			ruleID = &id
		}
	}
	status := c.Query("status")
	events, err := h.AlertEngine.ListEvents(c.Request.Context(), tenantID, ruleID, status)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusInternalServerError, apierrors.CodeInternalError, err.Error()))
		return
	}
	c.JSON(http.StatusOK, events)
}
