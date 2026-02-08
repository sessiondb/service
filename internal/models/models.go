package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base model ensuring UUIDs are used
type Base struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time `json:"timestamp"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// QueryTab model
type QueryTab struct {
	Base
	UserID   uuid.UUID `gorm:"index"`
	Name     string
	Query    string
	IsActive bool
}

// User model
type User struct {
	Base
	Name             string `gorm:"not null"`
	Email            string `gorm:"uniqueIndex;not null"`
	PasswordHash     string
	DBUsername       string `gorm:"uniqueIndex"`
	RoleID           uuid.UUID
	Role             Role
	Status           string `gorm:"default:'active'"` // active, inactive, suspended
	IsSessionBased   bool   `gorm:"default:false"`
	SessionExpiresAt *time.Time
	LastLogin        *time.Time
	SSOID            string

	Permissions      []Permission      `gorm:"foreignKey:UserID"`
	ApprovalRequests []ApprovalRequest `gorm:"foreignKey:RequesterID"`
	SavedScripts     []SavedScript     `gorm:"foreignKey:UserID"`
	QueryTabs        []QueryTab        `gorm:"foreignKey:UserID"`
}

// Role model
type Role struct {
	Base
	Name         string `gorm:"uniqueIndex;not null"`
	Description  string
	IsSystemRole bool         `gorm:"default:false"`
	Permissions  []Permission `gorm:"foreignKey:RoleID"`
	UserCount    int          `gorm:"-"` // Computed field, not stored in DB
}

// Permission model
type Permission struct {
	Base
	RoleID     *uuid.UUID `gorm:"index"`
	UserID     *uuid.UUID `gorm:"index"`
	Database   string     `gorm:"not null"` // '*' for all
	Table      string     `gorm:"not null"` // '*' for all
	Privileges []string   `gorm:"type:text[]"` // Array of strings: READ, WRITE, DELETE, EXECUTE, ALL
	Type       string     `gorm:"default:'permanent'"` // permanent, temp, expiring
	ExpiresAt  *time.Time
}

// ApprovalRequest model
type ApprovalRequest struct {
	Base
	Type                 string `gorm:"not null"` // TEMP_USER, ROLE_CHANGE, PERM_UPGRADE
	RequesterID          uuid.UUID
	Requester            User `gorm:"foreignKey:RequesterID"`
	TargetUserID         *uuid.UUID
	Description          string
	Justification        string
	RequestedPermissions []byte     `gorm:"type:jsonb"` // Serialized permissions
	Status               string     `gorm:"default:'pending'"` // pending, approved, rejected, partially_approved
	ReviewedBy           *uuid.UUID
	ReviewedAt           *time.Time
	ApprovedPermissions  []byte    `gorm:"type:jsonb"`
	RejectionReason      string
	ExpiresAt            time.Time `gorm:"index"`
}

// AuditLog model
type AuditLog struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Timestamp    time.Time `gorm:"index"`
	UserID       uuid.UUID `gorm:"index"`
	User         User      `gorm:"foreignKey:UserID"`
	SessionUser  string
	Action       string `gorm:"index"`
	Resource     string
	ResourceType string
	Database     string
	Table        string
	Query        string
	QueryParams  []byte `gorm:"type:jsonb"`
	Status       string // Success, Failure, Warning
	ErrorMessage string
	IPAddress    string
	UserAgent    string
	DurationMs   int64
	RowsAffected int64
}

// QueryHistory model
type QueryHistory struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID          uuid.UUID `gorm:"index"`
	Query           string
	Database        string
	ExecutionTimeMs int64
	RowsReturned    int64
	Status          string
	ErrorMessage    string
	ExecutedAt      time.Time `gorm:"index"`
}

// SavedScript model
type SavedScript struct {
	Base
	UserID      uuid.UUID `gorm:"index"`
	Name        string
	Description string
	Query       string
	IsPublic    bool `gorm:"default:false"`
}

// DBInstance model
type DBInstance struct {
	Base
	Name        string `gorm:"not null" json:"name"`
	Host        string `gorm:"not null" json:"host"`
	Port        int    `gorm:"not null" json:"port"`
	Type        string `gorm:"not null" json:"type"` // e.g., postgres, mysql
	Username    string `gorm:"not null" json:"username,omitempty"`
	Password    string `gorm:"not null" json:"-"` // Never export password in general JSON
	Status      string `gorm:"default:'offline'" json:"status"`
	LastSync    *time.Time `json:"lastSync"`

	// Relationships
	Tables     []DBTable     `gorm:"foreignKey:InstanceID" json:"-"`
	Entities   []DBEntity    `gorm:"foreignKey:InstanceID" json:"-"`
	Privileges []DBPrivilege `gorm:"foreignKey:InstanceID" json:"-"`
}

// DBTable model
type DBTable struct {
	Base
	InstanceID uuid.UUID `gorm:"index" json:"instanceId"`
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
	Name       string    `json:"name"`
	Type       string    `json:"type"` // ROLE, USER
}

// DBPrivilege model
type DBPrivilege struct {
	Base
	InstanceID uuid.UUID `gorm:"index" json:"instanceId"`
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
