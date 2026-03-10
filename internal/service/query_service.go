// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package service

import (
	"context"
	"database/sql"
	"fmt"
	"sessiondb/internal/apierrors"
	"sessiondb/internal/config"
	"sessiondb/internal/dialect"
	"sessiondb/internal/engine"
	"sessiondb/internal/models"
	"sessiondb/internal/repository"
	"sessiondb/internal/utils"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// OnQueryExecutedHook is called after a query execution (e.g. for AlertEngine.EvaluateRules). Optional; premium only.
type OnQueryExecutedHook func(ctx context.Context, eventSource string, payload map[string]interface{})

type QueryService struct {
	QueryRepo        *repository.QueryRepository
	InstanceRepo     *repository.InstanceRepository
	DBUserCredRepo   *repository.DBUserCredentialRepository
	AuditService     *AuditService
	AccessEngine     engine.AccessEngine // optional: when set, enforces data access before execution
	Config           *config.Config
	OnQueryExecuted  OnQueryExecutedHook // optional: when set (e.g. pro), alert engine can evaluate rules
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

// SetAccessEngine sets the optional AccessEngine for data-level access checks before query execution.
func (s *QueryService) SetAccessEngine(ae engine.AccessEngine) {
	s.AccessEngine = ae
}

// SetOnQueryExecuted sets the optional hook called after each query execution (e.g. for premium alert evaluation).
func (s *QueryService) SetOnQueryExecuted(hook OnQueryExecutedHook) {
	s.OnQueryExecuted = hook
}

func (s *QueryService) ExecuteQuery(userID uuid.UUID, instanceID uuid.UUID, query, ipAddress, userAgent string) (interface{}, error) {
	// Lookup instance
	instance, err := s.InstanceRepo.FindByID(instanceID)
	if err != nil {
		return nil, fmt.Errorf("instance not found: %w", err)
	}

	// Data access check: user must have at least one permission on this instance
	if s.AccessEngine != nil {
		perms, err := s.AccessEngine.GetEffectivePermissions(context.Background(), userID, instanceID)
		if err != nil {
			return nil, fmt.Errorf("access check: %w", err)
		}
		if len(perms) == 0 {
			return nil, apierrors.NewAppError(403, apierrors.CodeForbidden, "no data access to this instance")
		}
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

	d, err := dialect.GetDialect(instance.Type)
	if err != nil {
		return nil, fmt.Errorf("unsupported database type: %w", err)
	}
	dsn := d.BuildDSNForUser(instance, "", dbUser, dbPass)
	db, err := sql.Open(d.DriverName(), dsn)
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

	// Optional: fire alert evaluation (premium) after query execution
	if s.OnQueryExecuted != nil {
		payload := map[string]interface{}{
			"duration_ms":   duration,
			"user_id":       userID.String(),
			"instance_id":   instanceID.String(),
			"instance_name": instance.Name,
			"status":        status,
		}
		go s.OnQueryExecuted(context.Background(), "query_execution", payload)
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
