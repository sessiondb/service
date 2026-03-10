//go:build pro

package report

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"sessiondb/internal/apierrors"
	"sessiondb/internal/engine"
	"sessiondb/internal/models"
)

// CreateReportBody is the request body for creating a report definition (JSONB fields as JSON).
type CreateReportBody struct {
	Name             string          `json:"name" binding:"required"`
	Description      string          `json:"description"`
	DataSources      json.RawMessage `json:"dataSources"`
	Filters          json.RawMessage `json:"filters"`
	ScheduleCron     *string         `json:"scheduleCron"`
	DeliveryChannels json.RawMessage `json:"deliveryChannels"`
	Format           string          `json:"format"`
	IsEnabled        *bool           `json:"isEnabled"`
}

// Handler handles premium report HTTP endpoints.
type Handler struct {
	ReportEngine engine.ReportEngine
}

// NewHandler returns a new report Handler.
func NewHandler(reportEngine engine.ReportEngine) *Handler {
	return &Handler{ReportEngine: reportEngine}
}

// TenantIDFromContext returns tenant ID; placeholder until multi-tenant is wired.
func TenantIDFromContext(c *gin.Context) uuid.UUID {
	return uuid.Nil
}

// CreateDefinition creates a report definition.
// POST /reports/definitions
func (h *Handler) CreateDefinition(c *gin.Context) {
	tenantID := TenantIDFromContext(c)
	userID := c.MustGet("userID").(uuid.UUID)
	var body CreateReportBody
	if err := c.ShouldBindJSON(&body); err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, err.Error()))
		return
	}
	def := models.ReportDefinition{
		TenantID:         tenantID,
		CreatedBy:        userID,
		Name:             body.Name,
		Description:      body.Description,
		DataSources:      body.DataSources,
		Filters:          body.Filters,
		ScheduleCron:     body.ScheduleCron,
		DeliveryChannels: body.DeliveryChannels,
		Format:           body.Format,
		IsEnabled:        true,
	}
	if def.Format == "" {
		def.Format = "csv"
	}
	if body.IsEnabled != nil {
		def.IsEnabled = *body.IsEnabled
	}
	if err := h.ReportEngine.CreateDefinition(c.Request.Context(), &def); err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusInternalServerError, apierrors.CodeInternalError, err.Error()))
		return
	}
	c.JSON(http.StatusCreated, def)
}

// ListDefinitions lists report definitions for the tenant.
// GET /reports/definitions
func (h *Handler) ListDefinitions(c *gin.Context) {
	tenantID := TenantIDFromContext(c)
	defs, err := h.ReportEngine.ListDefinitions(c.Request.Context(), tenantID)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusInternalServerError, apierrors.CodeInternalError, err.Error()))
		return
	}
	c.JSON(http.StatusOK, defs)
}

// UpdateReportBody is the request body for updating a report definition.
type UpdateReportBody struct {
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	DataSources      json.RawMessage `json:"dataSources"`
	Filters          json.RawMessage `json:"filters"`
	ScheduleCron     *string         `json:"scheduleCron"`
	DeliveryChannels json.RawMessage `json:"deliveryChannels"`
	Format           string          `json:"format"`
	IsEnabled        *bool           `json:"isEnabled"`
}

// GetDefinition returns one definition by ID.
// GET /reports/definitions/:id
func (h *Handler) GetDefinition(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, "Invalid definition ID"))
		return
	}
	def, err := h.ReportEngine.GetDefinition(c.Request.Context(), id)
	if err != nil || def == nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusNotFound, apierrors.CodeNotFound, "Definition not found"))
		return
	}
	c.JSON(http.StatusOK, def)
}

// UpdateDefinition updates a definition.
// PUT /reports/definitions/:id
func (h *Handler) UpdateDefinition(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, "Invalid definition ID"))
		return
	}
	def, err := h.ReportEngine.GetDefinition(c.Request.Context(), id)
	if err != nil || def == nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusNotFound, apierrors.CodeNotFound, "Definition not found"))
		return
	}
	var body UpdateReportBody
	if err := c.ShouldBindJSON(&body); err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, err.Error()))
		return
	}
	if body.Name != "" {
		def.Name = body.Name
	}
	if body.Description != "" || body.Name != "" {
		def.Description = body.Description
	}
	if len(body.DataSources) > 0 {
		def.DataSources = body.DataSources
	}
	if len(body.Filters) > 0 {
		def.Filters = body.Filters
	}
	if body.ScheduleCron != nil {
		def.ScheduleCron = body.ScheduleCron
	}
	if len(body.DeliveryChannels) > 0 {
		def.DeliveryChannels = body.DeliveryChannels
	}
	if body.Format != "" {
		def.Format = body.Format
	}
	if body.IsEnabled != nil {
		def.IsEnabled = *body.IsEnabled
	}
	if err := h.ReportEngine.UpdateDefinition(c.Request.Context(), def); err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusInternalServerError, apierrors.CodeInternalError, err.Error()))
		return
	}
	c.JSON(http.StatusOK, def)
}

// DeleteDefinition deletes a definition.
// DELETE /reports/definitions/:id
func (h *Handler) DeleteDefinition(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, "Invalid definition ID"))
		return
	}
	if err := h.ReportEngine.DeleteDefinition(c.Request.Context(), id); err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusInternalServerError, apierrors.CodeInternalError, err.Error()))
		return
	}
	c.Status(http.StatusNoContent)
}

// RunReport runs a report by definition ID.
// POST /reports/definitions/:id/run
func (h *Handler) RunReport(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, "Invalid definition ID"))
		return
	}
	ex, err := h.ReportEngine.RunReport(c.Request.Context(), id)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusInternalServerError, apierrors.CodeInternalError, err.Error()))
		return
	}
	c.JSON(http.StatusAccepted, ex)
}

// ListExecutions lists executions for a definition.
// GET /reports/definitions/:id/executions
func (h *Handler) ListExecutions(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, "Invalid definition ID"))
		return
	}
	list, err := h.ReportEngine.ListExecutions(c.Request.Context(), id)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusInternalServerError, apierrors.CodeInternalError, err.Error()))
		return
	}
	c.JSON(http.StatusOK, list)
}
