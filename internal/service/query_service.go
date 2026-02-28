package service

import (
	"database/sql"
	"fmt"
	"sessiondb/internal/apierrors"
	"sessiondb/internal/config"
	"sessiondb/internal/models"
	"sessiondb/internal/repository"
	"sessiondb/internal/utils"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
	"github.com/google/uuid"
	_ "github.com/lib/pq" // Postgres driver
)

type QueryService struct {
	QueryRepo      *repository.QueryRepository
	InstanceRepo   *repository.InstanceRepository
	DBUserCredRepo *repository.DBUserCredentialRepository
	AuditService   *AuditService
	Config         *config.Config
}

func NewQueryService(queryRepo *repository.QueryRepository, cfg *config.Config) *QueryService {
	return &QueryService{
		QueryRepo: queryRepo,
		Config:    cfg,
	}
}

// SetInstanceRepo allows injection of InstanceRepository after construction
func (s *QueryService) SetInstanceRepo(repo *repository.InstanceRepository) {
	s.InstanceRepo = repo
}

// SetAuditService allows injection of AuditService after construction
func (s *QueryService) SetAuditService(svc *AuditService) {
	s.AuditService = svc
}

// SetDBUserCredRepo allows injection of DBUserCredentialRepository after construction
func (s *QueryService) SetDBUserCredRepo(repo *repository.DBUserCredentialRepository) {
	s.DBUserCredRepo = repo
}

func (s *QueryService) ExecuteQuery(userID uuid.UUID, instanceID uuid.UUID, query, ipAddress, userAgent string) (interface{}, error) {
	// Lookup instance
	instance, err := s.InstanceRepo.FindByID(instanceID)
	if err != nil {
		return nil, fmt.Errorf("instance not found: %w", err)
	}

	// Resolve credentials: try user-level creds first, fall back to admin
	dbUser := instance.Username
	dbPass := instance.Password
	usingUserCreds := false

	if s.DBUserCredRepo != nil {
		cred, credErr := s.DBUserCredRepo.FindByUserAndInstance(userID, instanceID)
		if credErr != nil || cred == nil {
			// No user credential found — return specific error code
			return nil, apierrors.ErrUserCredsReq
		}
		// Decrypt the stored password
		plainPass, decErr := utils.DecryptPassword(cred.DBPassword)
		if decErr != nil {
			return nil, apierrors.ErrUserCredsInvalid
		}
		dbUser = cred.DBUsername
		dbPass = plainPass
		usingUserCreds = true
	}

	// Build DSN based on instance type
	var dsn string
	var driverName string
	switch instance.Type {
	case "postgres":
		driverName = "postgres"
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=disable",
			instance.Host, instance.Port, dbUser, dbPass)
	case "mysql":
		driverName = "mysql"
		cleanUser := dbUser
		if strings.Contains(cleanUser, "@") {
			cleanUser = strings.Split(cleanUser, "@")[0]
			cleanUser = strings.Trim(cleanUser, "'\"")
		}
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/",
			cleanUser, dbPass, instance.Host, instance.Port)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", instance.Type)
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		if usingUserCreds {
			return nil, apierrors.ErrUserCredsInvalid
		}
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer db.Close()

	// Verify connection — catches auth failures
	if err := db.Ping(); err != nil {
		if usingUserCreds && isAuthError(err) {
			return nil, apierrors.ErrUserCredsInvalid
		}
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	start := time.Now()
	rows, err := db.Query(query)
	duration := time.Since(start).Milliseconds()

	status := "success"
	errMsg := ""
	if err != nil {
		status = "error"
		errMsg = err.Error()
	}

	// Async log history via AuditService
	if s.AuditService != nil {
		s.AuditService.LogQuery(userID, instanceID, instance.Name, query, instance.Name, status, errMsg, ipAddress, userAgent, duration)
	} else {
		// Fallback logging if service not injected (shouldn't happen in prod)
		fmt.Printf("WARNING: AuditService not injected into QueryService\n")
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, _ := rows.Columns()
	var dataRows [][]interface{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		rows.Scan(valuePtrs...)

		row := make([]interface{}, len(columns))
		for i := range columns {
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				row[i] = string(b)
			} else {
				row[i] = val
			}
		}
		dataRows = append(dataRows, row)
	}

	if dataRows == nil {
		dataRows = [][]interface{}{}
	}

	return map[string]interface{}{
		"columns":  columns,
		"rows":     dataRows,
		"rowCount": len(dataRows),
	}, nil
}

func (s *QueryService) GetHistory(userID uuid.UUID) ([]models.AuditLog, error) {
	if s.AuditService == nil {
		return nil, fmt.Errorf("audit service not injected")
	}
	return s.AuditService.GetQueryHistory(userID)
}

func (s *QueryService) SaveScript(userID uuid.UUID, name, query string, isPublic bool) (*models.SavedScript, error) {
	script := &models.SavedScript{
		UserID:   userID,
		Name:     name,
		Query:    query,
		IsPublic: isPublic,
	}
	if err := s.QueryRepo.SaveScript(script); err != nil {
		return nil, err
	}
	return script, nil
}

func (s *QueryService) GetScripts(userID uuid.UUID) ([]models.SavedScript, error) {
	return s.QueryRepo.GetScripts(userID)
}

// isAuthError checks if a database error is an authentication/access-denied failure.
func isAuthError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "access denied") ||
		strings.Contains(msg, "password authentication failed") ||
		strings.Contains(msg, "authentication failed") ||
		strings.Contains(msg, "invalid password")
}
