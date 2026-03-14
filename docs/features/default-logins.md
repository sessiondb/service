# Default Platform Logins (TOML)

## Purpose

Allow predefined platform users (e.g. admin, guest) to be created from a TOML config file so they can sign in to SessionDB without manual user creation. Seeding runs **once** when no users exist.

## Configuration

### TOML file

- **Preferred:** Use the single **config.toml** created by `scli init`; it includes `[auth]` with `[[auth.default_logins]]`. See [config.toml](config-toml.md).
- **Legacy / override:** `config.default.toml` at the repo root, or set `CONFIG_TOML_PATH` to a TOML path.
- **Structure:**
  - `[server]` — optional defaults for port, mode (env still overrides).
  - `[auth]` with `[[auth.default_logins]]` — list of default platform logins.

Example:

```toml
[server]
port = "8080"
mode = "debug"

[auth]
[[auth.default_logins]]
email = "admin@example.com"
password = "admin123"
role_key = "super_admin"

[[auth.default_logins]]
email = "guest@example.com"
password = "guest123"
role_key = "analyst"
```

- **`role_key`** must match an existing role’s `key` (e.g. `super_admin`, `developer`, `analyst`). Roles are seeded in DB migrations; if a role_key is missing, that default login is skipped and a log message is written.

### Precedence

- Server port/mode: **env vars** (and .env) override TOML; TOML is not used for server when env is set.
- Default logins: **only** from TOML (when the file exists and `[auth]` has `default_logins`). If the file is missing or unreadable, `DefaultLogins` is left nil and no seeding runs.

## When seeding runs

- **When:** On app startup, after DB connect and migrations, and after user/role repositories are created.
- **Condition:** Seeding runs **only if** there are **no** platform users (`users` table count is 0). If at least one user exists, seeding is skipped.
- **Action:** For each entry in `DefaultLogins`, if a user with that email does not exist, the app creates a user (hashed password, role from `role_key`, name derived from email) and continues. Existing users are never updated or deleted.
- **Skip in production:** Set env `SKIP_DEFAULT_LOGINS=true` (or `1` or `yes`, case-insensitive) to disable seeding entirely.

## Security notes

- Passwords in the TOML file are plaintext. Restrict file permissions and do not commit real passwords to version control.
- For production, prefer creating users via the UI or API and use `SKIP_DEFAULT_LOGINS` if you do not want any TOML-based default logins.

## Summary

| Topic | Detail |
|-------|--------|
| **File** | `config.toml` (from scli init) or `config.default.toml` / `CONFIG_TOML_PATH` |
| **When** | Once on first run (no users in DB) |
| **Skip** | `SKIP_DEFAULT_LOGINS=true` (or `1`/`yes`) |
| **Role** | `role_key` must match a seeded role key |
