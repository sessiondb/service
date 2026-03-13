# Binders and Versioned Releases

## Purpose

Provide versioned binaries and assets (server, UI, scripts, config) so users can run SessionDB locally or on a VM without Docker. Binaries are stored in the [sessiondb/.github](https://github.com/sessiondb/.github) repo under **releases/<version>/binaries/**.

## Where releases live

- **Repo:** [sessiondb/.github](https://github.com/sessiondb/.github)
- **Path:** `releases/<version>/binaries/` (e.g. `releases/1.0.1/binaries/`)
- **Contents:** `server/`, `ui/`, `README.md`, `setup.sh`, `sessiondb.yaml`

## What you get

Inside `releases/<version>/binaries/`:

- **server/** — backend binary and run script
- **ui/** — frontend assets and serve script
- **setup.sh** — starts server and UI (env is injected by CLI from sessiondb.yaml)
- **sessiondb.yaml** — runtime config: server/UI ports and paths, and **env** (all backend and UI env vars). No .env file; the CLI injects these values into server and UI when you run `sessiondb run <version>`.
- **README.md** — version and usage guidelines

## Config: sessiondb.yaml (no .env)

All environment variables for the server and UI live in **sessiondb.yaml** under the `env:` section. When you run **`sessiondb run <version>`**, the CLI reads sessiondb.yaml and injects those env vars into the server and UI processes. You do not need a .env file.

Edit `sessiondb.yaml` (e.g. set `DB_PASSWORD`, `JWT_SECRET`, `VITE_API_URL`) and run `sessiondb run 1.0.1` again; the new values are used.

## How to get a release

1. **From the repo:** Clone or download [sessiondb/.github](https://github.com/sessiondb/.github) and use `releases/<version>/binaries/`. Edit `sessiondb.yaml` with your env, then run via CLI: `sessiondb run <version>` from a directory that contains (or will contain) the extracted `sessiondb/` folder.

2. **CLI:** The SessionDB CLI is in a separate repo ([scli](https://github.com/sessiondb/scli)). Build with `go build -o sessiondb .` then run `sessiondb get 1.0.1` (downloads to `./sessiondb/`), edit `sessiondb/sessiondb.yaml` if needed, and run `sessiondb run 1.0.1` so the CLI injects env from sessiondb.yaml into server and UI.

## How releases are produced

The workflow in **sessiondb/.github** runs on a version tag (e.g. `1.0.1`): it builds the backend and frontend at that tag, assembles the contents, and pushes them to `releases/<version>/binaries/` in the same repo. Ensure backend and frontend repos have the same version tag before tagging sessiondb/.github.

## Limits

- Server binary is linux/amd64. Other platforms require building from source.
- First run requires Node (for `npx serve`) or Python to serve the UI; server is a single binary.
