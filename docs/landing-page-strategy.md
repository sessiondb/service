# SessionDB Landing Page Strategy — Open Source First

This document defines messaging and structure for the SessionDB landing page. The goal is to **reach as many people as possible** by leading with open-source value and framing future capabilities as a **product roadmap**, not a pricing or commercial pitch. Commercialization is deferred; the page should feel like an active, community-driven project.

---

## 1. Messaging Strategy: "Community First"

- **Avoid on the landing page:** "Pro," "Essential tier," "commercial," "managed service," "upgrade," "buy," or any pricing.
- **Use instead:** "Core" vs "Roadmap," "Available now" vs "In development" / "Planned," "Beta," "Coming soon."
- **Trust line:** *"The core proxy and query engine will always remain open source. Advanced governance features are currently in development."*
- **CTA for roadmap items:** "Notify me when this is ready" (captures leads without asking for money).

---

## 2. Hero Section

- **Headline:** *The Open Source Database Proxy for Modern Teams*
- **Sub-headline:** *SessionDB gives your team secure, AI-powered access to MySQL and Postgres—without sharing a single password.*
- **Primary CTA:** [View on GitHub]
- **Secondary CTA:** [Try the Demo]

---

## 3. Architecture (How It Works)

Simple 3-step visual:

1. **Connect** — Link your MySQL and Postgres instances.
2. **Proxy** — SessionDB masks credentials and acts as the secure gateway.
3. **Govern** — Approve requests, audit every query, and use AI to write SQL (BYOK).

---

## 4. Feature Grid — Open Source First

### Available Now (The Open Source Core)

Use fully colored cards. These are what users can use today.

| Feature | One-line description |
|--------|----------------------|
| **Secure Query Runner** | Browser-based SQL execution for MySQL and Postgres with permission checks and history. |
| **Schema Discovery** | Automatic metadata sync: see tables, columns, and types without exposing DB credentials. |
| **Identity & User Management** | Users, roles, and granular permissions so only the right people reach the gateway. |
| **Data-Level Access Control** | Control access by instance → database → schema → table → column. |
| **Audit Logging** | Full transparency: every query and access change is logged for compliance. |
| **Approval Workflow** | Human-in-the-loop: request database access and credentials; approvers grant or deny. |
| **DB User Provisioning** | Create and manage database users and credentials through the gateway. |
| **Instance & AI Config** | Connect your DBs and bring your own AI keys (BYOK) for SQL generation and explanation. |
| **Multi-Database Support** | One gateway for PostgreSQL and MySQL via a pluggable dialect layer. |
| **Query History & Scripts** | Run SQL, save scripts, and keep a searchable history of executions. |

### The Roadmap (In Development / Planned)

Use slightly desaturated cards or a subtle "Planned" / "Beta" badge. No pricing or "Pro" language.

| Feature | One-line description | Badge |
|--------|----------------------|--------|
| **Live Session Management** | Monitor and terminate active DB connections in real time. | `PLANNED` or `BETA` |
| **Alerting & Metrics** | Health dashboards and notifications for database anomalies. | `PLANNED` |
| **Reporting** | Automated insights and query performance history. | `PLANNED` |
| **TTL & Time-Based Access** | Time-bound credentials and expiring grants for temporary access. | `PLANNED` |

**Teasing roadmap items:**

- **Badge:** Light blue or purple badge: `BETA` or `PLANNED` (no "Pro" or "Premium").
- **CTA:** "Notify me when this is ready" (email capture for future use).
- **Footnote:** *"The core proxy and query engine will always remain open source. Advanced governance features are currently in development."*

---

## 5. Features You Had vs. What Was Added (From Codebase)

Your original "Available Now" list is aligned with the codebase. The following were **added** so the landing page matches what’s actually there and sounds open-source first:

- **Schema Discovery** — Implemented (metadata sync, WebSocket progress). Strong differentiator: "see your schema without sharing credentials."
- **Data-Level Access Control** — Not just "basic RBAC"; instance → DB → table → column is in community (see `docs/features/access-control.md`). Frame as a core feature.
- **DB User Provisioning** — CRUD for DB users and credentials via the gateway; part of the core.
- **Multi-Database Support** — Postgres + MySQL via the dialect layer; one gateway, multiple engines.
- **Query History & Scripts** — Part of the query runner; worth calling out for daily use.

**Roadmap** — Your list (Live Sessions, Alerting, Reporting, TTL) matches the premium engines (session, alert, report) and design docs. Kept as "Planned" with no commercial wording.

---

## 6. Open Source Call to Action

- **Line:** *"SessionDB is built by and for the community. Star us on GitHub to follow our progress toward the 1.0 release."*
- **Optional:** GitHub Star button with real-time star count for social proof.

---

## 7. Proposed One-Page Structure

| Section | Content |
|--------|---------|
| **1. Hero** | Headline, sub-headline, [View on GitHub], [Try the Demo]. |
| **2. How It Works** | Connect → Proxy → Govern (3-step visual). |
| **3. Feature Grid** | "Available Now" (full-color cards) + "Roadmap" (desaturated + badge + "Notify me"). |
| **4. Open Source CTA** | "Built by and for the community" + Star on GitHub. |

Optional later: **Installation** (Docker Compose one-liner, link to docs), **Architecture** (high-level diagram), **FAQ**.

---

## 8. Frontend Alignment

The app UI (feature gates, upgrade prompts, audit export, and roadmap feature pages) follows this strategy:

- **FeatureGate / UpgradePrompt:** Headings say "Coming soon"; body says "on our roadmap and not yet available"; CTA is "Notify me when this is ready" (no "Upgrade" or "Pro").
- **AccessContext:** Roadmap features use `minimumPlan: 'Planned'` and `reason: 'feature_in_development'` (no "Pro" or "Enterprise" in default feature config).
- **Audit export:** Copy uses "Planned" and "coming soon"; no "Upgrade to Pro."
- **Page titles:** Sessions, Alerts, Reports show "(Planned)" instead of "(Premium)."

Internal names (e.g. `PremiumRegistry`, `UpgradePrompt` component, `FeatureGate`) are unchanged; only user-visible copy and default feature labels were updated.

---

## 9. Wording to Avoid vs. Use

| Avoid | Use |
|-------|-----|
| Pro / Premium / Essential tier | Roadmap / In development / Planned |
| Upgrade / Buy / Pricing | Notify me when ready / Star / Contribute |
| Commercial / Managed service | Community / Open source core |
| "Pro features" | "Advanced governance features (in development)" |
| "Will launch Pro" | "We’re building these next" / "On our roadmap" |

---

## 10. Technical Notes for the Page

- **Domain:** e.g. `sessiondb.io` if available.
- **Stack:** Astro (or existing React) for a fast, roadmap-style landing page.
- **GitHub:** Star button with live count for trust.
- **Waitlist:** "Notify me when Beta launches" or "Notify me when [feature] is ready" to capture emails without a sales pitch.

---

## 11. Installation Section (Optional for Landing)

If you add a short "Get started" block:

- **Docker (recommended):** `docker compose up -d` — backend + metadata DB (and UI if in same compose).
- **Go:** `go run ./cmd/server` or `go build -o sessiondb ./cmd/server`.
- Link to full instructions in the docs (e.g. README or `docs/README.md`).

This keeps the landing page focused on value and roadmap while still giving a clear path to run the open source core.
