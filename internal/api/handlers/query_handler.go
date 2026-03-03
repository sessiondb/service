// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package handlers

import (
	"errors"
	"net/http"
	"sessiondb/internal/apierrors"
	"sessiondb/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type QueryHandler struct {
	Service *service.QueryService
}

func NewQueryHandler(service *service.QueryService) *QueryHandler {
	return &QueryHandler{Service: service}
}

type ExecuteQueryRequest struct {
	InstanceID string `json:"instanceId" binding:"required"`
	Query      string `json:"query" binding:"required"`
}

type SaveScriptRequest struct {
	Name     string `json:"name" binding:"required"`
	Query    string `json:"query" binding:"required"`
	IsPublic bool   `json:"isPublic"`
}

func (h *QueryHandler) ExecuteQuery(c *gin.Context) {
	var req ExecuteQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, err.Error()))
		return
	}

	userID := c.MustGet("userID").(uuid.UUID)

	instanceID, err := uuid.Parse(req.InstanceID)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, "Invalid instance ID"))
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.Request.UserAgent()

	result, err := h.Service.ExecuteQuery(userID, instanceID, req.Query, ipAddress, userAgent)
	if err != nil {
		// Use the sentinel AppError if it's one of ours, otherwise wrap it
		var appErr *apierrors.AppError
		if errors.As(err, &appErr) {
			apierrors.Respond(c, appErr)
		} else {
			// e.g. some generic SQL error
			apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *QueryHandler) GetHistory(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	history, err := h.Service.GetHistory(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, history)
}

func (h *QueryHandler) SaveScript(c *gin.Context) {
	var req SaveScriptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.MustGet("userID").(uuid.UUID)

	script, err := h.Service.SaveScript(userID, req.Name, req.Query, req.IsPublic)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, script)
}

func (h *QueryHandler) GetScripts(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	scripts, err := h.Service.GetScripts(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, scripts)
}
