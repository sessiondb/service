# TOML Config, Default Logins, and Request/Approval Flow — Design

**Status:** Approved (Sections 1–3).  
**Implementation plan:** `docs/plans/2026-03-12-toml-approval-request-access.md`

---

## Section 1: TOML config and default platform logins (backend)

- **TOML path:** `config.default.toml` at repo root (or `CONFIG_TOML_PATH`). Structure: `[server]`, `[auth]`, `[[auth.default_logins]]` (email, password, role_key).
- **Precedence:** Env > TOML > code defaults. Load TOML after .env in `LoadConfig()`; extend Config with `DefaultLogins`.
- **Seed once:** After DB connect and migrations, if no platform users exist (`SELECT COUNT(*) FROM users = 0`), create users from `DefaultLogins` (hash password, set role by role_key). No updates to existing users. Option: `SKIP_DEFAULT_LOGINS` env to disable in production.

---

## Section 2: Approval flow backend

- **Request shape:** One `ApprovalRequest` per submission. Add `requested_items` (JSONB): array of `{ instance_id, database, table, privileges[] }`. API accepts/returns this; frontend sends only new shape.
- **Create:** `POST /v1/requests` — body: type, description, justification, requested_items. Validation: at least one item; each has instance_id, database, table, privileges (non-empty); instance exists.
- **Approve side effects:** On approve: for each item create Permission for requester (user_id, instance_id, database, table, privileges); ensure DB user exists (provisioning); run dialect grants. All-or-nothing: on failure roll back and return error.
- **Reject:** No permission/DB changes; set status, reviewed_by, reviewed_at, rejection_reason.
- **List:** `GET /v1/requests` includes requested_items in response.

---

## Section 3: Frontend

- **Shared form:** One component: instance (dropdown), database (text/dropdown), table(s) (multi or repeatable), privileges (checkboxes), description/justification (required). Submit → `POST /v1/requests` with requested_items.
- **No-access screen:** Shown when user has no instance access. Message + “Request access” opens shared form (no prefill). After submit, “Refresh”/“Check my access” refetches permissions.
- **In-editor:** “Request access” button in query editor; opens same form with instance/database/table prefilled from context. After submit, optional refresh of schema/instance list.
- **Admin:** Requests page shows requested_items; Refresh button refetches `GET /v1/requests`; Approve/Reject per request with list refresh on success.
