// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package handlers

import (
	"errors"
	"net/http"
	"sessiondb/internal/apierrors"
	"sessiondb/internal/engine"
	"sessiondb/internal/models"
	"sessiondb/internal/repository"
	"sessiondb/internal/utils"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AIHandler handles AI-related endpoints (generate SQL, explain, config, usage).
type AIHandler struct {
	AIEngine     engine.AIEngine
	AIConfigRepo *repository.AIConfigRepository
	UsageRepo    *repository.AIUsageRepository
}

// NewAIHandler returns a new AIHandler.
func NewAIHandler(aiEngine engine.AIEngine, aiConfigRepo *repository.AIConfigRepository, usageRepo *repository.AIUsageRepository) *AIHandler {
	return &AIHandler{AIEngine: aiEngine, AIConfigRepo: aiConfigRepo, UsageRepo: usageRepo}
}

// GenerateSQLRequest is the request body for generating SQL from a prompt.
type GenerateSQLRequest struct {
	InstanceID string `json:"instanceId" binding:"required"`
	Prompt     string `json:"prompt" binding:"required"`
}

// GenerateSQLResponse is the response for generate SQL.
type GenerateSQLResponse struct {
	SQL              string `json:"sql"`
	RequiresApproval bool   `json:"requiresApproval"`
}

// GenerateSQL generates SQL from a natural-language prompt using the user's BYOK AI config.
func (h *AIHandler) GenerateSQL(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	var req GenerateSQLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, err.Error()))
		return
	}
	instanceID, err := uuid.Parse(req.InstanceID)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, "Invalid instance ID"))
		return
	}
	sql, err := h.AIEngine.GenerateSQL(c.Request.Context(), userID, instanceID, req.Prompt, nil)
	if err != nil {
		var appErr *apierrors.AppError
		if errors.As(err, &appErr) {
			apierrors.Respond(c, appErr)
		} else {
			apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, err.Error()))
		}
		return
	}
	if h.UsageRepo != nil {
		uc, _ := h.AIConfigRepo.GetUserAIConfig(userID)
		useGlobal := uc == nil || uc.APIKey == ""
		_ = h.UsageRepo.RecordUsage(&models.AITokenUsage{
			UserID: userID, UsedGlobal: useGlobal, Feature: "generate_sql", Model: "",
		})
	}
	intent, _ := h.AIEngine.ClassifyIntent(c.Request.Context(), userID, req.Prompt)
	actionType := intentToActionType(intent)
	requiresApproval, _ := h.AIEngine.RequiresApproval(c.Request.Context(), userID, instanceID, actionType)
	c.JSON(http.StatusOK, GenerateSQLResponse{SQL: sql, RequiresApproval: requiresApproval})
}

// ExplainRequest is the request body for explaining a query.
type ExplainRequest struct {
	Query string `json:"query" binding:"required"`
}

// ExplainQuery returns a short explanation of the given SQL.
func (h *AIHandler) ExplainQuery(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	var req ExplainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, err.Error()))
		return
	}
	explanation, err := h.AIEngine.ExplainQuery(c.Request.Context(), userID, req.Query)
	if err != nil {
		var appErr *apierrors.AppError
		if errors.As(err, &appErr) {
			apierrors.Respond(c, appErr)
		} else {
			apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, err.Error()))
		}
		return
	}
	if h.UsageRepo != nil {
		uc, _ := h.AIConfigRepo.GetUserAIConfig(userID)
		useGlobal := uc == nil || uc.APIKey == ""
		_ = h.UsageRepo.RecordUsage(&models.AITokenUsage{
			UserID: userID, UsedGlobal: useGlobal, Feature: "explain", Model: "",
		})
	}
	c.JSON(http.StatusOK, gin.H{"explanation": explanation})
}

func intentToActionType(intent engine.Intent) string {
	switch intent {
	case engine.IntentQuery:
		return "SELECT"
	case engine.IntentMutation:
		return "UPDATE"
	case engine.IntentDDL:
		return "DDL"
	case engine.IntentUserMgmt:
		return "USER_MGMT"
	default:
		return "SELECT"
	}
}

// GetAIConfig returns the effective AI config for the current user (user key first, then global). API key is never returned. source is "user" or "global".
func (h *AIHandler) GetAIConfig(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	uc, err := h.AIConfigRepo.GetUserAIConfig(userID)
	if err == nil && uc != nil && uc.APIKey != "" {
		c.JSON(http.StatusOK, gin.H{
			"configured":   true,
			"source":       "user",
			"providerType": uc.ProviderType,
			"modelName":    uc.ModelName,
			"baseUrl":      uc.BaseURL,
		})
		return
	}
	gc, gerr := h.AIConfigRepo.GetGlobalAIConfig()
	if gerr != nil || gc == nil || gc.APIKey == "" {
		c.JSON(http.StatusOK, gin.H{"configured": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"configured":   true,
		"source":       "global",
		"providerType": gc.ProviderType,
		"modelName":    gc.ModelName,
		"baseUrl":      gc.BaseURL,
	})
}

// UpdateAIConfigRequest is the request body for saving AI config (BYOK).
type UpdateAIConfigRequest struct {
	ProviderType string  `json:"providerType" binding:"required"`
	APIKey       string  `json:"apiKey" binding:"required"`
	BaseURL      *string `json:"baseUrl"`
	ModelName    string  `json:"modelName"`
}

// UpdateAIConfig saves the user's AI provider config (API key is encrypted).
func (h *AIHandler) UpdateAIConfig(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	var req UpdateAIConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, err.Error()))
		return
	}
	encrypted, err := utils.EncryptPassword(req.APIKey)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusInternalServerError, apierrors.CodeInternalError, "Failed to store API key"))
		return
	}
	cfg := &models.UserAIConfig{
		UserID:       userID,
		ProviderType: req.ProviderType,
		APIKey:       encrypted,
		BaseURL:      req.BaseURL,
		ModelName:    req.ModelName,
	}
	existing, _ := h.AIConfigRepo.GetUserAIConfig(userID)
	if existing != nil {
		cfg.ID = existing.ID
	}
	if err := h.AIConfigRepo.SaveUserAIConfig(cfg); err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusInternalServerError, apierrors.CodeInternalError, err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// UpdateGlobalAIConfig saves the organization-wide AI config (admin only). Same request body as UpdateAIConfig.
func (h *AIHandler) UpdateGlobalAIConfig(c *gin.Context) {
	var req UpdateAIConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, err.Error()))
		return
	}
	encrypted, err := utils.EncryptPassword(req.APIKey)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusInternalServerError, apierrors.CodeInternalError, "Failed to store API key"))
		return
	}
	existing, _ := h.AIConfigRepo.GetGlobalAIConfig()
	cfg := &models.GlobalAIConfig{
		ProviderType: req.ProviderType,
		APIKey:       encrypted,
		BaseURL:      req.BaseURL,
		ModelName:    req.ModelName,
	}
	if existing != nil {
		cfg.ID = existing.ID
	}
	if err := h.AIConfigRepo.SaveGlobalAIConfig(cfg); err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusInternalServerError, apierrors.CodeInternalError, err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// GetAIUsage returns the current user's AI usage for the last 30 days.
func (h *AIHandler) GetAIUsage(c *gin.Context) {
	if h.UsageRepo == nil {
		c.JSON(http.StatusOK, gin.H{"usage": []interface{}{}, "total": 0})
		return
	}
	userID := c.MustGet("userID").(uuid.UUID)
	to := time.Now()
	from := to.AddDate(0, 0, -30)
	list, err := h.UsageRepo.GetUsageByUser(userID, from, to)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusInternalServerError, apierrors.CodeInternalError, err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"usage": list, "total": len(list)})
}

// GetAdminAIUsage returns global usage total and per-user summary for the last 30 days (admin only).
func (h *AIHandler) GetAdminAIUsage(c *gin.Context) {
	if h.UsageRepo == nil {
		c.JSON(http.StatusOK, gin.H{"global": gin.H{"count": 0, "inputTokens": 0, "outputTokens": 0}, "byUser": []interface{}{}})
		return
	}
	to := time.Now()
	from := to.AddDate(0, 0, -30)
	count, inTok, outTok, err := h.UsageRepo.GetGlobalUsageTotal(from, to)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusInternalServerError, apierrors.CodeInternalError, err.Error()))
		return
	}
	byUser, err := h.UsageRepo.GetUsageSummaryByUser(from, to)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusInternalServerError, apierrors.CodeInternalError, err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"global": gin.H{"count": count, "inputTokens": inTok, "outputTokens": outTok},
		"byUser": byUser,
	})
}
