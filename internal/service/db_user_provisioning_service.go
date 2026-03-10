// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package service

import (
	"database/sql"
	"fmt"
	"sessiondb/internal/dialect"
	"sessiondb/internal/models"
	"sessiondb/internal/repository"
	"sessiondb/internal/utils"
	"strings"

	"github.com/google/uuid"
)

type DBUserProvisioningService struct {
	DBUserCredRepo *repository.DBUserCredentialRepository
	InstanceRepo   *repository.InstanceRepository
}

func NewDBUserProvisioningService(
	dbUserCredRepo *repository.DBUserCredentialRepository,
	instanceRepo *repository.InstanceRepository,
) *DBUserProvisioningService {
	return &DBUserProvisioningService{
		DBUserCredRepo: dbUserCredRepo,
		InstanceRepo:   instanceRepo,
	}
}

// GenerateDBUsername generates a database username based on naming convention
// Format: {prefix}_{role}_{name}_{suffix}
// Example: sdb_dev_john_perm or sdb_analyst_jane_temp
func (s *DBUserProvisioningService) GenerateDBUsername(user *models.User) string {
	prefix := "sdb" // TODO: Make configurable

	// Sanitize name (remove spaces, special chars)
	name := strings.ToLower(user.Name)
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, ".", "")
	name = strings.ReplaceAll(name, "-", "_")

	// Get role name (sanitized)
	role := strings.ToLower(user.Role.Name)
	role = strings.ReplaceAll(role, " ", "")

	// Determine suffix based on user type
	suffix := "perm"
	if user.IsSessionBased {
		suffix = "temp"
	}

	return fmt.Sprintf("%s_%s_%s_%s", prefix, role, name, suffix)
}

// ProvisionDBUser creates a database user on the target instance
func (s *DBUserProvisioningService) ProvisionDBUser(user *models.User, instance *models.DBInstance) (*models.DBUserCredential, error) {
	// Check if credential already exists
	existing, err := s.DBUserCredRepo.FindByUserAndInstance(user.ID, instance.ID)
	if err == nil && existing != nil {
		return existing, nil // Already provisioned
	}

	// Generate username and password
	dbUsername := s.GenerateDBUsername(user)
	plainPassword, err := utils.GenerateSecurePassword(24)
	if err != nil {
		return nil, fmt.Errorf("failed to generate password: %w", err)
	}

	// Encrypt password
	encryptedPassword, err := utils.EncryptPassword(plainPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt password: %w", err)
	}

	// Create user on database
	if err := s.createDBUserOnInstance(instance, dbUsername, plainPassword); err != nil {
		return nil, fmt.Errorf("failed to create DB user: %w", err)
	}

	// Store credential
	cred := &models.DBUserCredential{
		UserID:     user.ID,
		InstanceID: instance.ID,
		DBUsername: dbUsername,
		DBPassword: encryptedPassword,
		Status:     "active",
	}

	// Set expiry for session-based users
	if user.IsSessionBased && user.SessionExpiresAt != nil {
		cred.ExpiresAt = user.SessionExpiresAt
	}

	if err := s.DBUserCredRepo.Create(cred); err != nil {
		return nil, fmt.Errorf("failed to store credential: %w", err)
	}

	return cred, nil
}

// createDBUserOnInstance creates the actual database user using the dialect for the instance type.
func (s *DBUserProvisioningService) createDBUserOnInstance(instance *models.DBInstance, username, password string) error {
	d, err := dialect.GetDialect(instance.Type)
	if err != nil {
		return err
	}
	dsn := d.BuildAdminDSN(instance)
	db, err := sql.Open(d.DriverName(), dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to instance: %w", err)
	}
	defer db.Close()
	if _, err := db.Exec(d.CreateUserSQL(username, password)); err != nil {
		return fmt.Errorf("failed to execute CREATE USER: %w", err)
	}
	return nil
}

// GrantPermissions grants permissions to a DB user using the instance's dialect.
func (s *DBUserProvisioningService) GrantPermissions(cred *models.DBUserCredential, permissions []models.Permission) error {
	instance, err := s.InstanceRepo.FindByID(cred.InstanceID)
	if err != nil {
		return fmt.Errorf("failed to find instance: %w", err)
	}
	d, err := dialect.GetDialect(instance.Type)
	if err != nil {
		return err
	}
	dsn := d.BuildAdminDSN(instance)
	db, err := sql.Open(d.DriverName(), dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to instance: %w", err)
	}
	defer db.Close()
	for _, perm := range permissions {
		if err := s.grantPermission(db, d, cred.DBUsername, perm); err != nil {
			return fmt.Errorf("failed to grant permission: %w", err)
		}
	}
	return nil
}

// grantPermission grants a single permission using the dialect to build SQL.
func (s *DBUserProvisioningService) grantPermission(db *sql.DB, d dialect.DatabaseDialect, username string, perm models.Permission) error {
	schema := "public"
	if d.Type() == "mysql" {
		schema = perm.Database
	}
	grantSQL := d.GrantTableSQL(username, perm.Database, schema, perm.Table, perm.Privileges)
	if _, err := db.Exec(grantSQL); err != nil {
		return fmt.Errorf("failed to execute GRANT: %w", err)
	}
	return nil
}

// RevokePermissions revokes permissions from a DB user using the instance's dialect.
func (s *DBUserProvisioningService) RevokePermissions(cred *models.DBUserCredential, permissions []models.Permission) error {
	instance, err := s.InstanceRepo.FindByID(cred.InstanceID)
	if err != nil {
		return fmt.Errorf("failed to find instance: %w", err)
	}
	d, err := dialect.GetDialect(instance.Type)
	if err != nil {
		return err
	}
	dsn := d.BuildAdminDSN(instance)
	db, err := sql.Open(d.DriverName(), dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to instance: %w", err)
	}
	defer db.Close()
	for _, perm := range permissions {
		if err := s.revokePermission(db, d, cred.DBUsername, perm); err != nil {
			return fmt.Errorf("failed to revoke permission: %w", err)
		}
	}
	return nil
}

func (s *DBUserProvisioningService) revokePermission(db *sql.DB, d dialect.DatabaseDialect, username string, perm models.Permission) error {
	schema := "public"
	if d.Type() == "mysql" {
		schema = perm.Database
	}
	revokeSQL := d.RevokeTableSQL(username, perm.Database, schema, perm.Table, perm.Privileges)
	if _, err := db.Exec(revokeSQL); err != nil {
		return fmt.Errorf("failed to execute REVOKE: %w", err)
	}
	return nil
}

// DeprovisionDBUser drops the database user using the instance's dialect.
func (s *DBUserProvisioningService) DeprovisionDBUser(cred *models.DBUserCredential) error {
	instance, err := s.InstanceRepo.FindByID(cred.InstanceID)
	if err != nil {
		return fmt.Errorf("failed to find instance: %w", err)
	}
	d, err := dialect.GetDialect(instance.Type)
	if err != nil {
		return err
	}
	dsn := d.BuildAdminDSN(instance)
	db, err := sql.Open(d.DriverName(), dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to instance: %w", err)
	}
	defer db.Close()
	if _, err := db.Exec(d.DropUserSQL(cred.DBUsername)); err != nil {
		return fmt.Errorf("failed to drop user: %w", err)
	}

	// Update credential status
	if err := s.DBUserCredRepo.UpdateStatus(cred.ID, "revoked"); err != nil {
		return fmt.Errorf("failed to update credential status: %w", err)
	}

	return nil
}

// UpdateUserRole updates the role of a DB user
func (s *DBUserProvisioningService) UpdateUserRole(credentialID uuid.UUID, newRole string) error {
	cred, err := s.DBUserCredRepo.FindByID(credentialID)
	if err != nil {
		return fmt.Errorf("failed to find credential: %w", err)
	}

	instance, err := s.InstanceRepo.FindByID(cred.InstanceID)
	if err != nil {
		return fmt.Errorf("failed to find instance: %w", err)
	}

	// 1. Get new permissions
	newPermissions := s.getPermissionsForRole(newRole, instance.Type)
	if len(newPermissions) == 0 {
		return fmt.Errorf("invalid or empty role: %s", newRole)
	}

	d, err := dialect.GetDialect(instance.Type)
	if err != nil {
		return fmt.Errorf("failed to get dialect: %w", err)
	}
	dsn := d.BuildAdminDSN(instance)
	db, err := sql.Open(d.DriverName(), dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to instance: %w", err)
	}
	defer db.Close()

	// Revoke all then grant new permissions
	if _, err := db.Exec(d.RevokeAllSQL(cred.DBUsername)); err != nil {
		// Ignore revoke error (user might have no grants)
		_ = err
	}
	for _, perm := range newPermissions {
		if err := s.grantPermission(db, d, cred.DBUsername, perm); err != nil {
			return fmt.Errorf("failed to grant permission: %w", err)
		}
	}

	// 5. Update Role in Repo
	if err := s.DBUserCredRepo.UpdateRole(cred.ID, newRole); err != nil {
		return fmt.Errorf("failed to update role in repo: %w", err)
	}

	return nil
}

func (s *DBUserProvisioningService) getPermissionsForRole(role, dbType string) []models.Permission {
	var perms []models.Permission

	// Define scopes
	dbs := []string{"*"} // For MySQL
	if dbType == "postgres" {
		dbs = []string{"postgres"} // Default DB, logic usually requires loop over all DBs
		// For MVP, we assume a single tenant DB or main DB.
		// Detailed postgres permission management requires more context.
	}

	for _, dbName := range dbs {
		perm := models.Permission{
			Database: dbName,
			Table:    "*",
			Type:     "permanent",
		}

		switch role {
		case "read_only":
			perm.Privileges = []string{"SELECT"}
		case "read_write":
			perm.Privileges = []string{"SELECT", "INSERT", "UPDATE", "DELETE"}
		case "admin", "db_owner":
			perm.Privileges = []string{"ALL PRIVILEGES"} // MySQL specific, Postgres uses ALL
			if dbType == "postgres" {
				perm.Privileges = []string{"ALL"}
			}
		default:
			return nil
		}
		perms = append(perms, perm)
	}
	return perms
}
