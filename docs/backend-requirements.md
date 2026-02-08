# SessionDB - Backend Requirements Document

## 1. Executive Summary

**SessionDB** is an enterprise-grade database management and access control system that provides:
- Secure, role-based database access control
- Session-based temporary user management
- SQL query execution with audit logging
- Approval workflows for privilege escalation
- Comprehensive audit trail for compliance

## 2. System Architecture

### 2.1 Technology Stack
- **Backend Framework**: Node.js with Express.js / Python with FastAPI (recommended)
- **Database**: PostgreSQL (primary) with support for multiple database connections
- **Authentication**: JWT-based authentication with optional SSO integration
- **Authorization**: Role-Based Access Control (RBAC) with granular permissions
- **Caching**: Redis for session management and query result caching
- **Message Queue**: RabbitMQ/Redis for async approval workflows
- **Logging**: Structured logging with ELK stack integration

### 2.2 High-Level Architecture

```
┌─────────────┐
│   Frontend  │
│  (React UI) │
└──────┬──────┘
       │ HTTPS/REST API
       │
┌──────▼──────────────────────────────────────┐
│         API Gateway / Load Balancer         │
└──────┬──────────────────────────────────────┘
       │
┌──────▼──────────────────────────────────────┐
│          Application Server Layer           │
│  ┌────────────┐  ┌──────────┐  ┌─────────┐ │
│  │ Auth       │  │ Query    │  │ Admin   │ │
│  │ Service    │  │ Service  │  │ Service │ │
│  └────────────┘  └──────────┘  └─────────┘ │
└──────┬──────────────────────────────────────┘
       │
┌──────▼──────────────────────────────────────┐
│         Data Access Layer                   │
│  ┌────────────┐  ┌──────────┐  ┌─────────┐ │
│  │ SessionDB  │  │ Target   │  │ Redis   │ │
│  │ Metadata   │  │ Database │  │ Cache   │ │
│  └────────────┘  └──────────┘  └─────────┘ │
└─────────────────────────────────────────────┘
```

### 2.3 Core Components

#### Authentication Service
- User authentication (password/SSO)
- JWT token generation and validation
- Session management
- Password reset workflows

#### Authorization Service
- Role-based permission checking
- Database-level and table-level access control
- Temporary permission management
- Permission expiry handling

#### Query Execution Service
- SQL query parsing and validation
- Query execution with permission checks
- Result set pagination
- Query history tracking

#### Admin Service
- User management (CRUD)
- Role management (CRUD)
- Approval workflow processing
- System configuration

#### Audit Service
- Activity logging
- Query logging
- Access logging
- Compliance reporting

## 3. Data Models

### 3.1 Core Entities

#### User
```typescript
interface User {
  id: string;                    // UUID
  name: string;                  // Display name
  email: string;                 // Unique email
  db_username: string;           // Database username mapping
  role_id: string;               // Foreign key to Role
  status: 'active' | 'inactive' | 'suspended';
  is_session_based: boolean;     // Temporary user flag
  session_expires_at?: Date;     // For session-based users
  last_login: Date;
  created_at: Date;
  updated_at: Date;
  created_by: string;
  password_hash?: string;        // For password auth
  sso_id?: string;              // For SSO auth
}
```

#### Role
```typescript
interface Role {
  id: string;                    // UUID
  name: string;                  // Unique role name
  description: string;
  is_system_role: boolean;       // Cannot be deleted
  created_at: Date;
  updated_at: Date;
  created_by: string;
}
```

#### Permission
```typescript
interface Permission {
  id: string;                    // UUID
  role_id?: string;              // FK to Role (null for user-specific)
  user_id?: string;              // FK to User (null for role-based)
  database: string;              // Database name or '*' for all
  table: string;                 // Table name or '*' for all
  privileges: string[];          // ['READ', 'WRITE', 'DELETE', 'EXECUTE', 'ALL']
  type: 'permanent' | 'temp' | 'expiring';
  expires_at?: Date;             // For temp/expiring permissions
  created_at: Date;
  created_by: string;
}
```

#### ApprovalRequest
```typescript
interface ApprovalRequest {
  id: string;                    // UUID
  type: 'TEMP_USER' | 'ROLE_CHANGE' | 'PERM_UPGRADE';
  requester_id: string;          // FK to User
  target_user_id?: string;       // FK to User (for role changes)
  description: string;
  justification: string;
  requested_permissions: Permission[];
  status: 'pending' | 'approved' | 'rejected' | 'partially_approved';
  reviewed_by?: string;          // FK to User
  reviewed_at?: Date;
  approved_permissions?: Permission[]; // For partial approvals
  rejection_reason?: string;
  created_at: Date;
  expires_at: Date;              // Auto-reject after expiry
}
```

#### AuditLog
```typescript
interface AuditLog {
  id: string;                    // UUID
  timestamp: Date;
  user_id: string;               // FK to User
  session_user?: string;         // DB username used
  action: string;                // Action type (enum)
  resource: string;              // Resource affected
  resource_type: string;         // 'user', 'role', 'query', etc.
  database?: string;
  table?: string;
  query?: string;                // SQL query executed
  query_params?: object;
  status: 'Success' | 'Failure' | 'Warning';
  error_message?: string;
  ip_address: string;
  user_agent: string;
  duration_ms?: number;          // Query execution time
  rows_affected?: number;
}
```

#### QueryHistory
```typescript
interface QueryHistory {
  id: string;                    // UUID
  user_id: string;               // FK to User
  query: string;
  database: string;
  execution_time_ms: number;
  rows_returned: number;
  status: 'success' | 'error';
  error_message?: string;
  executed_at: Date;
}
```

#### SavedScript
```typescript
interface SavedScript {
  id: string;                    // UUID
  user_id: string;               // FK to User
  name: string;
  description?: string;
  query: string;
  is_public: boolean;            // Shareable with team
  created_at: Date;
  updated_at: Date;
}
```

#### QueryTab
```typescript
interface QueryTab {
  id: string;                    // UUID
  user_id: string;               // FK to User
  name: string;
  query: string;
  order_index: number;
  is_active: boolean;
  created_at: Date;
  updated_at: Date;
}
```

#### DatabaseSchema
```typescript
interface DatabaseSchema {
  id: string;
  database_name: string;
  table_name: string;
  columns: ColumnInfo[];
  last_synced: Date;
}

interface ColumnInfo {
  name: string;
  type: string;
  nullable: boolean;
  default_value?: string;
  is_primary_key: boolean;
}
```

### 3.2 Database Relationships

```
User ──┬── 1:N ─→ Permission (user-specific)
       ├── N:1 ─→ Role
       ├── 1:N ─→ ApprovalRequest (as requester)
       ├── 1:N ─→ AuditLog
       ├── 1:N ─→ QueryHistory
       ├── 1:N ─→ SavedScript
       └── 1:N ─→ QueryTab

Role ──── 1:N ─→ Permission (role-based)

ApprovalRequest ──── N:1 ─→ User (reviewer)
```

## 4. Authentication & Authorization

### 4.1 Authentication Methods

#### Password-Based Authentication
- Bcrypt password hashing (cost factor: 12)
- JWT tokens with 24-hour expiry
- Refresh tokens with 30-day expiry
- Password complexity requirements:
  - Minimum 12 characters
  - At least 1 uppercase, 1 lowercase, 1 number, 1 special char
  - Password history (last 5 passwords)
  - 90-day password rotation policy

#### SSO Integration
- SAML 2.0 support
- OAuth 2.0 / OIDC support
- Support for major providers (Okta, Azure AD, Google Workspace)
- Just-in-time (JIT) user provisioning
- Attribute mapping for roles

### 4.2 Authorization Model

#### Permission Hierarchy
1. **System Admin**: Full access to all databases and system configuration
2. **Database Admin**: Full access to specific databases
3. **Role-based**: Permissions inherited from role
4. **User-specific**: Additional permissions granted to individual users
5. **Temporary**: Time-limited permissions

#### Permission Evaluation
```
Effective Permissions = 
  System Role Permissions 
  ∪ User Role Permissions 
  ∪ User-Specific Permissions (non-expired)
```

#### Access Control Rules
- Deny by default
- Explicit permissions required for all operations
- Wildcard support: `*.*` (all databases/tables)
- Table-level granularity: `production.users`
- Permission types: READ, WRITE, DELETE, EXECUTE, ALL

## 5. Functional Requirements

### 5.1 User Management

#### FR-UM-001: User Creation
- **Description**: Create new users with role assignment
- **Inputs**: name, email, role_id, db_username, is_session_based
- **Validations**:
  - Email uniqueness
  - Valid role_id
  - db_username format validation
- **Outputs**: Created user object
- **Audit**: Log user creation event

#### FR-UM-002: User Update
- **Description**: Update user details and role
- **Inputs**: user_id, updated fields
- **Validations**: 
  - User exists
  - Role change requires approval for privilege escalation
- **Outputs**: Updated user object
- **Audit**: Log user update event

#### FR-UM-003: User Deactivation
- **Description**: Soft delete user (set status to inactive)
- **Inputs**: user_id
- **Side Effects**:
  - Revoke all active sessions
  - Expire temporary permissions
- **Outputs**: Success confirmation
- **Audit**: Log user deactivation

#### FR-UM-004: Session-Based User Management
- **Description**: Create temporary users with auto-expiry
- **Inputs**: user details, expiry_duration
- **Validations**: Requires approval for production access
- **Side Effects**: Auto-cleanup on expiry
- **Outputs**: Temporary user credentials
- **Audit**: Log temporary user lifecycle

### 5.2 Role Management

#### FR-RM-001: Role Creation
- **Description**: Create role templates with baseline permissions
- **Inputs**: name, description, permissions[]
- **Validations**: Role name uniqueness
- **Outputs**: Created role object
- **Audit**: Log role creation

#### FR-RM-002: Role Update
- **Description**: Modify role permissions
- **Inputs**: role_id, updated permissions
- **Side Effects**: Affects all users with this role
- **Outputs**: Updated role object
- **Audit**: Log role modification with affected users

#### FR-RM-003: Role Deletion
- **Description**: Delete role (if no users assigned)
- **Inputs**: role_id
- **Validations**: No active users with this role
- **Outputs**: Success confirmation
- **Audit**: Log role deletion

### 5.3 Query Execution

#### FR-QE-001: Query Execution
- **Description**: Execute SQL queries with permission checks
- **Inputs**: query, database, parameters
- **Pre-checks**:
  - Parse SQL to extract tables
  - Verify user has required permissions
  - Check for dangerous operations (DROP, TRUNCATE)
- **Execution**:
  - Set session user (db_username)
  - Execute query with timeout (30s default)
  - Limit result set (1000 rows default)
- **Outputs**: Query results, execution metadata
- **Audit**: Log query execution with full SQL

#### FR-QE-002: Query Validation
- **Description**: Validate SQL syntax without execution
- **Inputs**: query
- **Outputs**: Validation result, syntax errors
- **Audit**: Not logged (read-only operation)

#### FR-QE-003: Query History
- **Description**: Retrieve user's query history
- **Inputs**: user_id, pagination params
- **Outputs**: Paginated query history
- **Audit**: Not logged (read-only operation)

#### FR-QE-004: Save Query Script
- **Description**: Save frequently used queries
- **Inputs**: name, query, is_public
- **Outputs**: Saved script object
- **Audit**: Log script creation

### 5.4 Approval Workflows

#### FR-AW-001: Create Approval Request
- **Description**: Request temporary access or privilege escalation
- **Inputs**: type, description, requested_permissions, justification
- **Validations**: 
  - Valid permission structure
  - Justification required for production access
- **Outputs**: Created approval request
- **Notifications**: Notify approvers
- **Audit**: Log request creation

#### FR-AW-002: Approve Request
- **Description**: Approve access request (full or partial)
- **Inputs**: request_id, approved_permissions[]
- **Validations**: User has approval authority
- **Side Effects**:
  - Grant permissions to user
  - Set expiry for temporary permissions
- **Outputs**: Updated request status
- **Notifications**: Notify requester
- **Audit**: Log approval decision

#### FR-AW-003: Reject Request
- **Description**: Reject access request
- **Inputs**: request_id, rejection_reason
- **Outputs**: Updated request status
- **Notifications**: Notify requester
- **Audit**: Log rejection

#### FR-AW-004: Auto-Expiry
- **Description**: Automatically reject expired requests
- **Trigger**: Scheduled job (hourly)
- **Side Effects**: Update request status to 'rejected'
- **Audit**: Log auto-rejection

### 5.5 Audit & Compliance

#### FR-AC-001: Activity Logging
- **Description**: Log all user actions
- **Events**:
  - Authentication (login/logout)
  - Query execution
  - User/role modifications
  - Permission changes
  - Approval decisions
- **Data**: timestamp, user, action, resource, status, metadata
- **Retention**: 90 days (configurable)

#### FR-AC-002: Audit Log Query
- **Description**: Search and filter audit logs
- **Inputs**: filters (user, action, date range, status)
- **Outputs**: Paginated audit logs
- **Audit**: Log audit log access (meta-audit)

#### FR-AC-003: Compliance Reporting
- **Description**: Generate compliance reports
- **Reports**:
  - User access report
  - Privilege escalation report
  - Failed access attempts
  - Query execution summary
- **Outputs**: CSV/PDF export
- **Audit**: Log report generation

### 5.6 Schema Management

#### FR-SM-001: Schema Discovery
- **Description**: Sync database schema metadata
- **Trigger**: Manual or scheduled (daily)
- **Process**:
  - Connect to target databases
  - Query information_schema
  - Cache schema metadata
- **Outputs**: Updated schema cache
- **Audit**: Log schema sync

#### FR-SM-002: Schema Search
- **Description**: Search tables and columns
- **Inputs**: search_term
- **Outputs**: Matching tables/columns
- **Audit**: Not logged (read-only operation)

## 6. Non-Functional Requirements

### 6.1 Performance
- **Query Execution**: < 30s timeout
- **API Response Time**: < 500ms (p95)
- **Concurrent Users**: Support 100+ concurrent users
- **Database Connections**: Connection pooling (min: 10, max: 50)

### 6.2 Security
- **Encryption**: TLS 1.3 for data in transit
- **Data at Rest**: AES-256 encryption for sensitive data
- **SQL Injection**: Parameterized queries only
- **Rate Limiting**: 100 requests/minute per user
- **Session Security**: HttpOnly, Secure, SameSite cookies

### 6.3 Reliability
- **Uptime**: 99.9% availability
- **Backup**: Daily automated backups with 30-day retention
- **Disaster Recovery**: RPO: 24 hours, RTO: 4 hours

### 6.4 Scalability
- **Horizontal Scaling**: Stateless API servers
- **Database**: Read replicas for query execution
- **Caching**: Redis for session and schema caching

### 6.5 Monitoring
- **Metrics**: Prometheus + Grafana
- **Logging**: Centralized logging (ELK/Splunk)
- **Alerting**: PagerDuty integration
- **Health Checks**: /health and /ready endpoints

## 7. Integration Requirements

### 7.1 Database Connectivity
- **Supported Databases**: PostgreSQL, MySQL, SQL Server
- **Connection Management**: 
  - Encrypted credential storage
  - Connection pooling
  - Automatic reconnection
- **Multi-tenancy**: Support multiple database connections

### 7.2 SSO Integration
- **Protocols**: SAML 2.0, OAuth 2.0, OIDC
- **Providers**: Okta, Azure AD, Google Workspace, Custom
- **User Provisioning**: JIT provisioning with role mapping

### 7.3 Notification System
- **Channels**: Email, Slack, Microsoft Teams
- **Events**: 
  - Approval requests
  - Permission grants/revocations
  - Session expiry warnings
  - Security alerts

## 8. Deployment Requirements

### 8.1 Environment Configuration
- **Development**: Local Docker Compose setup
- **Staging**: Kubernetes cluster (minikube/kind)
- **Production**: Kubernetes cluster with HA

### 8.2 Infrastructure
- **Container**: Docker images
- **Orchestration**: Kubernetes
- **CI/CD**: GitHub Actions / GitLab CI
- **Secrets Management**: HashiCorp Vault / AWS Secrets Manager

### 8.3 Configuration Management
- **Environment Variables**: 12-factor app principles
- **Feature Flags**: LaunchDarkly / custom solution
- **Config Files**: YAML-based configuration

## 9. Security Considerations

### 9.1 Threat Model
- **SQL Injection**: Mitigated by parameterized queries
- **Privilege Escalation**: Approval workflows + audit logs
- **Session Hijacking**: Secure token management + IP validation
- **Data Exfiltration**: Rate limiting + query result size limits

### 9.2 Compliance
- **SOC 2**: Audit logging, access controls
- **GDPR**: Data retention policies, user data export
- **HIPAA**: Encryption, audit trails (if applicable)

## 10. Future Enhancements

### 10.1 Planned Features
- **Query Builder**: Visual query construction
- **Data Masking**: PII/sensitive data redaction
- **Query Scheduling**: Cron-based query execution
- **Collaborative Queries**: Real-time query sharing
- **AI-Powered Suggestions**: Query optimization hints

### 10.2 Advanced Security
- **MFA**: Multi-factor authentication
- **Behavioral Analytics**: Anomaly detection
- **Data Loss Prevention**: Pattern-based blocking
- **Zero Trust**: Continuous authentication

## 11. Appendix

### 11.1 Glossary
- **Session-based User**: Temporary user with auto-expiry
- **db_username**: Database-level username mapping
- **Privilege Escalation**: Requesting higher permissions
- **Approval Workflow**: Multi-step permission grant process

### 11.2 References
- PostgreSQL Documentation
- OWASP Security Guidelines
- JWT Best Practices
- RBAC Design Patterns
