// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// Base model ensuring UUIDs are used
type Base struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// QueryTab model
type QueryTab struct {
	Base
	UserID   uuid.UUID `gorm:"index" json:"userId"`
	Name     string    `json:"name"`
	Query    string    `json:"query"`
	IsActive bool      `json:"isActive"`
}

// User model
type User struct {
	Base
	Name             string     `gorm:"not null" json:"name"`
	Email            string     `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash     string     `json:"-"`
	RoleID           uuid.UUID  `json:"roleId"`
	Role             Role       `json:"role"`
	Status           string     `gorm:"default:'active'" json:"status"` // active, inactive, suspended
	IsSessionBased   bool       `gorm:"default:false" json:"isSessionBased"`
	SessionExpiresAt *time.Time `json:"sessionExpiresAt,omitempty"`
	LastLogin        *time.Time `json:"lastLogin,omitempty"`
	SSOID            string     `json:"ssoId,omitempty"`

	Permissions      []Permission       `gorm:"foreignKey:UserID" json:"permissions"`
	ApprovalRequests []ApprovalRequest  `gorm:"foreignKey:RequesterID" json:"approvalRequests,omitempty"`
	SavedScripts     []SavedScript      `gorm:"foreignKey:UserID" json:"savedScripts"`
	QueryTabs        []QueryTab         `gorm:"foreignKey:UserID" json:"queryTabs"`
	DBCredentials    []DBUserCredential `gorm:"foreignKey:UserID" json:"dbCredentials,omitempty"`
	RBACPermissions  pq.StringArray     `gorm:"type:text[]" json:"rbacPermissions"`
}

// Tenant model
type Tenant struct {
	Base
	Name           string `gorm:"not null" json:"name"`
	TenantFeatures []byte `gorm:"type:jsonb" json:"tenantFeatures"` // map[string]bool stored as JSONB
}

// Role model
type Role struct {
	Base
	Name         string       `gorm:"uniqueIndex;not null" json:"name"`
	Key          string       `gorm:"column:key;uniqueIndex;not null" json:"key"` // Role key, underscored e.g. super_admin
	Description  string       `json:"description,omitempty"`
	IsSystemRole bool         `gorm:"default:false" json:"isSystemRole"`
	Permissions  []Permission `gorm:"foreignKey:RoleID" json:"permissions"`
	UserCount    int          `gorm:"-" json:"userCount,omitempty"` // Computed field, not stored in DB
}

// Permission model — data-level access (instance/database/schema/table/column).
// System RBAC (users:read, etc.) is separate; see utils/rbac.go.
type Permission struct {
	Base
	RoleID       *uuid.UUID `gorm:"index" json:"roleId,omitempty"`
	UserID       *uuid.UUID `gorm:"index" json:"userId,omitempty"`
	InstanceID   *uuid.UUID `gorm:"index" json:"instanceId,omitempty"` // which target DB instance (nil = legacy)
	Database     string     `gorm:"not null" json:"database"`          // '*' for all
	Schema       string     `json:"schema,omitempty"`
	Table        string     `gorm:"not null" json:"table"`   // '*' for all
	Column       string     `json:"column,omitempty"`        // empty = table-level; specific = column-level
	Privileges   pq.StringArray `gorm:"type:text[]" json:"privileges"` // SELECT, INSERT, UPDATE, DELETE, ALL
	Type         string     `gorm:"default:'permanent'" json:"type"`    // permanent, temp, expiring
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
	Expiry       *time.Time `gorm:"-" json:"expiry,omitempty"` // Alias for frontend compatibility
	GrantedBy    uuid.UUID  `gorm:"type:uuid" json:"grantedBy,omitempty"`
	ScheduleCron *string    `json:"scheduleCron,omitempty"` // Premium: time-window access (e.g. "0 9-17 * * MON-FRI")
}

func (p *Permission) BeforeSave(tx *gorm.DB) error {
	if p.Expiry != nil && p.ExpiresAt == nil {
		p.ExpiresAt = p.Expiry
	}
	return nil
}

// ApprovalRequest model
type ApprovalRequest struct {
	Base
	Type                 string     `gorm:"not null" json:"type"` // TEMP_USER, ROLE_CHANGE, PERM_UPGRADE
	RequesterID          uuid.UUID  `json:"requesterId"`
	Requester            User       `gorm:"foreignKey:RequesterID" json:"requester"`
	TargetUserID         *uuid.UUID `json:"targetUserId,omitempty"`
	Description          string     `json:"description"`
	Justification        string     `json:"justification,omitempty"`
	RequestedPermissions []byte     `gorm:"type:jsonb" json:"requestedPermissions,omitempty"` // Serialized permissions
	Status               string     `gorm:"default:'pending'" json:"status"`                  // pending, approved, rejected, partially_approved
	ReviewedBy           *uuid.UUID `json:"reviewedBy,omitempty"`
	ReviewedAt           *time.Time `json:"reviewedAt,omitempty"`
	ApprovedPermissions  []byte     `gorm:"type:jsonb" json:"approvedPermissions,omitempty"`
	RejectionReason      string     `json:"rejectionReason,omitempty"`
	ExpiresAt            time.Time  `gorm:"index" json:"expiresAt"`
}

// AuditLog model
type AuditLog struct {
	ID           uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Timestamp    time.Time  `gorm:"index" json:"timestamp"`
	UserID       uuid.UUID  `gorm:"index" json:"userId"`
	User         User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	SessionUser  string     `json:"sessionUser,omitempty"`
	Action       string     `gorm:"index" json:"action"`
	Resource     string     `json:"resource"`
	ResourceType string     `json:"resourceType,omitempty"`
	InstanceID   *uuid.UUID `gorm:"index" json:"instanceId,omitempty"`
	InstanceName string     `json:"instanceName,omitempty"`
	Database     string     `json:"database,omitempty"`
	Table        string     `json:"table,omitempty"`
	Query        string     `json:"query,omitempty"`
	QueryParams  []byte     `gorm:"type:jsonb" json:"queryParams,omitempty"`
	Status       string     `json:"status"` // Success, Failure, Warning
	ErrorMessage string     `json:"errorMessage,omitempty"`
	IPAddress    string     `json:"ipAddress,omitempty"`
	UserAgent    string     `json:"userAgent,omitempty"`
	RequestID    string     `gorm:"index" json:"requestId,omitempty"`
	SessionID    string     `gorm:"index" json:"sessionId,omitempty"`
	DBUsername   string     `gorm:"index" json:"dbUsername,omitempty"` // Denormalized for performance
	DurationMs   int64      `json:"durationMs,omitempty"`
	RowsAffected int64      `json:"rowsAffected,omitempty"`
}

// SavedScript model
type SavedScript struct {
	Base
	UserID      uuid.UUID `gorm:"index" json:"userId"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Query       string    `json:"query"`
	IsPublic    bool      `gorm:"default:false" json:"isPublic"`
}

// DBUserCredential model - Maps platform users to database users
type DBUserCredential struct {
	Base
	UserID     uuid.UUID  `gorm:"index;not null" json:"userId"`
	User       User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	InstanceID uuid.UUID  `gorm:"index;not null" json:"instanceId"`
	Instance   DBInstance `gorm:"foreignKey:InstanceID" json:"instance,omitempty"`

	DBUsername string `gorm:"not null;index" json:"dbUsername"` // Actual DB username (e.g., sdb_dev_john_perm)
	DBPassword string `gorm:"not null" json:"-"`                // AES-256 encrypted password
	Role       string `json:"role"`                             // e.g., read_only, read_write, admin

	ExpiresAt *time.Time `gorm:"index" json:"expiresAt,omitempty"`     // For temporary users
	Status    string     `gorm:"default:'active';index" json:"status"` // active, expired, revoked
}

// DBInstance model
type DBInstance struct {
	Base
	Name     string     `gorm:"not null" json:"name"`
	Host     string     `gorm:"not null" json:"host"`
	Port     int        `gorm:"not null" json:"port"`
	Type     string     `gorm:"not null" json:"type"` // e.g., postgres, mysql
	Username string     `gorm:"not null" json:"username,omitempty"`
	Password string     `gorm:"not null" json:"-"` // Never export password in general JSON
	Status   string     `gorm:"default:'offline'" json:"status"`
	LastSync *time.Time `json:"lastSync"`

	// Monitoring
	IsProd            bool   `gorm:"default:false" json:"isProd"`
	MonitoringEnabled bool   `gorm:"default:false" json:"monitoringEnabled"`
	AlertEmail        string `json:"alertEmail,omitempty"`

	// Relationships
	Tables     []DBTable     `gorm:"foreignKey:InstanceID" json:"-"`
	Entities   []DBEntity    `gorm:"foreignKey:InstanceID" json:"-"`
	Privileges []DBPrivilege `gorm:"foreignKey:InstanceID" json:"-"`
}

// DBMonitoringLog model
type DBMonitoringLog struct {
	Base
	InstanceID uuid.UUID  `gorm:"index;not null" json:"instanceId"`
	Instance   DBInstance `gorm:"foreignKey:InstanceID" json:"-"`
	Status     string     `json:"status"` // online, offline
	Uptime     int64      `json:"uptime"`
	Metrics    []byte     `gorm:"type:jsonb" json:"metrics"` // Storing other performance metrics as JSON
	Message    string     `json:"message"`
}

// DBTable model
type DBTable struct {
	Base
	InstanceID uuid.UUID  `gorm:"index" json:"instanceId"`
	Database   string     `json:"database"`
	Schema     string     `json:"schema"`
	Name       string     `json:"name"`
	Type       string     `json:"type"` // BASE TABLE, VIEW
	Columns    []DBColumn `gorm:"foreignKey:TableID" json:"columns"`
}

// DBColumn model
type DBColumn struct {
	Base
	TableID      uuid.UUID `gorm:"index" json:"tableId"`
	Name         string    `json:"name"`
	DataType     string    `json:"dataType"`
	IsNullable   bool      `json:"isNullable"`
	DefaultValue *string   `json:"defaultValue"`
	IsPrimaryKey bool      `json:"isPrimaryKey"`
}

// DBEntity model (Represents target DB Roles/Users)
type DBEntity struct {
	Base
	InstanceID uuid.UUID `gorm:"index" json:"instanceId"`
	Database   string    `json:"database"`
	Name       string    `json:"name"`
	DBKey      string    `json:"dbKey"` // Original name from DB (e.g. snake_case)
	Type       string    `json:"type"`  // ROLE, USER
}

// DBPrivilege model
type DBPrivilege struct {
	Base
	InstanceID  uuid.UUID `gorm:"index" json:"instanceId"`
	Database    string    `json:"database"`
	Grantee     string    `json:"grantee"` // Name of user/role in target DB
	Schema      string    `json:"schema"`
	Table       string    `json:"table"`
	Privilege   string    `json:"privilege"` // SELECT, INSERT, etc.
	IsGrantable bool      `json:"isGrantable"`
}

// DBRoleMembership model - tracks role hierarchy in target DBs
type DBRoleMembership struct {
	Base
	InstanceID uuid.UUID `gorm:"index" json:"instanceId"`
	RoleName   string    `json:"roleName"`   // The role being granted
	MemberName string    `json:"memberName"` // The user/role receiving the grant
}

// UserAIConfig stores BYOK API keys for the AI query engine (community). Encrypted at rest.
type UserAIConfig struct {
	Base
	UserID       uuid.UUID `gorm:"uniqueIndex:idx_user_ai_provider;not null" json:"userId"`
	ProviderType string    `gorm:"not null" json:"providerType"` // openai, anthropic, custom
	APIKey       string    `gorm:"not null" json:"-"`            // AES-256 encrypted
	BaseURL      *string   `json:"baseUrl,omitempty"`            // for custom OpenAI-compatible endpoints
	ModelName    string    `json:"modelName"`                    // gpt-4, claude-3-sonnet, etc.
}

// GlobalAIConfig stores the organization-wide AI API key (admin-set). Single row per deployment.
type GlobalAIConfig struct {
	Base
	ProviderType string  `gorm:"not null" json:"providerType"`
	APIKey       string  `gorm:"not null" json:"-"`
	BaseURL      *string `json:"baseUrl,omitempty"`
	ModelName    string  `json:"modelName"`
}

// AITokenUsage records per-call AI usage for dashboard (generate_sql, explain). Token counts may be 0 if provider does not return them.
type AITokenUsage struct {
	Base
	UserID       uuid.UUID `gorm:"index;not null" json:"userId"`
	UsedGlobal   bool      `gorm:"index" json:"usedGlobal"`
	InputTokens  int       `json:"inputTokens"`
	OutputTokens int       `json:"outputTokens"`
	Model        string    `json:"model"`
	Feature      string    `gorm:"index;not null" json:"feature"` // generate_sql, explain
}

// AIExecutionPolicy defines per-instance rules for AI-generated query execution (e.g. require approval for DDL).
type AIExecutionPolicy struct {
	Base
	InstanceID      uuid.UUID      `gorm:"index;not null" json:"instanceId"`
	ActionType      string         `gorm:"not null" json:"actionType"`   // SELECT, INSERT, UPDATE, DELETE, DDL, USER_MGMT
	RequireApproval bool           `gorm:"default:true" json:"requireApproval"`
	AllowedRoles    pq.StringArray `gorm:"type:text[]" json:"allowedRoles"` // role names that can auto-execute
}

// CredentialSession (Phase 4 – Session Engine, premium) links ephemeral or leased DB credentials to a user and time window for audit.
type CredentialSession struct {
	Base
	UserID       uuid.UUID  `gorm:"index;not null" json:"userId"`
	InstanceID   uuid.UUID  `gorm:"index;not null" json:"instanceId"`
	SessionType  string     `gorm:"not null" json:"sessionType"` // "ephemeral", "lease"
	DBUsername   string     `gorm:"not null;index" json:"dbUsername"`
	CredentialID *uuid.UUID `gorm:"type:uuid;index" json:"credentialId,omitempty"` // for lease mode
	StartedAt    time.Time  `gorm:"not null" json:"startedAt"`
	ExpiresAt    time.Time  `gorm:"not null" json:"expiresAt"`
	EndedAt      *time.Time `json:"endedAt,omitempty"`
	Status       string     `gorm:"default:'active';index" json:"status"` // "active", "expired", "revoked"
}

// AlertRule (Phase 5 – Alert Engine, premium) defines when to fire alerts (e.g. threshold on query_execution).
type AlertRule struct {
	Base
	TenantID    uuid.UUID `gorm:"index;not null" json:"tenantId"`
	CreatedBy   uuid.UUID `gorm:"index;not null" json:"createdBy"`
	Name        string    `gorm:"not null" json:"name"`
	Description string    `json:"description,omitempty"`
	EventSource string    `gorm:"not null;index" json:"eventSource"` // e.g. "query_execution", "instance_health"
	Condition   []byte    `gorm:"type:jsonb" json:"condition"`       // threshold, filters, etc.
	Severity    string    `gorm:"not null;default:'medium'" json:"severity"` // low, medium, high, critical
	IsEnabled   bool      `gorm:"default:true" json:"isEnabled"`
	Channels    []byte    `gorm:"type:jsonb" json:"channels,omitempty"` // email, webhook, etc.
}

// AlertEvent (Phase 5) records a fired alert for a rule.
type AlertEvent struct {
	Base
	RuleID     uuid.UUID `gorm:"index;not null" json:"ruleId"`
	TenantID   uuid.UUID `gorm:"index;not null" json:"tenantId"`
	Severity   string    `gorm:"not null" json:"severity"`
	Title      string    `gorm:"not null" json:"title"`
	Description string   `json:"description,omitempty"`
	Metadata   []byte    `gorm:"type:jsonb" json:"metadata,omitempty"`
	Status     string    `gorm:"default:'open';index" json:"status"` // open, acknowledged, resolved
	Source     string    `json:"source,omitempty"`
}

// ReportDefinition (Phase 6 – Report Engine, premium) defines a report (data sources, filters, schedule).
type ReportDefinition struct {
	Base
	TenantID         uuid.UUID  `gorm:"index;not null" json:"tenantId"`
	CreatedBy        uuid.UUID  `gorm:"index;not null" json:"createdBy"`
	Name             string     `gorm:"not null" json:"name"`
	Description      string     `json:"description,omitempty"`
	DataSources      []byte     `gorm:"type:jsonb" json:"dataSources"`       // instance/queries or views
	Filters          []byte     `gorm:"type:jsonb" json:"filters,omitempty"`
	ScheduleCron     *string    `json:"scheduleCron,omitempty"`              // optional; nil = on-demand only
	DeliveryChannels []byte     `gorm:"type:jsonb" json:"deliveryChannels,omitempty"`
	Format           string     `gorm:"default:'csv'" json:"format"`        // csv, json
	IsEnabled        bool       `gorm:"default:true" json:"isEnabled"`
	LastRunAt        *time.Time `json:"lastRunAt,omitempty"`
}

// ReportExecution (Phase 6) records a single run of a report.
type ReportExecution struct {
	Base
	DefinitionID uuid.UUID  `gorm:"index;not null" json:"definitionId"`
	Status       string     `gorm:"not null;index" json:"status"` // running, completed, failed
	StartedAt    time.Time  `gorm:"not null" json:"startedAt"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
	ResultURL    string     `json:"resultUrl,omitempty"`
	Error        string     `json:"error,omitempty"`
}

// FeatureNotifyRequest stores "Notify me when this is ready" sign-ups for roadmap features.
// Used by the FeatureGate / waitlist flow; email comes from JWT when authenticated.
type FeatureNotifyRequest struct {
	Base
	Email      string    `gorm:"index;not null" json:"email"`
	FeatureKey string    `gorm:"index;not null" json:"featureKey"`
	UserID     *uuid.UUID `gorm:"index" json:"userId,omitempty"`
}

// BeforeCreate hook to generate UUIDs if not present
func (base *Base) BeforeCreate(tx *gorm.DB) error {
	if base.ID == uuid.Nil {
		base.ID = uuid.New()
	}
	return nil
}
