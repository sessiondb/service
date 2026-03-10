// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package handlers

import (
	"net/http"
	"sessiondb/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// FeatureNotifyHandler handles POST /notify-me for "Notify me when this is ready" sign-ups.
type FeatureNotifyHandler struct {
	Service *service.FeatureNotifyService
}

// NewFeatureNotifyHandler returns a new FeatureNotifyHandler.
func NewFeatureNotifyHandler(svc *service.FeatureNotifyService) *FeatureNotifyHandler {
	return &FeatureNotifyHandler{Service: svc}
}

// RegisterRequest is the JSON body for POST /notify-me.
type RegisterNotifyRequest struct {
	FeatureKey string `json:"featureKey" binding:"required"`
}

// Register handles POST /v1/notify-me. Email is taken from the authenticated user (JWT); featureKey from body.
func (h *FeatureNotifyHandler) Register(c *gin.Context) {
	var req RegisterNotifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "featureKey is required"})
		return
	}
	email, _ := c.Get("email")
	emailStr, _ := email.(string)
	if emailStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email required; ensure you are logged in"})
		return
	}
	var userID *uuid.UUID
	if uid, exists := c.Get("userID"); exists {
		if id, ok := uid.(uuid.UUID); ok {
			userID = &id
		}
	}
	created, alreadyRegistered, err := h.Service.RegisterRequest(emailStr, req.FeatureKey, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register"})
		return
	}
	if alreadyRegistered {
		c.JSON(http.StatusOK, gin.H{"message": "You're already on the list. We'll notify you when this feature is ready."})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message":   "Thanks! We'll notify you when this feature is ready.",
		"requestId": created.ID,
	})
}
