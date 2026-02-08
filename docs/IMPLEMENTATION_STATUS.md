# SessionDB - Implementation Status

This document tracks the technical maturity of the core responsibilities outlined in our system design.

## Current Maturity Matrix

| Feature Area | Status | Implementation Details |
| :--- | :--- | :--- |
| **Authentication** | ✅ Done | Password-based login (bcrypt), JWT tokens, and basic logout are implemented. |
| **RBAC / Models** | ✅ Done | `User`, `Role`, and `Permission` models are defined with UUIDs and relationships. |
| **Audit Logging** | ✅ Done | `AuditLog` and `QueryHistory` repositories exist. `ExecuteQuery` saves history asynchronously. |
| **User Mapping** | ⚠️ Partial | Model supports `db_username`, but `ExecuteQuery` currently defaults to a single administrative connection for the MVP. |
| **Instance Management** | ✅ Done | APIs for CRUD and Sync trigger implemented. |
| **Metadata Sync** | ✅ Done | Engine implemented with Postgres scrapers, WebSocket progress updates, and background `SyncWorker`. |
| **Permission Expiry** | ❌ Todo | Models support `ExpiresAt`, but the background worker/ticker to automatically revoke access is not yet implemented. |
| **Approval Flow** | ✅ Done | `ApprovalHandler` and `Repository` are implemented, supporting the creation and review of access requests. |

## Technical Implementation Notes

### Audited Execution Path
- **Current Flow**: `QueryHandler` -> `QueryService.ExecuteQuery` -> `QueryRepository.SaveHistory` (Goroutine).
- **Enhancement Needed**: Switch from administrative DSN to dynamic DSN using the user's mapped `db_username`.

### Access Controls
- **Permission Model**: Uses a `Type` field (permanent/temp/expiring). 
- **Validation**: Permissions are checked during the query execution phase (currently a TODO in `internal/service/query_service.go:28`).

### Background Tasks
- **Requirement**: A global worker to monitor `Permission.ExpiresAt` and `User.SessionExpiresAt` is required to fulfill the "auto-revocation" responsibility.
