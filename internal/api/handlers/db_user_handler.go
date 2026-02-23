package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"sessiondb/internal/apierrors"
	"sessiondb/internal/models"
	"sessiondb/internal/repository"
	"sessiondb/internal/service"
	"sessiondb/internal/utils"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
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

// LinkDBUserRequest represents the request body for linking a DB user
type LinkDBUserRequest struct {
	PlatformUserID string `json:"platform_user_id" binding:"required"`
}

// VerifyCredentialsRequest represents the request body for verifying DB credentials
type VerifyCredentialsRequest struct {
	InstanceID string `json:"instanceId" binding:"required"`
	DBPassword string `json:"dbPassword" binding:"required"`
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

// LinkDBUser handles PUT /db-users/:id/link
// The :id refers to the DBEntity ID of an existing user on the database.
// It maps the DB user to a platform user by creating a dummy DBUserCredential.
func (h *DBUserHandler) LinkDBUser(c *gin.Context) {
	idStr := c.Param("id")
	entityID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var req LinkDBUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	platformUserID, err := uuid.Parse(req.PlatformUserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid platform user ID format"})
		return
	}

	// 1. Verify the DBEntity exists
	var entity *models.DBEntity
	for _, _ = range []string{"USER", "ROLE"} {
		// Just a quick check to ensure the entity is valid. Could use a direct DB query here,
		// but since we only have FindEntitiesByInstance in repo, it's easier to check directly.
		var ent models.DBEntity
		if rErr := h.MetaRepo.DB.First(&ent, "id = ?", entityID).Error; rErr == nil {
			entity = &ent
			break
		}
	}

	if entity == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "DB user entity not found"})
		return
	}

	// 2. We don't have the password for existing unmanaged users.
	// The DBUserCredential record acts as a mapping. It handles authorization, but if auth is required,
	// this approach assumes they will authenticate with their own credentials or admin credentials.

	// Check if already linked to this user
	existing, _ := h.ProvisioningService.DBUserCredRepo.FindByUserAndInstance(platformUserID, entity.InstanceID)
	if existing != nil && existing.DBUsername == entity.Name {
		c.JSON(http.StatusOK, gin.H{"message": "Already linked", "credentialId": existing.ID})
		return
	}

	// Delete old mapping for this user + instance if requested
	if existing != nil {
		_ = h.ProvisioningService.DBUserCredRepo.Delete(existing.ID)
	}

	// 3. Create the credential mapping
	cred := &models.DBUserCredential{
		UserID:     platformUserID,
		InstanceID: entity.InstanceID,
		DBUsername: entity.Name,
		DBPassword: "", // Empty password - signals it's an unmanaged linked user
		Status:     "active",
		Role:       "custom", // Or derive from existing
	}

	if err := h.ProvisioningService.DBUserCredRepo.Create(cred); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to link user: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User linked successfully", "credentialId": cred.ID})
}

// VerifyCredentials handles POST /db-credentials/verify
// It receives a password from the frontend, uses it to connect to the DB
// alongside the user's mapped DBUsername, and if successful, saves the encrypted password.
func (h *DBUserHandler) VerifyCredentials(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	var req VerifyCredentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	instanceID, err := uuid.Parse(req.InstanceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instance ID format"})
		return
	}

	// Find the mapped credential for this user & instance
	cred, err := h.ProvisioningService.DBUserCredRepo.FindByUserAndInstance(userID, instanceID)
	if err != nil || cred == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No database user mapping found for this instance"})
		return
	}

	// We need the instance metadata to connect (Host, Port, etc.)
	instance, err := h.InstanceRepo.FindByID(instanceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve instance details"})
		return
	}

	// Try to connect using the provided credentials
	dsn := ""
	if instance.Type == "postgres" {
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=disable",
			instance.Host, instance.Port, cred.DBUsername, req.DBPassword)
	} else if instance.Type == "mysql" {
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/",
			cred.DBUsername, req.DBPassword, instance.Host, instance.Port)
	} else {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusBadRequest, apierrors.CodeInvalidRequest, "Unsupported database engine"))
		return
	}

	driverName := instance.Type

	// Create temporary connection purely to ping
	db, err := sql.Open(driverName, dsn)
	if err == nil {
		err = db.Ping()
		db.Close()
	}

	if err != nil {
		// Log the underlying database error if needed, but return generic invalid credentials code
		apierrors.Respond(c, apierrors.ErrUserCredsInvalid)
		return
	}

	// Connection successful. Encrypt and save the password.
	encryptedPass, err := utils.EncryptPassword(req.DBPassword)
	if err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusInternalServerError, apierrors.CodeInternalError, "Failed to encrypt credentials"))
		return
	}

	cred.DBPassword = encryptedPass
	// Update specifically the db_password field to bypass json:"-" gorm save ignoring
	if err := h.ProvisioningService.DBUserCredRepo.DB.Model(cred).UpdateColumn("db_password", encryptedPass).Error; err != nil {
		apierrors.Respond(c, apierrors.NewAppError(http.StatusInternalServerError, apierrors.CodeInternalError, "Failed to save credentials"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Credentials verified and saved successfully"})
}
