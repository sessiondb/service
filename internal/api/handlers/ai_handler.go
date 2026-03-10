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

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AIHandler handles AI-related endpoints (generate SQL, explain, config).
type AIHandler struct {
	AIEngine    engine.AIEngine
	AIConfigRepo *repository.AIConfigRepository
}

// NewAIHandler returns a new AIHandler.
func NewAIHandler(aiEngine engine.AIEngine, aiConfigRepo *repository.AIConfigRepository) *AIHandler {
	return &AIHandler{AIEngine: aiEngine, AIConfigRepo: aiConfigRepo}
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

// GetAIConfig returns the current user's AI config (API key is never returned).
func (h *AIHandler) GetAIConfig(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	cfg, err := h.AIConfigRepo.GetUserAIConfig(userID)
	if err != nil || cfg == nil {
		c.JSON(http.StatusOK, gin.H{"configured": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"configured":   true,
		"providerType": cfg.ProviderType,
		"modelName":    cfg.ModelName,
		"baseUrl":      cfg.BaseURL,
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
