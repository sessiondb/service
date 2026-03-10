//go:build pro

package session

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"sessiondb/internal/dialect"
	"sessiondb/internal/engine"
	"sessiondb/internal/models"
	core "sessiondb/internal/repository"
	premrepo "sessiondb/internal/premium/repository"
)

// Engine implements engine.SessionEngine for premium: ephemeral DB users with grants from Permission set.
type Engine struct {
	SessionRepo  *premrepo.CredentialSessionRepository
	InstanceRepo *core.InstanceRepository
	AccessEngine engine.AccessEngine
}

// NewEngine returns a new premium SessionEngine.
func NewEngine(
	sessionRepo *premrepo.CredentialSessionRepository,
	instanceRepo *core.InstanceRepository,
	accessEngine engine.AccessEngine,
) *Engine {
	return &Engine{
		SessionRepo:  sessionRepo,
		InstanceRepo: instanceRepo,
		AccessEngine: accessEngine,
	}
}

// StartSession creates an ephemeral DB user, grants permissions from the user's Permission set, and returns the session and temporary password.
func (e *Engine) StartSession(ctx context.Context, userID, instanceID uuid.UUID, ttl time.Duration) (*models.CredentialSession, string, error) {
	instance, err := e.InstanceRepo.FindByID(instanceID)
	if err != nil {
		return nil, "", fmt.Errorf("instance not found: %w", err)
	}
	d, err := dialect.GetDialect(instance.Type)
	if err != nil {
		return nil, "", err
	}
	perms, err := e.AccessEngine.GetEffectivePermissions(ctx, userID, instanceID)
	if err != nil {
		return nil, "", fmt.Errorf("get permissions: %w", err)
	}
	if len(perms) == 0 {
		return nil, "", fmt.Errorf("no data access to this instance; grant permissions first")
	}

	password, err := generatePassword(24)
	if err != nil {
		return nil, "", err
	}
	username := fmt.Sprintf("sdb_eph_%s_%d", shortUUID(userID), time.Now().Unix())

	dsn := d.BuildAdminDSN(instance)
	db, err := sql.Open(d.DriverName(), dsn)
	if err != nil {
		return nil, "", fmt.Errorf("connect to instance: %w", err)
	}
	defer db.Close()

	if _, err := db.Exec(d.CreateUserSQL(username, password)); err != nil {
		return nil, "", fmt.Errorf("create user: %w", err)
	}

	for _, perm := range perms {
		if err := e.grantPermission(db, d, username, perm); err != nil {
			_, _ = db.Exec(d.DropUserSQL(username))
			return nil, "", fmt.Errorf("grant %s.%s: %w", perm.Database, perm.Table, err)
		}
	}

	now := time.Now()
	session := &models.CredentialSession{
		UserID:      userID,
		InstanceID:  instanceID,
		SessionType: "ephemeral",
		DBUsername:  username,
		StartedAt:   now,
		ExpiresAt:   now.Add(ttl),
		Status:      "active",
	}
	if err := e.SessionRepo.Create(session); err != nil {
		_, _ = db.Exec(d.DropUserSQL(username))
		return nil, "", fmt.Errorf("create session record: %w", err)
	}
	return session, password, nil
}

// EndSession drops the ephemeral user and updates the CredentialSession record.
func (e *Engine) EndSession(ctx context.Context, sessionID uuid.UUID) error {
	session, err := e.SessionRepo.FindByID(sessionID)
	if err != nil {
		return err
	}
	if session.Status != "active" {
		return fmt.Errorf("session already ended: %s", session.Status)
	}

	instance, err := e.InstanceRepo.FindByID(session.InstanceID)
	if err != nil {
		return err
	}
	d, err := dialect.GetDialect(instance.Type)
	if err != nil {
		return err
	}
	dsn := d.BuildAdminDSN(instance)
	db, err := sql.Open(d.DriverName(), dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	if _, err := db.Exec(d.DropUserSQL(session.DBUsername)); err != nil {
		return fmt.Errorf("drop user: %w", err)
	}

	now := time.Now()
	session.EndedAt = &now
	session.Status = "revoked"
	return e.SessionRepo.Update(session)
}

// GetActiveSession returns the active CredentialSession for the user on the given instance, if any.
func (e *Engine) GetActiveSession(ctx context.Context, userID, instanceID uuid.UUID) (*models.CredentialSession, error) {
	return e.SessionRepo.FindActiveByUserAndInstance(userID, instanceID)
}

// GetSessionByID returns a session by ID.
func (e *Engine) GetSessionByID(ctx context.Context, sessionID uuid.UUID) (*models.CredentialSession, error) {
	return e.SessionRepo.FindByID(sessionID)
}

func (e *Engine) grantPermission(db *sql.DB, d dialect.DatabaseDialect, username string, perm models.Permission) error {
	schema := "public"
	if d.Type() == "mysql" {
		schema = perm.Database
	}
	privs := mapToSQLPrivileges(perm.Privileges)
	if len(perm.Column) > 0 {
		cols := []string{perm.Column}
		sqlStr := d.GrantColumnSQL(username, perm.Database, schema, perm.Table, cols, privs)
		_, err := db.Exec(sqlStr)
		return err
	}
	sqlStr := d.GrantTableSQL(username, perm.Database, schema, perm.Table, privs)
	_, err := db.Exec(sqlStr)
	return err
}

func mapToSQLPrivileges(privs []string) []string {
	out := make([]string, 0, len(privs))
	for _, p := range privs {
		switch strings.ToUpper(p) {
		case "READ":
			out = append(out, "SELECT")
		case "WRITE":
			out = append(out, "INSERT", "UPDATE")
		case "DELETE":
			out = append(out, "DELETE")
		case "EXECUTE":
			out = append(out, "EXECUTE")
		case "SELECT", "INSERT", "UPDATE", "ALL", "ALL PRIVILEGES":
			out = append(out, p)
		default:
			out = append(out, p)
		}
	}
	return out
}

func generatePassword(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b)[:n], nil
}

func shortUUID(id uuid.UUID) string {
	return strings.ReplaceAll(id.String(), "-", "")[:12]
}

// Ensure Engine implements engine.SessionEngine.
var _ engine.SessionEngine = (*Engine)(nil)
