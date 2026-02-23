package handlers

import (
	"fmt"
	"net/http"
	"sessiondb/internal/models"
	"sessiondb/internal/repository"
	"sessiondb/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DBUserHandler struct {
	ProvisioningService *service.DBUserProvisioningService
	MetaRepo            *repository.MetadataRepository
	InstanceRepo        *repository.InstanceRepository
}

func NewDBUserHandler(
	provisioningService *service.DBUserProvisioningService,
	metaRepo *repository.MetadataRepository,
	instanceRepo *repository.InstanceRepository,
) *DBUserHandler {
	return &DBUserHandler{
		ProvisioningService: provisioningService,
		MetaRepo:            metaRepo,
		InstanceRepo:        instanceRepo,
	}
}

// DBUserResponse represents the API response for a DB User
type DBUserResponse struct {
	ID               uuid.UUID             `json:"id"`
	Username         string                `json:"username"`
	InstanceID       uuid.UUID             `json:"instanceId"`
	InstanceName     string                `json:"instanceName"`
	Role             string                `json:"role"`
	Status           string                `json:"status"`
	CreatedAt        string                `json:"created_at"`
	IsManaged        bool                  `json:"isManaged"` // true = provisioned by this system
	RolePrivileges   []DBPrivilegeResponse `json:"rolePrivileges"`
	DirectPrivileges []DBPrivilegeResponse `json:"directPrivileges"`
}

type DBPrivilegeResponse struct {
	Object    string `json:"object"` // schema.table
	Type      string `json:"type"`   // SELECT, etc.
	Grantable bool   `json:"grantable"`
}

// GetDBUsers handles GET /db-users
// It returns all real DB users synced from target databases (DBEntity),
// enriched with managed credential info (role, status) where available.
func (h *DBUserHandler) GetDBUsers(c *gin.Context) {
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

	entities, err := h.MetaRepo.FindEntitiesByInstance(instanceID, "USER")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Build a lookup map: instanceID -> instance name
	instanceNames := map[uuid.UUID]string{}

	// Build a lookup map: (instanceID, dbUsername) -> DBUserCredential for enrichment
	allCreds, _ := h.ProvisioningService.DBUserCredRepo.FindAll()
	credMap := map[string]*models.DBUserCredential{}
	for i := range allCreds {
		key := fmt.Sprintf("%s|%s", allCreds[i].InstanceID, allCreds[i].DBUsername)
		credMap[key] = &allCreds[i]
	}

	var response []DBUserResponse
	for _, entity := range entities {
		// Resolve instance name (cache it)
		if _, ok := instanceNames[entity.InstanceID]; !ok {
			inst, instErr := h.InstanceRepo.FindByID(entity.InstanceID)
			if instErr == nil {
				instanceNames[entity.InstanceID] = inst.Name
			} else {
				instanceNames[entity.InstanceID] = entity.InstanceID.String()
			}
		}

		role := "unmanaged"
		status := "active" // If it exists in the DB it's active
		isManaged := false
		createdAt := entity.CreatedAt.Format("2006-01-02 15:04")

		// Enrich from credential if this user was provisioned by our system
		key := fmt.Sprintf("%s|%s", entity.InstanceID, entity.Name)
		if cred, ok := credMap[key]; ok {
			isManaged = true
			status = cred.Status
			createdAt = cred.CreatedAt.Format("2006-01-02 15:04")
			if cred.Role != "" {
				role = cred.Role
			} else {
				role = "custom"
			}
		}

		// Fetch Privileges
		directPrivs, _ := h.MetaRepo.FindPrivilegesByGrantee(entity.InstanceID, entity.Name)
		directResp := []DBPrivilegeResponse{}
		for _, p := range directPrivs {
			directResp = append(directResp, DBPrivilegeResponse{
				Object:    fmt.Sprintf("%s.%s", p.Schema, p.Table),
				Type:      p.Privilege,
				Grantable: p.IsGrantable,
			})
		}

		// Role Privileges (one level of nesting for now, common in PG/MySQL)
		rolePrivsResp := []DBPrivilegeResponse{}
		roles, _ := h.MetaRepo.FindRoleMembershipsByMember(entity.InstanceID, entity.Name)
		for _, rName := range roles {
			rPrivs, _ := h.MetaRepo.FindPrivilegesByGrantee(entity.InstanceID, rName)
			for _, p := range rPrivs {
				rolePrivsResp = append(rolePrivsResp, DBPrivilegeResponse{
					Object:    fmt.Sprintf("%s.%s", p.Schema, p.Table),
					Type:      p.Privilege,
					Grantable: p.IsGrantable,
				})
			}
		}

		response = append(response, DBUserResponse{
			ID:               entity.ID,
			Username:         entity.Name,
			InstanceID:       entity.InstanceID,
			InstanceName:     instanceNames[entity.InstanceID],
			Role:             role,
			Status:           status,
			CreatedAt:        createdAt,
			IsManaged:        isManaged,
			DirectPrivileges: directResp,
			RolePrivileges:   rolePrivsResp,
		})
	}

	if response == nil {
		response = []DBUserResponse{}
	}

	c.JSON(http.StatusOK, response)
}

// UpdateRoleRequest represents the request body for updating a role
type UpdateRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

// UpdateDBUserRole handles PUT /db-users/:id
// The :id refers to the DBUserCredential ID (managed users only).
func (h *DBUserHandler) UpdateDBUserRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.ProvisioningService.UpdateUserRole(id, req.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to update role: %v", err)})
		return
	}

	// Fetch updated credential to return
	cred, err := h.ProvisioningService.DBUserCredRepo.FindByID(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "Role updated successfully"})
		return
	}

	response := DBUserResponse{
		ID:           cred.ID,
		Username:     cred.DBUsername,
		InstanceID:   cred.InstanceID,
		InstanceName: cred.Instance.Name,
		Role:         cred.Role,
		Status:       cred.Status,
		CreatedAt:    cred.CreatedAt.Format("2006-01-02 15:04"),
		IsManaged:    true,
	}

	c.JSON(http.StatusOK, response)
}
