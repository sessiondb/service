// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"sessiondb/internal/models"
	"sessiondb/internal/repository"
	"sessiondb/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ApprovalHandler struct {
	Service      *service.ApprovalService
	InstanceRepo *repository.InstanceRepository
}

// NewApprovalHandler builds an ApprovalHandler with the given service and instance repository (for validating requested items).
func NewApprovalHandler(svc *service.ApprovalService, instanceRepo *repository.InstanceRepository) *ApprovalHandler {
	return &ApprovalHandler{Service: svc, InstanceRepo: instanceRepo}
}

type CreateRequestDTO struct {
	Type                 string                 `json:"type" binding:"required"`
	Description          string                 `json:"description" binding:"required"`
	Justification        string                 `json:"justification" binding:"required"`
	RequestedPermissions []models.Permission    `json:"requestedPermissions"`
	RequestedItems       []models.RequestedItem `json:"requestedItems" binding:"required"`
}

type UpdateRequestStatusDTO struct {
	Status             string              `json:"status" binding:"required"`
	PartialPermissions []models.Permission `json:"partialPermissions"`
	RejectionReason    string              `json:"rejectionReason"`
}

const errPrefixRequestedItems = "requestedItems["

// validateRequestedItems checks that there is at least one item, each with non-empty InstanceID, Database, Table, Privileges, and that each instance exists.
// Returns (errMsg, statusCode). statusCode 0 means valid; 400 for validation/not found; 500 for internal errors.
func (h *ApprovalHandler) validateRequestedItems(items []models.RequestedItem) (string, int) {
	if len(items) == 0 {
		return "at least one requested item is required", http.StatusBadRequest
	}
	for i := range items {
		item := &items[i]
		idx := errPrefixRequestedItems + strconv.Itoa(i) + "]: "
		if item.InstanceID == uuid.Nil {
			return idx + "instanceId is required", http.StatusBadRequest
		}
		if item.Database == "" {
			return idx + "database is required", http.StatusBadRequest
		}
		if item.Table == "" {
			return idx + "table is required", http.StatusBadRequest
		}
		if len(item.Privileges) == 0 {
			return idx + "at least one privilege is required", http.StatusBadRequest
		}
		_, err := h.InstanceRepo.FindByID(item.InstanceID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return idx + "instance not found", http.StatusBadRequest
			}
			return err.Error(), http.StatusInternalServerError
		}
	}
	return "", 0
}

func (h *ApprovalHandler) CreateRequest(c *gin.Context) {
	var req CreateRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	requesterID := c.MustGet("userID").(uuid.UUID)

	if errMsg, code := h.validateRequestedItems(req.RequestedItems); code != 0 {
		c.JSON(code, gin.H{"error": errMsg})
		return
	}

	permissionsJSON, err := json.Marshal(req.RequestedPermissions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize permissions"})
		return
	}
	requestedItemsJSON, err := json.Marshal(req.RequestedItems)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize requested items"})
		return
	}

	request, err := h.Service.CreateRequest(requesterID, req.Type, req.Description, req.Justification, permissionsJSON, requestedItemsJSON)
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
		var requestedItems []models.RequestedItem
		if len(r.RequestedItems) > 0 {
			_ = json.Unmarshal(r.RequestedItems, &requestedItems)
		}

		response = append(response, gin.H{
			"id":                   r.ID,
			"type":                 r.Type,
			"requester":            r.Requester.Name,
			"description":          r.Description,
			"timestamp":            r.CreatedAt, // Frontend expects "timestamp"
			"status":               r.Status,
			"requestedPermissions": permissions,
			"requestedItems":       requestedItems,
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
		reason := req.RejectionReason
		if reason == "" {
			reason = "Rejected by user"
		}
		request, err := h.Service.RejectRequest(id, reviewerID, reason)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, request)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
	}
}
