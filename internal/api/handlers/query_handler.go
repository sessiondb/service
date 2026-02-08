package handlers

import (
	"net/http"
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
	Database string `json:"database"` // Optional now
	Query    string `json:"query" binding:"required"`
}

type SaveScriptRequest struct {
	Name     string `json:"name" binding:"required"`
	Query    string `json:"query" binding:"required"`
	IsPublic bool   `json:"is_public"` // Frontend might still send it? Doc removed it. Let's keep it optional.
}

func (h *QueryHandler) ExecuteQuery(c *gin.Context) {
	var req ExecuteQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.MustGet("userID").(uuid.UUID)

	result, err := h.Service.ExecuteQuery(userID, req.Database, req.Query)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
