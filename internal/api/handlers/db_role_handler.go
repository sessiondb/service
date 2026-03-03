// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package handlers

import (
	"fmt"
	"net/http"
	"sessiondb/internal/repository"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DBRoleHandler struct {
	MetaRepo     *repository.MetadataRepository
	InstanceRepo *repository.InstanceRepository
}

func NewDBRoleHandler(
	metaRepo *repository.MetadataRepository,
	instanceRepo *repository.InstanceRepository,
) *DBRoleHandler {
	return &DBRoleHandler{
		MetaRepo:     metaRepo,
		InstanceRepo: instanceRepo,
	}
}

type DBRoleResponse struct {
	ID           uuid.UUID             `json:"id"`
	Name         string                `json:"name"`
	DBKey        string                `json:"dbKey"`
	InstanceID   uuid.UUID             `json:"instanceId"`
	InstanceName string                `json:"instanceName"`
	MemberCount  int64                 `json:"memberCount"`
	Privileges   []DBPrivilegeResponse `json:"privileges"`
	IsSystemRole bool                  `json:"isSystemRole"`
	CreatedAt    string                `json:"createdAt"`
}

// GetDBRoles handles GET /db-roles?instanceId=<id>
func (h *DBRoleHandler) GetDBRoles(c *gin.Context) {
	instanceIDStr := c.Query("instanceId")
	if instanceIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "instanceId is required"})
		return
	}

	instanceID, parseErr := uuid.Parse(instanceIDStr)
	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instanceId"})
		return
	}

	// Fetch all ROLE entities for this instance
	entities, err := h.MetaRepo.FindEntitiesByInstance(instanceID, "ROLE")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Resolve instance name
	instanceName := instanceID.String()
	if inst, instErr := h.InstanceRepo.FindByID(instanceID); instErr == nil {
		instanceName = inst.Name
	}

	var response []DBRoleResponse
	for _, entity := range entities {
		// Privileges are stored by db-native name (entity.DBKey)
		grantee := entity.DBKey
		if grantee == "" {
			grantee = entity.Name
		}

		privs, _ := h.MetaRepo.FindPrivilegesByGrantee(instanceID, grantee)
		privResp := make([]DBPrivilegeResponse, 0, len(privs))
		for _, p := range privs {
			privResp = append(privResp, DBPrivilegeResponse{
				Object:    fmt.Sprintf("%s.%s", p.Schema, p.Table),
				Type:      p.Privilege,
				Grantable: p.IsGrantable,
			})
		}

		// Count members that have this role
		memberCount, _ := h.MetaRepo.CountMembersByRole(instanceID, grantee)

		// Determine if this is a system role
		isSystem := isSystemRole(grantee)

		response = append(response, DBRoleResponse{
			ID:           entity.ID,
			Name:         entity.Name,
			DBKey:        entity.DBKey,
			InstanceID:   entity.InstanceID,
			InstanceName: instanceName,
			MemberCount:  memberCount,
			Privileges:   privResp,
			IsSystemRole: isSystem,
			CreatedAt:    entity.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	if response == nil {
		response = []DBRoleResponse{}
	}

	c.JSON(http.StatusOK, response)
}

// isSystemRole heuristic: pg_ prefix (Postgres) or mysql.* system accounts
func isSystemRole(dbKey string) bool {
	lower := strings.ToLower(dbKey)
	return strings.HasPrefix(lower, "pg_") ||
		lower == "public" ||
		lower == "mysql.sys" ||
		lower == "mysql.session" ||
		lower == "mysql.infoschema"
}
