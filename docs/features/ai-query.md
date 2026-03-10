# AI Query (BYOK)

## Purpose

Generate SQL from natural language and get query explanations using the user's own AI provider API key (BYOK), or an organization-level key set by an admin.

## How it works

1. **Configure:** User saves their API key and provider (OpenAI-compatible) via `PUT /v1/ai/config`. Admins can set an organization-level config via `PUT /v1/admin/ai-config` (stored as global AI config).
2. **Resolve order:** For each AI request (generate SQL, explain), the backend uses the **user's config first**; if the user has none, it falls back to the **organization (global) config**. So a user can override the org key with their own.
3. **Generate SQL:** `POST /v1/ai/generate-sql` with `instanceId` and `prompt`. The backend builds a schema context (tables/columns the user is allowed to see) and calls the provider. **Generated SQL is stripped of markdown code fences** (e.g. ```sql ... ```) so the editor shows plain SQL. Returns `sql` and `requiresApproval`.
4. **Explain:** `POST /v1/ai/explain` with a `query` returns a short explanation.
5. **Execute:** The frontend can send the generated SQL to `POST /v1/query/execute`; if `requiresApproval` is true, the UI should obtain approval before executing.
6. **Usage:** Token usage is recorded per request (input/output tokens, model, feature). Users see their own usage via `GET /v1/ai/usage`; admins see global and per-user usage via `GET /v1/admin/ai/usage`.

## Configuration

- **GET /v1/ai/config:** Returns the effective config and `source`: `"user"` or `"global"` so the UI can show "Using your key" or "Using organization key".
- **PUT /v1/ai/config:** User's own config. **PUT /v1/admin/ai-config:** Organization (global) config. Both store `providerType`, `apiKey` (encrypted), optional `baseUrl`, `modelName`.
- **Policies (optional):** `AIExecutionPolicy` per instance and action type (SELECT, UPDATE, DDL, etc.) controls whether execution requires approval or is allowed for certain roles.

## Usage dashboard

- **User:** "Your requests (last 30 days)" and per-feature breakdown (generate-sql, explain).
- **Admin:** Global request count and a table of usage by user (user ID, request count, input/output tokens).

## Limits

- Community edition requires users or the organization to bring their own API key.
- Only OpenAI-compatible chat APIs are supported (same request/response shape).
