# AI Query (BYOK)

## Purpose

Generate SQL from natural language and get query explanations using the user’s own AI provider API key (BYOK).

## How it works

1. **Configure:** User saves their API key and provider (OpenAI-compatible) via `PUT /v1/ai/config`.
2. **Generate SQL:** `POST /v1/ai/generate-sql` with `instanceId` and `prompt`. The backend builds a schema context (tables/columns the user is allowed to see) and calls the provider to generate SQL. Returns `sql` and `requiresApproval`.
3. **Explain:** `POST /v1/ai/explain` with a `query` returns a short explanation.
4. **Execute:** The frontend can send the generated SQL to `POST /v1/query/execute`; if `requiresApproval` is true, the UI should obtain approval before executing.

## Configuration

- **GET/PUT /v1/ai/config:** Store `providerType`, `apiKey` (encrypted), optional `baseUrl`, `modelName`. One config per user.
- **Policies (optional):** `AIExecutionPolicy` per instance and action type (SELECT, UPDATE, DDL, etc.) controls whether execution requires approval or is allowed for certain roles.

## Limits

- Community edition requires users to bring their own API key.
- Only OpenAI-compatible chat APIs are supported (same request/response shape).
