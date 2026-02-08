package handlers

import (
	"net/http"
	"sessiondb/internal/models"
	"sessiondb/internal/repository"
	"sessiondb/internal/service"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuditHandler struct {
	Service  *service.AuditService
	UserRepo *repository.UserRepository
}

func NewAuditHandler(service *service.AuditService, userRepo *repository.UserRepository) *AuditHandler {
	return &AuditHandler{Service: service, UserRepo: userRepo}
}

type CreateLogDTO struct {
	User     string `json:"user" binding:"required"`
	Action   string `json:"action" binding:"required"`
	Resource string `json:"resource" binding:"required"`
	Status   string `json:"status" binding:"required"`
}

func (h *AuditHandler) GetLogs(c *gin.Context) {
	logs, err := h.Service.GetLogs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	var response = make([]gin.H, 0)
	for _, l := range logs {
		response = append(response, gin.H{
			"id": l.ID,
			"timestamp": l.Timestamp,
			"user": l.User.Name, // Assuming User relation is loaded and Name is populated
			"session_user": l.SessionUser,
			"action": l.Action,
			"resource": l.Resource,
			"table": l.Table,
			"query": l.Query,
			"status": l.Status,
		})
	}

	c.JSON(http.StatusOK, response)
}

func (h *AuditHandler) CreateLog(c *gin.Context) {
	var req CreateLogDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Lookup user
	user, err := h.UserRepo.FindByName(req.User)
	userID := uuid.Nil
	if err == nil {
		userID = user.ID
	}
	// If user not found, we still log but maybe with Nil userID or handle it? 
	// Frontend sends "user": "admin_mouli".
	
	log := &models.AuditLog{
		Timestamp: time.Now(),
		UserID:    userID,
		Action:    req.Action,
		Resource:  req.Resource,
		Status:    req.Status,
		// Other fields optional or default
	}

	if err := h.Service.Create(log); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"status": "success"})
}
