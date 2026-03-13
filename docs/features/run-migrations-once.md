# Running Migrations via Admin API

## Purpose

SessionDB does **not** run migrations on app startup. Migrations are run by calling the **admin migrate API** after the server is up (e.g. from Docker Compose, CI, or a Kubernetes Job).

## Migrate API

- **Endpoint:** `POST /v1/migrate`
- **Auth:** Token must match the `MIGRATE_TOKEN` env var on the server:
  - Header: `X-Migrate-Token: <your-secret>`, or
  - Header: `Authorization: Bearer <your-secret>`
- **Env:** Set `MIGRATE_TOKEN` on the app container. If unset, the endpoint returns 503 (disabled).

```bash
curl -X POST http://localhost:8080/v1/migrate -H "X-Migrate-Token: your-secret"
```

## Docker Compose

The main `docker-compose.yml` includes a `migrate-via-api` service that runs after the app is up: it waits for `http://app:8080/health`, then calls `POST /v1/migrate` with the token.

```bash
MIGRATE_TOKEN=your-secret docker compose up -d
```

Set `MIGRATE_TOKEN` when running so both the app and the migrate step use the same secret.

## Kubernetes

Run a Job that waits for the app then calls the migrate API (same pattern as the compose service), or call the API from your CI after deploy.

The app **never** runs migrations in `main.go`; migration is only triggered by calling the migrate API (or the `./sessiondb migrate` subcommand in a one-off container).
