# Sentinel Scanner

Sentinel Scanner is an authorized web application and API security assessment platform. It only assesses systems the operator owns or has permission to test.

This repository is the monorepo skeleton. Scan execution, ZAP, and Nuclei are not implemented yet. An operator can register, sign in, create a project, and register an authorized target. Scan jobs refer to one of those targets and remain pending until job coordination is added.

## Layout

| Path | Responsibility |
| --- | --- |
| `apps/api` | Go API. Owns accounts, projects, targets, scan lifecycle, findings, and PostgreSQL. |
| `services/scanner` | Python worker. Will run assessments. It has no database credentials. |
| `configs` | Non-secret contracts shared by the API and the worker. |
| `infrastructure` | Local Nginx and the Redis startup script. |
| `docs` | Architecture, service boundaries, and security constraints. |
| `apps/web` | React console for sign-in, projects, targets, and scan jobs. It talks only to the API. |

The API and the worker coordinate through Redis. Queue names are in `configs/queues.yaml`. Nothing publishes or consumes those queues yet.

## Local infrastructure

Requirements: Docker Compose v2.

```bash
cp .env.example .env
# Replace POSTGRES_PASSWORD, REDIS_PASSWORD, and API_AUTH_SECRET.
# Each database password must be at least 16 characters and use only letters,
# numbers, underscores, and hyphens. API_AUTH_SECRET must be at least 32
# characters and must not be the placeholder.
docker compose up --build
```

Nginx listens on `127.0.0.1:8080` by default (`API_HOST_PORT` in `.env`). It serves the console and proxies `/api/` to the Go API. PostgreSQL, Redis, the API, the scanner, and the console container stay on an internal network with no route to the internet or the host. Only Nginx joins a second network so the host can reach the proxy.

Open `http://127.0.0.1:8080/` for the console. During UI development, from `apps/web`:

```bash
npm install
npm run dev
```

Use npm 11. npm 10 fails while building this dependency tree. The dev server listens on `127.0.0.1:5173` and proxies `/api` to `API_HOST_PORT`.

- `GET http://127.0.0.1:8080/api/v1/health` — API can reach PostgreSQL and Redis.
- `GET http://127.0.0.1:8080/nginx-health` — reverse proxy is up.
- `POST http://127.0.0.1:8080/api/v1/auth/register` and `POST .../auth/login` — create or open a session. The body is `email` and `password`. The response carries a short-lived bearer token. The refresh token is an HttpOnly cookie.
- `GET` and `POST http://127.0.0.1:8080/api/v1/projects` — list and create projects for the signed-in operator. A create request needs `name` and may include `description`.
- `POST http://127.0.0.1:8080/api/v1/projects/{id}/targets` — register an authorized target. The body needs `url` and `"authorized": true`.
- `GET` and `POST http://127.0.0.1:8080/api/v1/scan-jobs` — list and create scan jobs for a project the caller belongs to. A create request needs `name`, `target_id`, and `Authorization: Bearer`.
- `GET`, `POST .../cancel`, and `DELETE http://127.0.0.1:8080/api/v1/scan-jobs/{id}` — view, cancel, or delete one of that operator's jobs.

List accepts `status`, `target_id`, `page`, and `page_size`. A created job remains pending.

Compose applies database migrations before the API starts. To run them directly from `apps/api` against a reachable database:

```bash
export API_DATABASE_URL='postgres://sentinel:password@localhost:5432/sentinel?sslmode=disable'
go run ./cmd/migrate up
```

The API uses `sslmode=disable` on the private Compose network. Production database connections must use TLS. Do not reuse this Compose file or these passwords for production.

## Tests

Go tests and the API build run from `apps/api`:

```bash
go test ./...
go run ./cmd/api
```

`go run ./cmd/api` needs `API_ENVIRONMENT`, `API_HTTP_ADDR`, `API_DATABASE_URL`, `API_REDIS_URL`, and `API_AUTH_SECRET`.

Python tests run from `services/scanner`:

```bash
python3 -m venv .venv
.venv/bin/pip install -e '.[dev]'
.venv/bin/pytest
```

Frontend tests run from `apps/web`:

```bash
npm test
```
