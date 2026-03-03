// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package handlers

import (
	"net/http"
	"sessiondb/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	Service *service.AuthService
}

func NewAuthHandler(service *service.AuthService) *AuthHandler {
	return &AuthHandler{Service: service}
}

type LoginRequest struct {
	Username string `json:"username"` // Frontend sends username for admin, email for others maybe? or just generic "username" field mapping to email
	// The frontend doc says: "username": "admin_mouli", "password": "..."
	// But our backend uses email. For now, let's accept "username" and treat it as email if it looks like one, or handle it.
	// Actually, the previous implementation used 'email'. The doc says 'username'.
	// Let's assume 'username' field in JSON maps to Email in our DB for now, or we need to add Username field or use DBUsername.
	// The User model has 'Name', 'Email', 'DBUsername'.
	// I'll add 'Username' to the struct and try to find by Email OR DBUsername.
	// For MVP, lets assume username == email or DBUsername.
	Email    string `json:"email"` // Keep for backward compat if needed, but docs say username
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type SSORequest struct {
	Provider string `json:"provider" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Support both username and email fields from JSON
	identifier := req.Username
	if identifier == "" {
		identifier = req.Email
	}

	// Service currently only supports FindByEmail.
	// TODO: Update Service/Repo to support identifying by DBUsername or Username.
	// For now, passing identifier as email.
	user, tenantMap, token, refreshToken, err := h.Service.Login(identifier, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"token":         token,
			"refresh_token": refreshToken,
			"user": gin.H{
				"id":              user.ID,
				"name":            user.Name,
				"email":           user.Email,
				"role":            user.Role.Name,
				"status":          user.Status,
				"isSessionBased":  user.IsSessionBased,
				"lastLogin":       user.LastLogin,
				"permissions":     user.Permissions,
				"rbacPermissions": user.RBACPermissions,
				"tenantFeatures":  tenantMap,
				"savedScripts":    user.SavedScripts,
				"queryTabs":       user.QueryTabs,
			},
		},
	})
}

func (h *AuthHandler) SSO(c *gin.Context) {
	var req SSORequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Mock SSO response
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "SSO not implemented yet"})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.Service.Register(req.Name, req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":    user.ID,
		"email": user.Email,
		"name":  user.Name,
	})
}
