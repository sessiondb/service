# Access Control (Data-Level)

## Purpose

Control which **data** a user can read or write: instance → database → schema → table → column. This is separate from **system RBAC** (who can manage users, view logs, etc.).

## How it works

- **Permission model:** Each grant has `instanceId`, `database`, `schema`, `table`, optional `column`, and `privileges` (e.g. SELECT, INSERT).
- **Query execution:** Before running a query, the backend checks that the user has at least one permission on the target instance. If they have none, the API returns `403` with "no data access to this instance".
- **Database enforcement:** When permissions are synced to the target DB (via DB user provisioning), column-level grants are applied so the database itself rejects unauthorized column access.

## Configuration

- Permissions are attached to **users** (direct) or **roles** (all users with that role). Use existing user/role APIs and include `permissions` with `instanceId` when creating or updating.
- For column-level access, set `column` on the permission and ensure the instance has a provisioned DB user so the dialect can run `GRANT SELECT (col1, col2) ON table TO user`.

## Limits

- Column-level enforcement is accurate at the DB layer (Postgres/MySQL); application pre-check is best-effort for friendly errors.
- Legacy permissions without `instanceId` are treated as matching any instance for backward compatibility.
