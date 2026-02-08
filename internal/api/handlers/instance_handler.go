package handlers

import (
	"net/http"
	"sessiondb/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type InstanceHandler struct {
	Service     *service.InstanceService
	SyncService *service.SyncService
}

func NewInstanceHandler(service *service.InstanceService, syncService *service.SyncService) *InstanceHandler {
	return &InstanceHandler{Service: service, SyncService: syncService}
}

type CreateInstanceRequest struct {
	Name     string `json:"name" binding:"required"`
	Host     string `json:"host" binding:"required"`
	Port     int    `json:"port" binding:"required"`
	Type     string `json:"type" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *InstanceHandler) ListInstances(c *gin.Context) {
	instances, err := h.Service.GetAllInstances()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Filter for regular users (remove credentials)
	type UserView struct {
		ID     uuid.UUID `json:"id"`
		Name   string    `json:"name"`
		Host   string    `json:"host"`
		Port   int       `json:"port"`
		Type   string    `json:"type"`
		Status string    `json:"status"`
	}

	result := make([]UserView, len(instances))
	for i, inst := range instances {
		result[i] = UserView{
			ID:     inst.ID,
			Name:   inst.Name,
			Host:   inst.Host,
			Port:   inst.Port,
			Type:   inst.Type,
			Status: inst.Status,
		}
	}

	c.JSON(http.StatusOK, result)
}

func (h *InstanceHandler) AdminListInstances(c *gin.Context) {
	instances, err := h.Service.GetAllInstances()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Admin view includes username and lastSync
	c.JSON(http.StatusOK, instances)
}

func (h *InstanceHandler) CreateInstance(c *gin.Context) {
	var req CreateInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	instance, err := h.Service.CreateInstance(req.Name, req.Host, req.Port, req.Type, req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": instance})
}

func (h *InstanceHandler) UpdateInstance(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	instance, err := h.Service.UpdateInstance(id, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": instance})
}

func (h *InstanceHandler) SyncInstance(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	// Trigger background sync
	go h.SyncService.SyncInstance(id)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Sync started"})
}
