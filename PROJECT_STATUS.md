# Sentinel Scanner — Project Status

Living progress tracker. Update it from the repository, not from intent. A directory or placeholder file is not a completed feature.

Last inspected: 2026-11-27.

## A. Project Overview

Sentinel Scanner is an authorized web application and API security assessment platform. It registers systems the operator owns or is allowed to test, records assessment jobs, and is intended to run OWASP ZAP, Nuclei, and later checks from an isolated worker.

| Item | Current state |
| --- | --- |
| Stack | Go 1.26 API (Gin, GORM, golang-migrate), Python 3.12 worker (Pydantic, Redis client), PostgreSQL 17, Redis 8, Docker Compose, Nginx, React 19 console (TypeScript, Vite, Tailwind, TanStack Query). |
| Phase | Authentication, projects, registered targets, and scan-job records exist. Execution, findings, and reporting do not. |
| Progress | The local platform can boot, migrate, register an operator, and record that operator's scan jobs against a registered target. It cannot assess a target. This is not production-ready. |

The repository is not a Git checkout, so history below comes from the files and from verification runs, not from commits.

## B. Repository Structure

```text
apps/api/                 Go API. Owns PostgreSQL. Modules: health, auth, project, scanjob.
  cmd/api/                API process.
  cmd/migrate/            Migration process. The API does not migrate on startup.
  internal/modules/       health, auth, project, and scanjob (handler, service, repository, model).
  internal/platform/      Config, HTTP server, auth middleware, logging, Postgres, Redis, migrate.
  migrations/             golang-migrate SQL. 000001 pgcrypto, 000002 assessment tables, 000003 refresh tokens, 000004 projects.
services/scanner/         Python worker. Process, settings, Redis heartbeat, health check.
  scanner/jobs/           Placeholder. No dequeue or execution.
  scanner/engines/        Placeholder packages for ZAP and Nuclei.
  scanner/scope/          Placeholder. No scope enforcement.
  scanner/normalization/  Placeholder. No finding schema.
  scanner/reporting/      Placeholder. No result publishing.
  tests/                  Pytest for settings, queue names, and the idle worker.
apps/web/                 React console. Sign-in, projects, targets, and scan-job list, create, detail, cancel, and delete.
configs/queues.yaml       Queue names: sentinel:scan-jobs, sentinel:scan-results.
infrastructure/nginx/     Loopback reverse proxy.
infrastructure/docker/    Redis startup script.
docs/                     Architecture, database, security, ADR 0001.
docker-compose.yml        Postgres, Redis, migrate, API, scanner, web, Nginx.
```

## C. Development Progress

### Phase 1 — Foundation

The runnable local stack exists: API, worker, Postgres, Redis, migrations, Nginx, and a React console for scan jobs.

### Phase 2 — Authentication and Authorization

Operators can register, sign in, refresh a session, and sign out. Scan-job routes require that caller. New jobs record `created_by`. Passwords are Argon2id. Refresh tokens are stored as SHA-256 hashes and rotate on use.

### Phase 3 — Projects and Targets

An operator can create a project, invite an existing account as owner or member, and register an authorized target. A scan job must refer to one of those targets. Another operator receives 404 for the project, its targets, and its jobs.

### Phase 4 — Scanning Engine

Scan-job records can be created, listed, read, cancelled, and deleted. Status values `pending`, `queued`, `running`, `completed`, `failed`, and `cancelled` are defined. New jobs stay `pending`. Neither process reads or writes the Redis lists. ZAP, Nuclei, HTTP checks, and normalization are package docstrings only.

### Phase 5 — Vulnerability Management

`scan_results` and `vulnerabilities` exist as tables. Deleting a job removes those rows. There is no findings API, dashboard, evidence store, status workflow, or remediation text.

### Phase 6 — Reporting and Deployment

No PDF reports, analytics, or production deployment. Local Compose is the only runtime. A future VPS setup is described as separate from this Compose file and is not present.

## D. Task Status

- [x] Completed — implemented, with the verification noted on that line.
- [~] In progress — partial. Not production-ready.
- [ ] Pending — not implemented.
- [!] Blocked — waiting on a dependency.

### Phase 1 — Foundation

- [x] Repository layout and engineering rules. `README.md`, `agents.md`, `.cursor/rules/`, `docs/`.
- [x] Go API process, config, structured logging, and versioned `/api/v1` routes. `apps/api/cmd/api/main.go`, `apps/api/internal/platform/`.
- [x] React frontend for projects, targets, and scan jobs. `apps/web`. Standard controls use Shadcn UI. The shell, empty and loading states, and status colors stay specific to this console. `npm test` (20 tests) passed on 2026-11-27. The signed-in flow was exercised in the browser: create a project, register `https://example.com`, create a job from that target, delete the job, and delete the project after its target was removed. Deleting the project while the target remained returned a conflict. Verification accounts, jobs, and targets were removed.
- [~] Python scanner process. It loads settings, pings Redis, and exits healthy. `services/scanner/scanner/worker.py`, `scanner/healthcheck.py`. It does not scan.
- [x] Docker Compose for Postgres, Redis, migrate, API, scanner, web, and Nginx. `docker-compose.yml`. Internal network, resource limits, non-root API and scanner images. On 2026-11-27 the `web` image built, and the web, API, and Nginx containers were healthy. `http://127.0.0.1:18080/` returned the console. `/api/v1/health` returned ok and the job list was empty.
- [x] PostgreSQL connection pool and ping. `apps/api/internal/platform/postgres/`.
- [x] Redis client used for ping only. `apps/api/internal/platform/queue/queue.go`. No publish or consume methods.
- [x] Environment template. `.env.example`. Secrets stay in untracked `.env`. Placeholder Redis passwords are rejected by the Redis entrypoint.
- [x] Health checks. `GET /api/v1/health` pings Postgres and Redis. `GET /nginx-health` returns `ok`. Scanner health check pings Redis.
- [x] Database migrations. `apps/api/migrations/000001_enable_pgcrypto.*.sql`, `000002_create_assessment_schema.*.sql`, `000003_create_refresh_tokens.*.sql`, `000004_create_projects.*.sql`. `cmd/migrate` applies them. Compose runs migrate before the API. On 2026-11-27, migration version 4 was applied and was not dirty.
- [~] Initial tests. `go test ./...` passed on 2026-11-27. That run did not set `TEST_DATABASE_URL`, so `apps/api/internal/modules/scanjob/repository/postgres_test.go` skipped its database tests. Those integration tests passed on 2026-11-20 against a temporary Postgres. Python tests were not executed on 2026-11-27: `pytest` is not installed in the current environment. The test files were read.

### Phase 2 — Authentication and Authorization

- [x] Registration. `POST /api/v1/auth/register`. Duplicate email returns 409. Password must be 12–128 characters. Verified through the console on 2026-11-27.
- [x] Login and logout. `POST /api/v1/auth/login` and `POST /api/v1/auth/logout`. A wrong password returns 401 with `invalid email or password`. Logout clears the refresh cookie. Register and login are limited to 10 requests per minute per IP.
- [x] JWT access tokens. HS256, 15-minute default lifetime, `github.com/golang-jwt/jwt/v5`. The console keeps the access token in memory and sends `Authorization: Bearer`.
- [x] Refresh-token rotation. `refresh_tokens` stores a 64-character SHA-256 hex hash. `POST /api/v1/auth/refresh` rotates it. The raw token is an HttpOnly `sentinel_refresh` cookie, path `/api/v1/auth`, SameSite Lax. A reload of the console restored the session on 2026-11-27. Reuse detection is covered by `apps/api/internal/modules/auth/service/service_test.go`. That test uses an in-memory store. It was not run against PostgreSQL.
- [x] Authentication middleware. `apps/api/internal/platform/authn`. Health stays public. `GET /api/v1/scan-jobs` without a token returned 401 on 2026-11-27.
- [x] Authorization policies. Scan-job list, get, cancel, and delete are limited to `created_by`. Another operator's job returned 404 on 2026-11-27. New jobs set `created_by`.
- [x] Users table. `users` requires an email and an Argon2id hash prefix. Registration writes it. Constraint covered by `repository/contract_test.go` and, when a database is configured, `TestPostgresSchemaConstraints`.

### Phase 3 — Projects and Targets

- [x] Project CRUD. `POST`, `GET`, `PATCH`, and `DELETE /api/v1/projects`. The creator is the owner. Delete is rejected while the project still has targets. Verified through the console on 2026-11-27.
- [x] Project membership and roles. `owner` and `member`. An owner can add an existing account by email and remove a member. The last owner cannot be removed. A caller who is not a member receives 404.
- [x] Target registration API. `POST`, `GET`, and `DELETE /api/v1/projects/{id}/targets`. A target belongs to one project. The same URL can be registered on another project. A target used by a job cannot be deleted. Scan jobs no longer insert targets.
- [x] URL validation at registration and again when a job is created. `apps/api/internal/scope`. HTTP(S) only. Credentials, query strings, and fragments are rejected. Covered by `internal/scope/target_test.go`.
- [x] Authorization confirmation. Target registration requires `"authorized": true`. The database check `scan_targets_authorized` rejects a false value. The console asks for that confirmation before calling the API.
- [~] Target scope validation at registration and job creation. Literal loopback, private, link-local, metadata, `.local`, `.internal`, and ambiguous numeric hosts are rejected. A hostname that DNS says is missing, or that resolves to a disallowed address, is rejected when the resolver answers. If the resolver cannot be reached, the hostname is stored. Compose has no egress, so public names are stored without an address check. Execution-time redirect checks do not exist.

### Phase 4 — Scanning Engine

- [~] Scan job lifecycle. Implemented and unit-tested: create `201` from a registered `target_id`, list with `status`, `target_id`, `page`, and `page_size`, get, cancel, delete. A project member can see the project's jobs. Routes in `apps/api/internal/modules/scanjob/handler/handler.go`. Cancel allows `pending`, `queued`, and `running`. Delete allows `pending`, `completed`, `failed`, and `cancelled`. Database integration coverage exists and was verified on 2026-11-20, before projects. Those tests were not re-run against PostgreSQL on 2026-11-27. No code moves a job to `queued`, `running`, `completed`, or `failed`.
- [ ] Go and Python job communication. Queue names match in `configs/queues.yaml`, API config defaults, and scanner settings. `services/scanner/tests/test_contract.py` asserts that match. Nothing enqueues or dequeues. `worker.py` tests assert `blpop` and `brpop` are unused. Not executed on 2026-11-27.
- [ ] HTTP security checks.
- [ ] OWASP ZAP. `services/scanner/scanner/engines/zap/__init__.py` is a one-line placeholder. ZAP is not in Compose.
- [ ] Nuclei. `services/scanner/scanner/engines/nuclei/__init__.py` is a placeholder. Nuclei is not in Compose.
- [ ] Finding normalization. `scanner/normalization/` and `scanner/reporting/` are placeholders.

### Phase 5 — Vulnerability Management

- [ ] Findings dashboard.
- [~] Severity values in the schema only: `info`, `low`, `medium`, `high`, `critical`. No API.
- [ ] Evidence management. `vulnerabilities.location` and `description` exist. No upload or evidence API.
- [ ] Finding status tracking.
- [ ] Remediation recommendations.

### Phase 6 — Reporting and Deployment

- [ ] PDF reports.
- [ ] Scan history beyond the current job list.
- [ ] Analytics.
- [ ] VPS deployment. Documented as future work in `infrastructure/README.md`. No production compose or host config.
- [ ] Production hardening. Local defaults include `sslmode=disable` on the private network. No TLS termination for a public host. Register and login are rate limited. Other routes are not.

## E. Technical Decisions

- Go API is a modular monolith. Handlers call services. Services own rules. Repositories own SQL. `docs/decisions/0001-service-boundaries.md`.
- The API is the only PostgreSQL client. Workers must not receive database credentials.
- Python worker is a separate deployable. Engine packages must stay empty until execution is an explicit task.
- The frontend talks only to the API. It does not talk to Redis, Postgres, or the worker.
- Schema changes are versioned SQL in `apps/api/migrations/`. Applied files stay immutable. The runner splits files on semicolons and does not understand PL/pgSQL dollar quotes, so migrations stay free of function bodies. `apps/api/internal/platform/migrate/migrate.go`.
- Primary keys are UUIDs from `gen_random_uuid()`.
- Job coordination is two Redis lists: `sentinel:scan-jobs` and `sentinel:scan-results`. They stay unread until a consumer exists, so a published job is not dropped.
- Creating a job writes PostgreSQL and does not publish.
- Docker Compose is local only. Nginx on `127.0.0.1` is the published port. Postgres and Redis are on an internal network. Redis requires a password. Scanner containers do not mount the Docker socket.
- Assessment requires an explicit authorization flag. Stored targets must be authorized. Passwords are Argon2id hashes. Refresh tokens are stored as hashes.

## F. Known Issues and Technical Debt

- Auth repository behavior was not exercised by `go test ./...` on 2026-11-27. That command had no `TEST_DATABASE_URL`, so database tests were skipped. Refresh rotation was unit-tested in memory and the live register, login, refresh, and logout path was used through Nginx.
- Hostname registration fails open when DNS is unreachable. That is the normal Compose case. A name can be stored and later resolve to a private address. Execution must re-check before any request. That check is not implemented.
- `queued`, `running`, `completed`, and `failed` can be stored by direct SQL. The API never enters those states. Cancel and delete rules for them are tested by updating rows in the integration test.
- Results and vulnerabilities have foreign keys and cascade deletes, and no writer other than tests.
- Project and target repository SQL was used through Nginx on 2026-11-27. `go test ./...` did not set `TEST_DATABASE_URL`, so that SQL was not covered by the database integration tests.
- Rate limiting covers register and login only.
- Migration integration is not part of a plain `go test ./...`. It runs only when `TEST_DATABASE_URL` is set.
- Python tests were not run on 2026-11-27. Pytest is not installed here. Their behavior is unverified in this inspection.
- No Git repository was found, so there is no commit history to audit.
- `httpserver` tests use stub stores that return `not implemented` for unused scan-job methods. That is test scaffolding, not a product path.
- ZAP, Nuclei, and report generation are named in rules and docs and have no implementation. The React console covers sign-in, projects, targets, and scan-job records.

## G. Next Development Tasks

1. Publish a pending job to Redis and consume it without running an engine.
   - Outcome: the API moves a job from `pending` to `queued` only when the worker can receive it. The worker acknowledges the job and can observe cancellation. No ZAP or Nuclei yet.
   - Depends on registered targets. Keep the lists unread until both sides exist.
   - Files: `apps/api/internal/platform/queue/queue.go`, `services/scanner/scanner/worker.py`, `services/scanner/scanner/jobs/`, `configs/queues.yaml`.

2. Scope enforcement on the worker, then one engine.
   - Outcome: the worker resolves DNS, rejects private and metadata addresses, and re-checks redirects before any request. The first engine can be a small HTTP check. ZAP and Nuclei come after that path is proven.
   - Depends on task 1.
   - Files: `services/scanner/scanner/scope/`, `services/scanner/scanner/engines/`.

3. Persist normalized results.
   - Outcome: the worker publishes one result shape. The API writes `scan_results` and `vulnerabilities` and moves the job to `completed` or `failed`.
   - Depends on task 2.
   - Files: `services/scanner/scanner/normalization/`, `services/scanner/scanner/reporting/`, a new API consumer for `sentinel:scan-results`.

4. Extend the React console after findings exist.
   - Outcome: `apps/web` can manage findings through the API. Sign-in, projects, targets, and scan jobs already exist.
   - Depends on task 3.
   - Files: `apps/web/src/`.

## H. Change History

- 2026-11-27 — Added projects, owner and member roles, and registered targets. Scan jobs now require a target that was registered first. Migration version 4 was applied. The console can manage projects and targets. Verification rows were removed.
- 2026-11-27 — Added operator registration, login, logout, short-lived access tokens, and rotating hashed refresh tokens. Scan-job routes require the caller and new jobs set `created_by`. The console can sign in. Migration version 3 was applied. Verification accounts and rows were removed.
- 2026-11-27 — Switched the console’s buttons, inputs, checkbox, select, alert, table, card, and badge to Shadcn UI. The application shell, empty and loading states, and status colors remain custom.
- 2026-11-27 — Added this tracker after inspecting the tree, rules, migrations, API, worker, Compose file, and tests. Re-ran `go test ./...` without a database. Python tests were not run.
- 2026-11-20 — Added assessment tables and the scan-job module. Migration version 2 was applied to the Compose database. Repository integration tests passed with `TEST_DATABASE_URL` set. A live create, list, cancel, and delete passed through Nginx, and the verification rows were removed. Authentication was not added.
- Earlier — Foundation already present before the scan-job module: API skeleton, health checks, Postgres and Redis adapters, idle Python worker, Compose, Nginx, and queue-name contract. No commit dates are available.
