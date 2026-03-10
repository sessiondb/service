# Frontend Integration (Phases 1–3) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Integrate the backend behavior from Phases 1–3 (Dialect Layer, Access Engine, AI BYOK) into the SessionDB UI so the app correctly uses existing APIs, handles new errors, and adds AI config and generate/explain flows. The plan is structured so that when later phases (e.g. Session Engine, Alerting, Reporting) land, the same patterns can be reused.

**Architecture:** The UI is assumed to be a React 18 + TypeScript app (Vite, React Router, Context for state). Integration is done by: (1) centralizing API calls and error handling, (2) handling Phase 2’s 403 “no data access” on query execute, (3) adding AI API client and screens for config, generate-SQL, and explain. All paths below are relative to the **UI project root** (your React app—e.g. a `frontend/` folder in this repo or a separate repo).

**Tech Stack:** React 18, TypeScript, Vite, React Router v6, fetch or axios (per project rules: use axios), JWT in `Authorization: Bearer <token>`.

**Reference:** Backend contract and error shapes are in `docs/frontend-integration.md` (this repo).

---

## Task 1: API client base and auth

**Files:**
- Create: `src/api/client.ts` (or `src/services/api.ts` if you already have a services layer)
- Modify: Wherever you store the auth token (e.g. `src/context/AuthContext.tsx` or `src/store/auth.ts`)

**Step 1: Create the API client**

Create an axios instance that sends the JWT and uses the base URL from env.

```typescript
// src/api/client.ts
import axios, { AxiosError } from 'axios';

const baseURL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/v1';

export const api = axios.create({
  baseURL,
  headers: { 'Content-Type': 'application/json' },
});

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('accessToken'); // or your auth context
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

api.interceptors.response.use(
  (r) => r,
  (err: AxiosError<{ code?: string; error?: string }>) => {
    if (err.response?.status === 401) {
      // Clear token and redirect to login
      localStorage.removeItem('accessToken');
      window.location.href = '/login';
    }
    return Promise.reject(err);
  }
);
```

**Step 2: Export typed error helper**

In the same file or `src/api/errors.ts`:

```typescript
export function getApiErrorMessage(err: unknown): string {
  if (axios.isAxiosError(err) && err.response?.data?.error) {
    return err.response.data.error;
  }
  return err instanceof Error ? err.message : 'Request failed';
}

export function getApiErrorCode(err: unknown): string | undefined {
  if (axios.isAxiosError(err) && err.response?.data?.code) {
    return err.response.data.code;
  }
  return undefined;
}
```

**Step 3: Commit**

```bash
git add src/api/client.ts src/api/errors.ts
git commit -m "feat(ui): add API client with JWT and error helpers"
```

---

## Task 2: Handle Phase 2 — 403 “no data access” on query execute

**Files:**
- Modify: The component or service that calls `POST /v1/query/execute` (e.g. `src/pages/Query.tsx` or `src/services/queryService.ts`)

**Step 1: Use the API client for execute**

Ensure query execute uses the shared `api` and does not swallow 403.

Example (adjust to your file):

```typescript
// Example: src/services/queryService.ts
import { api } from '../api/client';
import { getApiErrorMessage, getApiErrorCode } from '../api/errors';

export async function executeQuery(instanceId: string, query: string) {
  const { data } = await api.post<{ columns?: string[]; rows?: unknown[][] }>('/query/execute', {
    instanceId,
    query,
  });
  return data;
}
```

**Step 2: Handle 403 and show “no data access” message**

Where you call `executeQuery`, catch errors and show a user-friendly message when the backend returns 403 with code `AUTH002` (no data access to this instance).

```typescript
// In the component that runs the query:
try {
  const result = await executeQuery(selectedInstanceId, queryText);
  setResult(result);
} catch (err) {
  const code = getApiErrorCode(err);
  const message = getApiErrorMessage(err);
  if (axios.isAxiosError(err) && err.response?.status === 403 && code === 'AUTH002') {
    setError('You don’t have data access to this instance. Ask an admin to grant you permissions.');
  } else {
    setError(message);
  }
}
```

**Step 3: Commit**

```bash
git add src/services/queryService.ts src/pages/Query.tsx
git commit -m "feat(ui): handle 403 no data access on query execute (Phase 2)"
```

---

## Task 3: AI API client (Phase 3)

**Files:**
- Create: `src/api/ai.ts`

**Step 1: Add types**

```typescript
// src/api/ai.ts
import { api } from './client';

export type AIConfig = {
  configured: boolean;
  providerType?: string;
  modelName?: string;
  baseUrl?: string | null;
};

export type GenerateSQLResponse = { sql: string; requiresApproval: boolean };
export type ExplainResponse = { explanation: string };
```

**Step 2: Implement AI API functions**

```typescript
export async function getAIConfig(): Promise<AIConfig> {
  const { data } = await api.get<AIConfig>('/ai/config');
  return data;
}

export async function updateAIConfig(params: {
  providerType: string;
  apiKey: string;
  baseUrl?: string | null;
  modelName: string;
}): Promise<void> {
  await api.put('/ai/config', params);
}

export async function generateSQL(instanceId: string, prompt: string): Promise<GenerateSQLResponse> {
  const { data } = await api.post<GenerateSQLResponse>('/ai/generate-sql', { instanceId, prompt });
  return data;
}

export async function explainQuery(query: string): Promise<ExplainResponse> {
  const { data } = await api.post<ExplainResponse>('/ai/explain', { query });
  return data;
}
```

**Step 3: Commit**

```bash
git add src/api/ai.ts
git commit -m "feat(ui): add AI API client (Phase 3)"
```

---

## Task 4: AI config screen (Phase 3)

**Files:**
- Create: `src/pages/SettingsAIConfig.tsx` (or under `src/pages/Settings/`)
- Modify: App router (e.g. `src/App.tsx` or `src/routes.tsx`) to add route `/settings/ai` or `/ai-config`

**Step 1: Add route**

Add a protected route, e.g. `Route path="settings/ai" element={<SettingsAIConfig />} />`.

**Step 2: Implement AI config page**

- On mount: call `getAIConfig()` and show current state (configured yes/no, provider type, model, base URL; never show API key).
- Form: provider type (e.g. openai, anthropic, custom), API key (password input), optional base URL, model name. Submit calls `updateAIConfig(...)`.
- On success: show “Saved” and optionally refetch `getAIConfig()`.
- On error: show `getApiErrorMessage(err)`.

**Step 3: Commit**

```bash
git add src/pages/SettingsAIConfig.tsx src/App.tsx
git commit -m "feat(ui): add AI config page (Phase 3)"
```

---

## Task 5: Generate SQL from prompt in query UI (Phase 3)

**Files:**
- Modify: Query / SQL editor page (e.g. `src/pages/Query.tsx` or the component that holds the query textarea and run button)

**Step 1: Add “Generate with AI” entry point**

Add a button or link “Generate with AI” (or an icon) that opens a small modal or inline section with:
- A text input for the natural-language prompt.
- Instance selector (reuse the same instance dropdown you use for execute), or pass the currently selected instance.

**Step 2: Call generate SQL and show result**

- On submit: call `generateSQL(selectedInstanceId, prompt)`.
- If the user has not configured AI (`getAIConfig().configured === false`), show a message: “Configure your AI provider in Settings → AI” and link to the AI config page.
- On success: put the returned `sql` into the query editor and optionally show a note if `requiresApproval` is true (e.g. “This query may require approval before running”).
- On error: show `getApiErrorMessage(err)` (e.g. invalid request or missing config).

**Step 3: Commit**

```bash
git add src/pages/Query.tsx
git commit -m "feat(ui): generate SQL from prompt (Phase 3)"
```

---

## Task 6: Explain query (Phase 3)

**Files:**
- Modify: Same query/SQL editor page or a shared “query actions” component

**Step 1: Add “Explain” action**

Add an “Explain” (or “What does this do?”) button that is enabled when the query textarea has content.

**Step 2: Call explain and show result**

- On click: call `explainQuery(currentQueryText)`.
- If the user has not configured AI, show the same “Configure your AI provider in Settings → AI” message.
- On success: show the returned `explanation` in a tooltip, modal, or inline collapse.
- On error: show `getApiErrorMessage(err)`.

**Step 3: Commit**

```bash
git add src/pages/Query.tsx
git commit -m "feat(ui): explain query with AI (Phase 3)"
```

---

## Task 7: Optional — instance-scoped permissions in user/role UI (Phase 2)

**Files:**
- Modify: User create/edit and/or role create/edit forms (wherever you send `permissions` to the backend)

**Step 1: Include instanceId in permission payloads**

Backend Phase 2 expects data-level permissions to have `instanceId`. Ensure that when you create or update a user or role with permissions, each permission object includes `instanceId` (UUID of the target instance) when it’s a data permission. If your UI only manages system RBAC and no data-level permissions yet, you can skip this task and add it when you add instance-scoped permission management.

**Step 2: Commit (if done)**

```bash
git add src/pages/Users.tsx src/pages/Roles.tsx
git commit -m "feat(ui): send instanceId for data permissions (Phase 2)"
```

---

## Task 8: Document and future phases

**Files:**
- Create or modify: `docs/README.md` or `FRONTEND.md` in the UI repo

**Step 1: Add a short “Backend integration” section**

- Point to the backend’s `docs/frontend-integration.md` for base URL, auth, and all endpoints.
- Note that 403 with code `AUTH002` means “no data access to this instance” (Phase 2).
- List the AI endpoints and that AI features require the user to set API config first (Phase 3).
- Add one line: “When new backend phases (e.g. Session Engine, Alerting, Reporting) are completed, add corresponding API client functions and screens following the same patterns (api module, error handling, settings/feature screens).”

**Step 2: Commit**

```bash
git add docs/README.md
git commit -m "docs: backend integration and future phases"
```

---

## Summary checklist

| Task | What |
|------|------|
| 1 | API client (axios + JWT + 401 redirect) and error helpers |
| 2 | Query execute uses client; handle 403 AUTH002 “no data access” |
| 3 | AI API client (get/put config, generate-sql, explain) |
| 4 | AI config page (GET/PUT /ai/config) |
| 5 | “Generate with AI” in query UI using selected instance |
| 6 | “Explain” in query UI |
| 7 | (Optional) instanceId in permission payloads for user/role |
| 8 | Docs: link to frontend-integration.md and note for future phases |

---

## Testing

- **Manual:** Run backend (`go run ./cmd/server` or Docker) and UI (`npm run dev`). Log in, open query page, run a query; revoke data access for that instance (or use a user with no instance permissions) and confirm 403 message. Configure AI in settings, then use “Generate with AI” and “Explain” and confirm requests and responses match `docs/frontend-integration.md`.
- **E2E (if present):** Add or extend a test that calls the API client and asserts on 403 and AI response shapes.

---

## When later phases land

- Add new API modules (e.g. `src/api/session.ts`, `src/api/alerts.ts`, `src/api/reports.ts`) and new routes/pages as needed.
- Reuse the same `api` client, `getApiErrorMessage` / `getApiErrorCode`, and the pattern: feature screen → API call → success/error handling.
