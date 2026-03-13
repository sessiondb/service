# Request / Approval Flow (DB Access)

## Purpose

Let users **request** DB access (instance, database, table, privileges). Admins **approve** or **reject** requests. On approve, the backend creates Permission records and provisions the DB user with the requested grants so the requester gets access without further manual steps.

## Request shape

Each request can ask for multiple (instance, database, table, privileges):

- **`requestedItems`** — array of:
  - `instanceId` (UUID)
  - `database` (string)
  - `table` (string)
  - `privileges` (array of strings, e.g. `["SELECT", "INSERT"]`)

One submitted form = one request with one or more items.

## API

- **Create request:** `POST /v1/requests` — body: `type`, `description`, `justification`, `requestedItems`. Requester is the authenticated user. Each item is validated (non-empty fields, instance exists).
- **List requests:** `GET /v1/requests` — returns all requests; each includes `requestedItems` so admins can see what was requested.
- **Approve/Reject:** `PUT /v1/requests/:id` — body: `status` (`"approved"` or `"rejected"`), and optionally `rejectionReason` when rejecting.

## On approve (backend)

When an admin sets status to **approved**:

1. For each item in `requestedItems`:
   - A **Permission** record is created for the requester (user, instance, database, table, privileges).
   - The requester’s **DB user** on that instance is ensured (provisioned if missing).
   - **Grants** are applied on the target database (e.g. `GRANT SELECT, INSERT ON table TO user`).
2. If any step fails, the request is reverted to **pending** and an error is returned (all-or-nothing).

## Frontend entry points

1. **No-access screen** — Shown when the user has no instance access. Message + “Request access” opens the request form (no prefill). After submit, “Refresh” or “Check my access” refetches permissions.
2. **Query editor** — “Request access” (or “Request more access”) button opens the same form with instance/database/table prefilled from the current context. After submit, schema/instance list can be refreshed.

## Admin

- **Requests page** lists all requests and shows `requestedItems` (instance, database, table(s), privileges).
- **Refresh** button refetches `GET /v1/requests` to show new or updated requests.
- **Approve** / **Reject** per request; optional rejection reason when rejecting. On success, the list is refreshed.

## Limits

- Approve/reject is all-or-nothing per request (no partial approve of individual items in this version).
- Request creation and list/approve/reject are gated by the existing approval feature and permissions (e.g. `advanced_approvals` feature gate, `PermApprovalsManage`).
