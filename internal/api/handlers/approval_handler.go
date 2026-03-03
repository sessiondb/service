// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package handlers

import (
	"encoding/json"
	"net/http"
	"sessiondb/internal/models"
	"sessiondb/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ApprovalHandler struct {
	Service *service.ApprovalService
}

func NewApprovalHandler(service *service.ApprovalService) *ApprovalHandler {
	return &ApprovalHandler{Service: service}
}

type CreateRequestDTO struct {
	Type                 string              `json:"type" binding:"required"`
	Description          string              `json:"description" binding:"required"`
	Justification        string              `json:"justification" binding:"required"`
	RequestedPermissions []models.Permission `json:"requestedPermissions"` // Using model, but might need custom DTO if model differs
}

type UpdateRequestStatusDTO struct {
	Status             string              `json:"status" binding:"required"`
	PartialPermissions []models.Permission `json:"partialPermissions"`
}

func (h *ApprovalHandler) CreateRequest(c *gin.Context) {
	var req CreateRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	requesterID := c.MustGet("userID").(uuid.UUID)

	permissionsJSON, err := json.Marshal(req.RequestedPermissions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize permissions"})
		return
	}

	request, err := h.Service.CreateRequest(requesterID, req.Type, req.Description, req.Justification, permissionsJSON)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, request)
}

func (h *ApprovalHandler) GetRequests(c *gin.Context) {
	requests, err := h.Service.GetAllRequests()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var response = make([]gin.H, 0)
	for _, r := range requests {
		var permissions []models.Permission
		if len(r.RequestedPermissions) > 0 {
			_ = json.Unmarshal(r.RequestedPermissions, &permissions)
		}

		response = append(response, gin.H{
			"id":                   r.ID,
			"type":                 r.Type,
			"requester":            r.Requester.Name,
			"description":          r.Description,
			"timestamp":            r.CreatedAt, // Frontend expects "timestamp"
			"status":               r.Status,
			"requestedPermissions": permissions,
		})
	}

	c.JSON(http.StatusOK, response)
}

func (h *ApprovalHandler) UpdateRequestStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var req UpdateRequestStatusDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reviewerID := c.MustGet("userID").(uuid.UUID)

	if req.Status == "approved" {
		request, err := h.Service.ApproveRequest(id, reviewerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, request)
	} else if req.Status == "rejected" {
		request, err := h.Service.RejectRequest(id, reviewerID, "Rejected by user") // Reason?
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, request)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
	}
}
