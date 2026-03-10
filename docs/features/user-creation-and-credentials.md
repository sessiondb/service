# User Creation and Credentials

## Purpose

Define how platform and database users are created, what "existing user" means, and how optional "send credentials by email" works.

## How it works

### Creating a user (admin)

1. Admin creates a **platform user** (name, role, password, permissions, etc.) via the User Management UI or `POST /v1/users`.
2. The backend creates the platform user and **auto-provisions DB users** on all instances: for each instance, if the user does not already have a DB credential for that instance, a DB user is created (username/password on the target database) and linked to the platform user. No catalog crawl is performed to "discover" an existing DB user; provisioning means **creating** a DB user when one does not exist.
3. Optional: when creating the user, the admin can check **"Send credentials to user by email"**. If checked, after the user is created the backend sends an email (when mail is configured) with login URL and temporary password. The API response includes `emailSent` (and optionally `emailError` if sending failed).

### Existing user semantics

- **Existing user** means a platform user that already exists (same email). Creating a user with an email that is already in use returns an error ("email already in use").
- **Provisioning on a new instance:** When a platform user already exists and a new instance is added, provisioning for that user on the new instance means **creating a DB user** on that instance if the user does not yet have a DB credential for it. There is no "link existing DB user" flow in the current implementation; provisioning always creates a new DB user when none exists for that user–instance pair.

### Send credentials by email

- **Configuration:** Mail is optional. When `MAIL_ENABLED` is true and SMTP (or equivalent) is configured, the mail service can send credential emails. When mail is disabled, the handler still accepts `sendCredentialsEmail: true` but does not send; `emailSent` will be false.
- **API:** `POST /v1/users` body may include `sendCredentialsEmail: true`. Response when true: `{ "data": <user>, "emailSent": true }` or `{ "data": <user>, "emailSent": false, "emailError": "..." }`. When `sendCredentialsEmail` is false, the response is the created user object only (unchanged).

## Configuration

- **Mail:** See backend config for `MAIL_*` and `APP_URL` (used as login link base). When mail is not configured, "send credentials by email" is a no-op and the UI can show that credentials were not sent.

## Limits

- Email delivery depends on SMTP (or configured provider). If mail fails, the admin should share credentials manually; the UI can show "Email could not be sent; share credentials manually."
