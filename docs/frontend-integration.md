# Frontend Integration Guide

**Audience:** Frontend developers integrating with the SessionDB backend APIs.

## Base URL and Auth

- **Base URL:** `http://localhost:PORT/v1` (dev) or your deployed host + `/v1`
- **Auth:** Send JWT on every request: `Authorization: Bearer <access_token>`
- **Login:** `POST /v1/auth/login` with `{ "email": "...", "password": "..." }` to obtain tokens.

## Existing APIs (unchanged)

These endpoints are stable. Phase 1 (Database Dialect Layer) did **not** change any request/response contracts.

| Area | Method | Path | Notes |
|------|--------|------|--------|
| Auth | POST | `/v1/auth/login` | Returns access + refresh tokens |
| Auth | POST | `/v1/auth/register` | Register new user |
| Auth | POST | `/v1/auth/logout` | Invalidate session |
| Config | GET | `/v1/config/auth` | Auth type (password/SSO) |
| Users | GET | `/v1/users` | List users (needs `users:read`) |
| Users | POST | `/v1/users` | Create user |
| Users | GET | `/v1/users/me` | Current user profile |
| Users | GET/PUT/DELETE | `/v1/users/:id` | Get/update/delete user |
| Roles | CRUD | `/v1/roles` | Role management |
| DB credentials | CRUD | `/v1/db-users` | DB user provisioning |
| DB credentials | POST | `/v1/db-credentials/verify` | Verify credentials |
| DB roles | GET | `/v1/db-roles` | List DB roles on instance |
| Query | POST | `/v1/query/execute` | Execute SQL (body: instanceId, query) |
| Query | GET | `/v1/query/history` | Query history |
| Query | CRUD | `/v1/query/scripts` | Saved scripts |
| Instances | GET | `/v1/instances` | List instances (read) |
| Admin instances | CRUD | `/v1/admin/instances` | Manage instances |
| Schema | GET | `/v1/schema` | Instance schema (tables/columns) |
| Logs | GET/POST | `/v1/logs` | Audit logs (needs `logs:view`) |
| Approvals | CRUD | `/v1/requests` | Approval workflows (feature-gated) |
| Health | GET | `/health` | No auth |
| WebSocket | GET | `/ws` | Real-time notifications |

## New APIs (Phase 2 & 3)

### Access control (Phase 2)

- **Behavior:** Before running a query, the backend checks that the user has at least one **data-level** permission on the target instance. If they have none, the API returns `403` with code `AUTH002` and message `"no data access to this instance"`.
- **Existing endpoints:** No new routes. `POST /v1/query/execute` now enforces this; ensure users have the right permissions (or instance-level grants) before calling it.

### AI (BYOK) — Phase 3

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/ai/generate-sql` | Generate SQL from a natural-language prompt. Requires AI config (API key) to be set. |
| POST | `/v1/ai/explain` | Get a short explanation of a SQL query. |
| GET | `/v1/ai/config` | Get current user's AI config (no API key returned). |
| PUT | `/v1/ai/config` | Save AI provider config (BYOK: API key, provider type, base URL, model). |

**Generate SQL**

- **Request:** `{ "instanceId": "uuid", "prompt": "e.g. show all users" }`
- **Response:** `{ "sql": "SELECT ...", "requiresApproval": true|false }`
- **Auth:** JWT required. User must have configured an AI provider (PUT `/v1/ai/config`) first.

**Explain query**

- **Request:** `{ "query": "SELECT * FROM users" }`
- **Response:** `{ "explanation": "..." }`

**Get/put AI config**

- **GET response:** `{ "configured": true, "providerType": "openai", "modelName": "gpt-4", "baseUrl": "..." }` or `{ "configured": false }`.
- **PUT body:** `{ "providerType": "openai", "apiKey": "sk-...", "baseUrl": null, "modelName": "gpt-4" }`. API key is stored encrypted.

## Errors

- **401:** Missing or invalid token → redirect to login.
- **403:** Valid token but insufficient permission → show “Permission denied”.
- **404:** Resource not found.
- **4xx/5xx:** Body may include `code` (e.g. `REQ001`, `AUTH001`) and `message`; display `message` to the user.

## WebSocket

Connect to `/ws` (same origin). After connection you can receive broadcast messages (e.g. sync progress, monitoring alerts). Message format is JSON; structure is event-specific.

---

**Last updated:** 2026-03-09 (Phase 1 — no API changes)
