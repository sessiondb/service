// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package handlers

import (
	"fmt"
	"log"
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

	log.Printf("DEBUG Schema Filter: Instance=%s, User=%s, dbUsername=%s", id, userID, dbUsername)

	privMap := make(map[string]map[string]bool)

	// Direct privileges
	directPrivs, _ := h.MetaRepo.FindPrivilegesByGrantee(id, dbUsername)
	log.Printf("DEBUG Schema Filter: DirectPrivs found = %d", len(directPrivs))
	for _, p := range directPrivs {
		key := fmt.Sprintf("%s.%s", p.Schema, p.Table)
		if privMap[key] == nil {
			privMap[key] = make(map[string]bool)
		}
		privMap[key][p.Privilege] = true
	}

	// Role-inherited privileges
	roles, _ := h.MetaRepo.FindRoleMembershipsByMember(id, dbUsername)
	log.Printf("DEBUG Schema Filter: Roles inherited = %v", roles)
	for _, roleName := range roles {
		rolePrivs, _ := h.MetaRepo.FindPrivilegesByGrantee(id, roleName)
		log.Printf("DEBUG Schema Filter: Role %s privs found = %d", roleName, len(rolePrivs))
		for _, p := range rolePrivs {
			key := fmt.Sprintf("%s.%s", p.Schema, p.Table)
			if privMap[key] == nil {
				privMap[key] = make(map[string]bool)
			}
			privMap[key][p.Privilege] = true
		}
	}

	log.Printf("DEBUG Schema Filter: Final privMap keys = %d", len(privMap))

	// Filter each database's tables: only include those with at least one privilege
	enriched := InstanceSchemaWithPrivileges{
		InstanceID: schema.InstanceID,
	}

	for _, db := range schema.Databases {
		var filteredTables []TableWithPrivileges

		// Check for global or DB-level wildcards
		globalWildcardPrivs := privMap["*.*"]
		dbWildcardPrivs := privMap[fmt.Sprintf("%s.*", db.Database)]

		for _, t := range db.Tables {
			key := fmt.Sprintf("%s.%s", t.Schema, t.Name)

			// Combine privileges from direct, DB-wildcard, and Global-wildcard
			combinedPrivs := make(map[string]bool)
			if privs, ok := privMap[key]; ok {
				for p := range privs {
					combinedPrivs[p] = true
				}
			}
			for p := range dbWildcardPrivs {
				combinedPrivs[p] = true
			}
			for p := range globalWildcardPrivs {
				combinedPrivs[p] = true
			}

			if len(combinedPrivs) > 0 {
				privList := make([]string, 0, len(combinedPrivs))
				for p := range combinedPrivs {
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
