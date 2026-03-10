//go:build pro

package session

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"sessiondb/internal/apierrors"
	"sessiondb/internal/engine"
)

// Handler handles premium session (ephemeral credential) HTTP endpoints.
type Handler struct {
	SessionEngine engine.SessionEngine
}

// NewHandler returns a new session Handler.
func NewHandler(sessionEngine engine.SessionEngine) *Handler {
	return &Handler{SessionEngine: sessionEngine}
}

// StartRequest is the request body for starting a session.
type StartRequest struct {
	InstanceID string `json:"instanceId" binding:"required"`
	TTLMinutes int    `json:"ttlMinutes"` // optional; default 60
}

// StartResponse is the response for starting a session.
type StartResponse struct {
	SessionID string `json:"sessionId"`
	DBUsername string `json:"dbUsername"`
	Password  string `json:"password"` // temporary; client must store securely for the session
	ExpiresAt string `json:"expiresAt"`
}

// Start starts an ephemeral credential session for the user on the given instance.
func (h *Handler) Start(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	var req StartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, err.Error()))
		return
	}
	instanceID, err := uuid.Parse(req.InstanceID)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, "Invalid instance ID"))
		return
	}
	ttl := 60 * time.Minute
	if req.TTLMinutes > 0 {
		ttl = time.Duration(req.TTLMinutes) * time.Minute
	}

	session, password, err := h.SessionEngine.StartSession(c.Request.Context(), userID, instanceID, ttl)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, StartResponse{
		SessionID:  session.ID.String(),
		DBUsername: session.DBUsername,
		Password:   password,
		ExpiresAt:  session.ExpiresAt.Format(time.RFC3339),
	})
}

// End ends a session by ID. Only the session owner can end it.
func (h *Handler) End(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	sessionIDStr := c.Param("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, "Invalid session ID"))
		return
	}

	session, err := h.SessionEngine.GetSessionByID(c.Request.Context(), sessionID)
	if err != nil || session == nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusNotFound, apierrors.CodeNotFound, "Session not found"))
		return
	}
	if session.UserID != userID {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusForbidden, apierrors.CodeForbidden, "Not allowed to end this session"))
		return
	}

	if err := h.SessionEngine.EndSession(c.Request.Context(), sessionID); err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusInternalServerError, apierrors.CodeInternalError, err.Error()))
		return
	}
	c.Status(http.StatusNoContent)
}

// GetActive returns the active session for the user on the given instance (query: instanceId).
func (h *Handler) GetActive(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	instanceIDStr := c.Query("instanceId")
	if instanceIDStr == "" {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, "instanceId required"))
		return
	}
	instanceID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, "Invalid instance ID"))
		return
	}

	session, err := h.SessionEngine.GetActiveSession(c.Request.Context(), userID, instanceID)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusInternalServerError, apierrors.CodeInternalError, err.Error()))
		return
	}
	if session == nil {
		c.JSON(http.StatusOK, gin.H{"active": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"active":     true,
		"sessionId":  session.ID.String(),
		"dbUsername": session.DBUsername,
		"expiresAt":  session.ExpiresAt.Format(time.RFC3339),
	})
}
