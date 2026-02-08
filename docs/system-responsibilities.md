# SessionDB - System Responsibilities

SessionDB is designed as a secure, audited gateway for database interactions. Its primary objective is to mediate between users (developers, analysts, etc.) and target databases, ensuring that direct credentials are never exposed and all activities are fully governed.

## Primary Responsibilities

### 1. Secure Access Gateway
*   **Credential Masking**: The platform holds target database credentials. Users never see or handle DB passwords.
*   **User Mapping**: Every "Platform User" is mapped to a specific `db_username`. The platform uses this mapping to establish connections on behalf of the user.
*   **Session Management**: Supports temporary "guest" access that automatically expires after a predefined duration.

### 2. Granular Access Control
*   **Role-Based Access Control (RBAC)**: Baseline permissions are governed by roles (e.g., Developer, Analyst).
*   **Expiring Privileges**: Supports "Normal Users" with temporary table-level access. For example:
    *   Permanent access to Table A.
    *   Temporary (expiring) access to Table B and Table C.
*   **Automatic Revocation**: The service is responsible for revoking temporary privileges once the provided time expires.

### 3. Metadata and Discovery Layer
*   **Local Metadata Sync**: To minimize load on target databases, SessionDB syncs metadata (schemas, tables, columns, privileges, roles) into its own metadata repository.
*   **Sync Engine**: Uses background tickers or triggers to keep the local metadata cache fresh (avoiding frequent hits to `information_schema` on client DBs).

### 4. Audit and Compliance
*   **Full Execution History**: Every SQL query executed through the platform is logged with its full query text, timestamp, execution duration, and the platform user responsible.
*   **System Audit**: Logs administrative actions such as role changes, permission grants, and approval decisions.

### 5. Approval Workflow
*   **Privilege Escalation**: Users can request temporary or permanent upgrades to their permissions.
*   **Administrative Oversight**: Admins review, approve, or reject these requests, with all decisions documented in the audit trail.
