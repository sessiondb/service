package service

import (
	"database/sql"
	"fmt"
	"sessiondb/internal/models"
	"sessiondb/internal/repository"
	"sessiondb/internal/utils"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
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

// createDBUserOnInstance creates the actual database user
func (s *DBUserProvisioningService) createDBUserOnInstance(instance *models.DBInstance, username, password string) error {
	dsn := s.buildAdminDSN(instance)
	
	db, err := sql.Open(instance.Type, dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to instance: %w", err)
	}
	defer db.Close()
	
	var createUserSQL string
	if instance.Type == "mysql" {
		createUserSQL = fmt.Sprintf("CREATE USER '%s'@'%%' IDENTIFIED BY '%s'", username, password)
	} else if instance.Type == "postgres" {
		createUserSQL = fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s'", username, password)
	} else {
		return fmt.Errorf("unsupported database type: %s", instance.Type)
	}
	
	if _, err := db.Exec(createUserSQL); err != nil {
		return fmt.Errorf("failed to execute CREATE USER: %w", err)
	}
	
	return nil
}

// GrantPermissions grants permissions to a DB user
func (s *DBUserProvisioningService) GrantPermissions(cred *models.DBUserCredential, permissions []models.Permission) error {
	instance, err := s.InstanceRepo.FindByID(cred.InstanceID)
	if err != nil {
		return fmt.Errorf("failed to find instance: %w", err)
	}
	
	dsn := s.buildAdminDSN(instance)
	db, err := sql.Open(instance.Type, dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to instance: %w", err)
	}
	defer db.Close()
	
	for _, perm := range permissions {
		if err := s.grantPermission(db, instance.Type, cred.DBUsername, perm); err != nil {
			return fmt.Errorf("failed to grant permission: %w", err)
		}
	}
	
	return nil
}

// grantPermission grants a single permission
func (s *DBUserProvisioningService) grantPermission(db *sql.DB, dbType, username string, perm models.Permission) error {
	privileges := strings.Join(perm.Privileges, ", ")
	
	var grantSQL string
	if dbType == "mysql" {
		if perm.Table == "*" {
			// Grant on entire database
			grantSQL = fmt.Sprintf("GRANT %s ON %s.* TO '%s'@'%%'", privileges, perm.Database, username)
		} else {
			// Grant on specific table
			grantSQL = fmt.Sprintf("GRANT %s ON %s.%s TO '%s'@'%%'", privileges, perm.Database, perm.Table, username)
		}
	} else if dbType == "postgres" {
		if perm.Table == "*" {
			// Grant on entire database
			grantSQL = fmt.Sprintf("GRANT CONNECT ON DATABASE %s TO %s; GRANT %s ON ALL TABLES IN SCHEMA public TO %s", 
				perm.Database, username, privileges, username)
		} else {
			// Grant on specific table
			grantSQL = fmt.Sprintf("GRANT %s ON TABLE %s TO %s", privileges, perm.Table, username)
		}
	}
	
	if _, err := db.Exec(grantSQL); err != nil {
		return fmt.Errorf("failed to execute GRANT: %w", err)
	}
	
	return nil
}

// RevokePermissions revokes permissions from a DB user
func (s *DBUserProvisioningService) RevokePermissions(cred *models.DBUserCredential, permissions []models.Permission) error {
	instance, err := s.InstanceRepo.FindByID(cred.InstanceID)
	if err != nil {
		return fmt.Errorf("failed to find instance: %w", err)
	}
	
	dsn := s.buildAdminDSN(instance)
	db, err := sql.Open(instance.Type, dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to instance: %w", err)
	}
	defer db.Close()
	
	for _, perm := range permissions {
		if err := s.revokePermission(db, instance.Type, cred.DBUsername, perm); err != nil {
			return fmt.Errorf("failed to revoke permission: %w", err)
		}
	}
	
	return nil
}

// revokePermission revokes a single permission
func (s *DBUserProvisioningService) revokePermission(db *sql.DB, dbType, username string, perm models.Permission) error {
	privileges := strings.Join(perm.Privileges, ", ")
	
	var revokeSQL string
	if dbType == "mysql" {
		if perm.Table == "*" {
			revokeSQL = fmt.Sprintf("REVOKE %s ON %s.* FROM '%s'@'%%'", privileges, perm.Database, username)
		} else {
			revokeSQL = fmt.Sprintf("REVOKE %s ON %s.%s FROM '%s'@'%%'", privileges, perm.Database, perm.Table, username)
		}
	} else if dbType == "postgres" {
		if perm.Table == "*" {
			revokeSQL = fmt.Sprintf("REVOKE %s ON ALL TABLES IN SCHEMA public FROM %s", privileges, username)
		} else {
			revokeSQL = fmt.Sprintf("REVOKE %s ON TABLE %s FROM %s", privileges, perm.Table, username)
		}
	}
	
	if _, err := db.Exec(revokeSQL); err != nil {
		return fmt.Errorf("failed to execute REVOKE: %w", err)
	}
	
	return nil
}

// DeprovisionDBUser drops the database user
func (s *DBUserProvisioningService) DeprovisionDBUser(cred *models.DBUserCredential) error {
	instance, err := s.InstanceRepo.FindByID(cred.InstanceID)
	if err != nil {
		return fmt.Errorf("failed to find instance: %w", err)
	}
	
	dsn := s.buildAdminDSN(instance)
	db, err := sql.Open(instance.Type, dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to instance: %w", err)
	}
	defer db.Close()
	
	var dropUserSQL string
	if instance.Type == "mysql" {
		dropUserSQL = fmt.Sprintf("DROP USER '%s'@'%%'", cred.DBUsername)
	} else if instance.Type == "postgres" {
		dropUserSQL = fmt.Sprintf("DROP USER %s", cred.DBUsername)
	}
	
	if _, err := db.Exec(dropUserSQL); err != nil {
		return fmt.Errorf("failed to drop user: %w", err)
	}
	
	// Update credential status
	if err := s.DBUserCredRepo.UpdateStatus(cred.ID, "revoked"); err != nil {
		return fmt.Errorf("failed to update credential status: %w", err)
	}
	
	return nil
}

// buildAdminDSN builds a DSN using admin credentials
func (s *DBUserProvisioningService) buildAdminDSN(instance *models.DBInstance) string {
	if instance.Type == "mysql" {
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			instance.Username,
			instance.Password,
			instance.Host,
			instance.Port,
			"mysql", // Connect to mysql database for user management
		)
	} else if instance.Type == "postgres" {
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			instance.Host,
			instance.Port,
			instance.Username,
			instance.Password,
			"postgres", // Connect to postgres database for user management
		)
	}
	return ""
}
