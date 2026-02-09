package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base model ensuring UUIDs are used
type Base struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
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
	DBUsername       string     `gorm:"uniqueIndex" json:"dbUsername"`
	RoleID           uuid.UUID  `json:"roleId"`
	Role             Role       `json:"role"`
	Status           string     `gorm:"default:'active'" json:"status"` // active, inactive, suspended
	IsSessionBased   bool       `gorm:"default:false" json:"isSessionBased"`
	SessionExpiresAt *time.Time `json:"sessionExpiresAt,omitempty"`
	LastLogin        *time.Time `json:"lastLogin,omitempty"`
	SSOID            string     `json:"ssoId,omitempty"`

	Permissions      []Permission      `gorm:"foreignKey:UserID" json:"permissions"`
	ApprovalRequests []ApprovalRequest `gorm:"foreignKey:RequesterID" json:"approvalRequests,omitempty"`
	SavedScripts     []SavedScript     `gorm:"foreignKey:UserID" json:"savedScripts"`
	QueryTabs        []QueryTab        `gorm:"foreignKey:UserID" json:"queryTabs"`
}

// Role model
type Role struct {
	Base
	Name         string       `gorm:"uniqueIndex;not null" json:"name"`
	Description  string       `json:"description,omitempty"`
	IsSystemRole bool         `gorm:"default:false" json:"isSystemRole"`
	Permissions  []Permission `gorm:"foreignKey:RoleID" json:"permissions"`
	UserCount    int          `gorm:"-" json:"userCount,omitempty"` // Computed field, not stored in DB
}

// Permission model
type Permission struct {
	Base
	RoleID     *uuid.UUID `gorm:"index" json:"roleId,omitempty"`
	UserID     *uuid.UUID `gorm:"index" json:"userId,omitempty"`
	Database   string     `gorm:"not null" json:"database"` // '*' for all
	Table      string     `gorm:"not null" json:"table"` // '*' for all
	Privileges []string   `gorm:"type:text[]" json:"privileges"` // Array of strings: READ, WRITE, DELETE, EXECUTE, ALL
	Type       string     `gorm:"default:'permanent'" json:"type"` // permanent, temp, expiring
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
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
	Status               string     `gorm:"default:'pending'" json:"status"` // pending, approved, rejected, partially_approved
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

// DBInstance model
type DBInstance struct {
	Base
	Name        string `gorm:"not null" json:"name"`
	Host        string `gorm:"not null" json:"host"`
	Port        int    `gorm:"not null" json:"port"`
	Type     string `gorm:"not null" json:"type"` // e.g., postgres, mysql
	Username string `gorm:"not null" json:"username,omitempty"`
	Password     string `gorm:"not null" json:"-"` // Never export password in general JSON
	Status       string `gorm:"default:'offline'" json:"status"`
	LastSync     *time.Time `json:"lastSync"`

	// Relationships
	Tables     []DBTable     `gorm:"foreignKey:InstanceID" json:"-"`
	Entities   []DBEntity    `gorm:"foreignKey:InstanceID" json:"-"`
	Privileges []DBPrivilege `gorm:"foreignKey:InstanceID" json:"-"`
}

// DBTable model
type DBTable struct {
	Base
	InstanceID uuid.UUID `gorm:"index" json:"instanceId"`
	Database   string    `json:"database"`
	Schema     string    `json:"schema"`
	Name       string    `json:"name"`
	Type       string    `json:"type"` // BASE TABLE, VIEW
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
	Type       string    `json:"type"` // ROLE, USER
}

// DBPrivilege model
type DBPrivilege struct {
	Base
	InstanceID uuid.UUID `gorm:"index" json:"instanceId"`
	Database   string    `json:"database"`
	Grantee    string    `json:"grantee"` // Name of user/role in target DB
	Schema     string    `json:"schema"`
	Table      string    `json:"table"`
	Privilege  string    `json:"privilege"` // SELECT, INSERT, etc.
}

// BeforeCreate hook to generate UUIDs if not present
func (base *Base) BeforeCreate(tx *gorm.DB) error {
	if base.ID == uuid.Nil {
		base.ID = uuid.New()
	}
	return nil
}
