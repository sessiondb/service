package handlers

import (
	"fmt"
	"net/http"
	"sessiondb/internal/apierrors"
	"sessiondb/internal/models"
	"sessiondb/internal/repository"
	"sessiondb/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MetadataHandler struct {
	Service        *service.MetadataService
	MetaRepo       *repository.MetadataRepository
	DBUserCredRepo *repository.DBUserCredentialRepository
}

func NewMetadataHandler(
	svc *service.MetadataService,
	metaRepo *repository.MetadataRepository,
	dbUserCredRepo *repository.DBUserCredentialRepository,
) *MetadataHandler {
	return &MetadataHandler{
		Service:        svc,
		MetaRepo:       metaRepo,
		DBUserCredRepo: dbUserCredRepo,
	}
}

func (h *MetadataHandler) ListDatabases(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, "Invalid instance ID"))
		return
	}

	databases, err := h.Service.GetDatabases(id)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusInternalServerError, apierrors.CodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, databases)
}

func (h *MetadataHandler) ListTables(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, "Invalid instance ID"))
		return
	}

	dbName := c.Param("dbName")
	tables, err := h.Service.GetTables(id, dbName)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusInternalServerError, apierrors.CodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, tables)
}

func (h *MetadataHandler) GetTableDetails(c *gin.Context) {
	idStr := c.Param("tableId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, "Invalid table ID"))
		return
	}

	table, err := h.Service.GetTableDetails(id)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusInternalServerError, apierrors.CodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, table)
}

// TableWithPrivileges wraps a DBTable with the user's allowed operations
type TableWithPrivileges struct {
	models.DBTable
	Privileges []string `json:"privileges"` // e.g. ["SELECT", "INSERT"]
}

// DatabaseSchemaWithPrivileges wraps tables with privilege info
type DatabaseSchemaWithPrivileges struct {
	Database string                `json:"database"`
	Tables   []TableWithPrivileges `json:"tables"`
}

// InstanceSchemaWithPrivileges is the privilege-enriched schema response
type InstanceSchemaWithPrivileges struct {
	InstanceID uuid.UUID                      `json:"instanceId"`
	Databases  []DatabaseSchemaWithPrivileges `json:"databases"`
}

func (h *MetadataHandler) GetInstanceSchema(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		idStr = c.Query("instanceId")
	}

	if idStr == "" {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, "Instance ID is required"))
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, "Invalid instance ID"))
		return
	}

	schema, err := h.Service.GetInstanceSchema(id)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusInternalServerError, apierrors.CodeInternalError, err.Error()))
		return
	}

	// If we have repos and user context, filter by privileges
	userIDRaw, exists := c.Get("userID")
	if !exists || h.MetaRepo == nil || h.DBUserCredRepo == nil {
		// No user context — return all tables unfiltered (backward compat)
		c.JSON(http.StatusOK, schema)
		return
	}

	userID := userIDRaw.(uuid.UUID)

	// Look up the user's DB credential for this instance
	cred, credErr := h.DBUserCredRepo.FindByUserAndInstance(userID, id)
	if credErr != nil || cred == nil {
		// No credential mapping → return unfiltered schema
		c.JSON(http.StatusOK, schema)
		return
	}

	// Build a privilege map: "schema.table" → set of privilege types
	dbUsername := cred.DBUsername
	privMap := make(map[string]map[string]bool)

	// Direct privileges
	directPrivs, _ := h.MetaRepo.FindPrivilegesByGrantee(id, dbUsername)
	for _, p := range directPrivs {
		key := fmt.Sprintf("%s.%s", p.Schema, p.Table)
		if privMap[key] == nil {
			privMap[key] = make(map[string]bool)
		}
		privMap[key][p.Privilege] = true
	}

	// Role-inherited privileges
	roles, _ := h.MetaRepo.FindRoleMembershipsByMember(id, dbUsername)
	for _, roleName := range roles {
		rolePrivs, _ := h.MetaRepo.FindPrivilegesByGrantee(id, roleName)
		for _, p := range rolePrivs {
			key := fmt.Sprintf("%s.%s", p.Schema, p.Table)
			if privMap[key] == nil {
				privMap[key] = make(map[string]bool)
			}
			privMap[key][p.Privilege] = true
		}
	}

	// Filter each database's tables: only include those with at least one privilege
	enriched := InstanceSchemaWithPrivileges{
		InstanceID: schema.InstanceID,
	}

	for _, db := range schema.Databases {
		var filteredTables []TableWithPrivileges
		for _, t := range db.Tables {
			key := fmt.Sprintf("%s.%s", t.Schema, t.Name)
			if privs, ok := privMap[key]; ok && len(privs) > 0 {
				privList := make([]string, 0, len(privs))
				for p := range privs {
					privList = append(privList, p)
				}
				filteredTables = append(filteredTables, TableWithPrivileges{
					DBTable:    t,
					Privileges: privList,
				})
			}
		}
		if len(filteredTables) > 0 {
			enriched.Databases = append(enriched.Databases, DatabaseSchemaWithPrivileges{
				Database: db.Database,
				Tables:   filteredTables,
			})
		}
	}

	if enriched.Databases == nil {
		enriched.Databases = []DatabaseSchemaWithPrivileges{}
	}

	c.JSON(http.StatusOK, enriched)
}
