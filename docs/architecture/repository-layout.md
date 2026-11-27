# Repository layout

```text
apps/api/                 Go API
  cmd/api/                Process entrypoint
  internal/platform/      Config, HTTP, PostgreSQL, and Redis adapters
  internal/modules/       health, auth, project, and scanjob
  migrations/             golang-migrate SQL migrations
apps/web/                 React console
  src/features/           Sign-in, project, and scan job screens
  src/components/ui/      shadcn controls
  src/services/           API client
services/scanner/         Python worker
  scanner/core/           Shared worker code
  scanner/jobs/           Future job execution
  scanner/engines/        Future ZAP and Nuclei adapters
  scanner/scope/          Future target authorization checks
  scanner/normalization/  Future common finding schema
  scanner/reporting/      Future result preparation
  tests/
configs/                  Non-secret shared contracts
infrastructure/nginx/     Local reverse proxy
docs/                     Architecture, decisions, and constraints
docker-compose.yml        Local PostgreSQL, Redis, API, worker, web, and Nginx
```

`health`, `auth`, `project`, and `scanjob` live under `apps/api/internal/modules/`. Finding APIs will be added there when they are implemented. HTTP handlers stay thin and call those modules. Platform packages know how to reach PostgreSQL and Redis. They do not own assessment rules. Target URL rules live in `apps/api/internal/scope`.

The worker packages are boundaries only. Engine packages must not grow shell commands or scanner clients until execution is an explicit task.
