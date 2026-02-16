package handlers

import (
	"net/http"
	"sessiondb/internal/models"
	"sessiondb/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserHandler struct {
	Service     *service.UserService
	RoleService *service.RoleService
}

func NewUserHandler(service *service.UserService, roleService *service.RoleService) *UserHandler {
	return &UserHandler{Service: service, RoleService: roleService}
}

type CreateUserRequest struct {
	Name           string              `json:"name" binding:"required"`
	Email          string              `json:"email"` // Made optional for frontend compatibility
	Password       string              `json:"password"` // Made optional, will generate default if missing
	Role           string              `json:"role" binding:"required"` // Role name
	Status         string              `json:"status"`
	IsSessionBased bool                `json:"isSessionBased"`
	Permissions    []models.Permission `json:"permissions"`
}

type UpdateUserRequest struct {
	Name           *string             `json:"name"`
	Role           *string             `json:"role"`
	Status         *string             `json:"status"`
	IsSessionBased *bool               `json:"isSessionBased"`
	Permissions    []models.Permission `json:"permissions"`
}

func (h *UserHandler) SyncTabs(c *gin.Context) {
	var tabs []models.QueryTab
	if err := c.ShouldBindJSON(&tabs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.MustGet("userID").(uuid.UUID)

	if err := h.Service.SyncTabs(userID, tabs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Handle missing email
	if req.Email == "" {
		req.Email = req.Name + "@sessiondb.local"
	}

	// Handle missing password
	if req.Password == "" {
		req.Password = "SessionDB!2026" // Default secure password for initial setup
	}

	role, err := h.RoleService.GetRoleByName(req.Role)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role: " + req.Role})
		return
	}

	user := &models.User{
		Name:           req.Name,
		Email:          req.Email,
		RoleID:         role.ID,
		Status:         req.Status,
		IsSessionBased: req.IsSessionBased,
		Permissions:    req.Permissions, 
	}
	
	createdUser, err := h.Service.Create(user, req.Password)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, createdUser)
}

func (h *UserHandler) GetAllUsers(c *gin.Context) {
	users, err := h.Service.GetAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

func (h *UserHandler) GetMe(c *gin.Context) {
	// Get user ID from JWT token (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	user, err := h.Service.GetUserByID(userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	user, err := h.Service.GetUserByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.Service.GetUserByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update fields if provided
	if req.Name != nil { user.Name = *req.Name }
	if req.Status != nil { user.Status = *req.Status }
	if req.Role != nil {
		role, err := h.RoleService.GetRoleByName(*req.Role)
		if err == nil {
			user.RoleID = role.ID
		}
	}
	if req.IsSessionBased != nil { 
		user.IsSessionBased = *req.IsSessionBased 
	}
	
	if req.Permissions != nil {
		user.Permissions = req.Permissions
	}

	if err := h.Service.Update(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := h.Service.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}
