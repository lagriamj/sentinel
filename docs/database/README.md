# Database

PostgreSQL is the system of record. The Go API is the only application client.

Schema changes are versioned SQL migrations in `apps/api/migrations/` using golang-migrate. Applied migration files stay immutable. The first migration enables `pgcrypto`. The API does not migrate on startup; `cmd/migrate` does.

## Tables

- `users` stores an account email and an Argon2id password hash. Passwords cannot be stored in plaintext. Registration writes this row.
- `refresh_tokens` stores the SHA-256 hash of a refresh token, its expiry, and the token that replaced it. The raw token is not stored.
- `projects` stores a named workspace. `project_members` assigns each user the role `owner` or `member`.
- `scan_targets` stores an authorized `http` or `https` target for one project. The `authorized` column must be true. The same URL may be registered on another project.
- `scan_jobs` stores one assessment run. Status is `pending`, `queued`, `running`, `completed`, `failed`, or `cancelled`. A finished status requires `finished_at`.
- `scan_results` stores one normalized engine summary per job. The scanner does not write this table.
- `vulnerabilities` stores findings for a job and target.

Report metadata is not in the schema yet.

Deleting a scan job removes its results and vulnerabilities. A target or user referenced by a job is kept. A project cannot be deleted while it still has targets. Creating a scan job does not publish to Redis.

Redis is a coordination channel, not a system of record. Queue contents are recoverable work, not the finding history.
