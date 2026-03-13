# Deploying SessionDB on Google Cloud (Cloud Run & App Engine)

## Overview

You can run SessionDB on **Google Cloud Run** or **App Engine (Flexible)** using the same container image. The app listens on the port provided by the platform (`PORT` env var).

**Requirements:** A PostgreSQL database (e.g. [Cloud SQL](https://cloud.google.com/sql/docs/postgres)).

---

## Migrations

The app **does not** run migrations on startup. The Docker image **entrypoint** runs migrations once when the container starts, then starts the server. So by default you get one migration run per container start.

**Alternatives:**

- **Run migration via API:** Set `RUN_MIGRATE=0` so the entrypoint skips migration, then call `POST /v1/migrate` with header `X-Migrate-Token: <MIGRATE_TOKEN>`. Set `MIGRATE_TOKEN` on the server. A Job or CI step can call this once after deploy. See [Run migrations once](./features/run-migrations-once.md).
- **Run migration in a Job:** Use a one-off Job that runs `./sessiondb migrate` (same image, same DB env).

---

## Cloud Run

### Build and push image

```bash
export PROJECT_ID=your-project
export REGION=us-central1
export IMAGE=$REGION-docker.pkg.dev/$PROJECT_ID/your-repo/sessiondb:latest

docker build -t $IMAGE .
docker push $IMAGE
```

### Deploy service

By default the image entrypoint runs migrations once then starts the server. Set DB and JWT env (and optional `MIGRATE_TOKEN` if you use the migrate API).

```bash
gcloud run deploy sessiondb \
  --image $IMAGE \
  --region $REGION \
  --set-env-vars "DB_HOST=..." \
  --set-env-vars "DB_USER=sessiondb" \
  --set-env-vars "DB_NAME=sessiondb" \
  --set-env-vars "DB_SSLMODE=require" \
  --set-secrets "DB_PASSWORD=db-password:latest" \
  --set-secrets "JWT_SECRET=jwt-secret:latest"
```

To run migrations via API instead of the entrypoint, set `RUN_MIGRATE=0` and `MIGRATE_TOKEN`, then call `POST /v1/migrate` from a Job or CI.

### Port

Cloud Run sets `PORT`; the app uses it automatically.

---

## App Engine Flexible

Use the same Docker image. Set `PORT` via the runtime (default). For “migrate once”, run a one-off task or Job that executes `./sessiondb migrate`, and use `RUN_MIGRATE=0` and call `POST /v1/migrate` (with `MIGRATE_TOKEN`) from a one-off task.

---

## Summary

| Topic            | Detail |
|------------------|--------|
| **Migrations**    | Default: entrypoint runs migrate once then server. Or use `POST /v1/migrate` (set `MIGRATE_TOKEN`) or a Job with `./sessiondb migrate`. See [Run migrations once](./features/run-migrations-once.md). |
| **Port**          | Use `PORT` (set by Cloud Run / App Engine). |
| **Database**      | Use [managed PostgreSQL](./features/managed-database.md) (e.g. Cloud SQL). |
