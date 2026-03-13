# Frontend Tasks 6–9: Copy-Paste Prompts

Use these in the **frontend** SessionDB workspace (the repo that contains `src/`). Run one task at a time; each prompt is self-contained.

**API base:** `POST /v1/requests`, `GET /v1/requests`, `PUT /v1/requests/:id`. Auth: include JWT (e.g. Bearer token). Request body for create: `{ type, description, justification, requestedItems: [{ instanceId, database, table, privileges }] }`. Reject body: `{ status: "rejected", rejectionReason?: string }`.

---

## Task 6: Shared Request DB Access form component

**Paste this into the frontend session:**

```
Implement Task 6 from the SessionDB implementation plan (docs/plans/2026-03-12-toml-approval-request-access.md).

Create:
1. src/api/requests.ts — axios helpers: createRequest(body: { type: string; description: string; justification: string; requestedItems: Array<{ instanceId: string; database: string; table: string; privileges: string[] }> }) calling POST /v1/requests with auth; getRequests() calling GET /v1/requests; updateRequestStatus(id, status, rejectionReason?) calling PUT /v1/requests/:id. Add JSDoc. Use axios (not fetch/request).

2. src/components/RequestAccessForm.tsx (or under src/features/ or src/pages/) — Shared "Request DB Access" form. Props: optional initialInstanceId, initialDatabase, initialTable (for prefill from query editor). State: instanceId, database, tables (array of table names), privileges (selected: e.g. SELECT, INSERT, UPDATE, DELETE), description, justification. Fetch instances list from existing instances API. UI: dropdown for instance; text or dropdown for database; multi-select or repeatable rows for table(s); checkboxes for privileges; required description and justification. On submit: build requestedItems (one object per table: same instanceId, database, privileges, table from list), call createRequest(), onSuccess run callback (e.g. onSuccess()) to close modal and show toast. Add JSDoc. Follow project rules: functions only, no new classes; proper JSDoc.

Verify: Run frontend, open form (e.g. temporary button), submit and confirm POST /v1/requests with correct body.
```

---

## Task 7: No-access screen (refresh screen)

**Paste this into the frontend session:**

```
Implement Task 7 from the SessionDB implementation plan.

Create or update a no-access page (e.g. src/pages/NoAccess.tsx) that shows when the user has no permissions to any instance (e.g. after login, if user has no instance access or no data permissions). Content: short message "You don't have access to any database yet" and a button "Request access". Click opens the shared RequestAccessForm (modal or inline). After successful submit, show success message and a "Refresh" or "Check my access" button that refetches user permissions or instance list and, if the user now has access, redirects to query editor or home.

Wire the route: add route for this page (e.g. /no-access) and ensure users with no instance access are directed here (e.g. from login or from a guard on the query editor route). Use the RequestAccessForm component from Task 6.
```

---

## Task 8: In-editor "Request access" button

**Paste this into the frontend session:**

```
Implement Task 8 from the SessionDB implementation plan.

In the query editor (e.g. src/pages/Query/Editor.tsx or wherever instance/database/table selector lives), add a "Request access" or "Request more access" button. On click, open the shared RequestAccessForm in a modal with prefill: initialInstanceId, initialDatabase, initialTable from the current editor context (selected instance, database, table). On submit success, optionally refetch instance/database/schema list so newly granted access appears in selectors.

Use the same RequestAccessForm component; pass prefill props from the current selection state.
```

---

## Task 9: Admin requests list and Refresh

**Paste this into the frontend session:**

```
Implement Task 9 from the SessionDB implementation plan.

In the admin requests/approvals page (e.g. src/pages/Admin/Approvals.tsx):
1. Ensure the list uses GET /v1/requests and displays requestedItems for each request: show instance name (resolve from instanceId if you have instance list), database, table(s), and privileges in a table or expandable row.
2. Add a "Refresh" button that refetches GET /v1/requests and updates the list.
3. When rejecting, send optional rejectionReason from a text field in the reject modal/flow. Call PUT /v1/requests/:id with body { status: "rejected", rejectionReason: "<user input or empty>" }.
4. On approve or reject success, refresh the list (e.g. call getRequests again).

Use axios; add JSDoc. Follow existing patterns in the admin section.
```

---

**Reference:** Design doc: `docs/plans/2026-03-12-toml-approval-request-access-design.md` (in backend repo). API: POST/GET/PUT `/v1/requests` as in backend `docs/api-documentation.md` and `docs/features/request-approval-flow.md`.
