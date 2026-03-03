// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package handlers

import (
	"net/http"
	"sessiondb/internal/service"

	"github.com/gin-gonic/gin"
)

type ConfigHandler struct {
	Service *service.ConfigService
}

func NewConfigHandler(service *service.ConfigService) *ConfigHandler {
	return &ConfigHandler{Service: service}
}

type UpdateAuthConfigRequest struct {
	Type string `json:"type" binding:"required"`
}

func (h *ConfigHandler) GetAuthConfig(c *gin.Context) {
	config := h.Service.GetAuthConfig()
	c.JSON(http.StatusOK, config)
}

func (h *ConfigHandler) UpdateAuthConfig(c *gin.Context) {
	var req UpdateAuthConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.Service.UpdateAuthConfig(req.Type)
	c.JSON(http.StatusOK, h.Service.GetAuthConfig())
}
