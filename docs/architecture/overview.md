# Architecture

Sentinel Scanner assesses only systems the operator owns or is explicitly authorized to test.

The platform is a Go API, a separately deployable Python worker, and a React console under `apps/web`. An operator can register, sign in, and manage projects and authorized targets. Scan jobs refer to a registered target. Findings and scan execution are not built yet.

## Responsibilities

### Go API (`apps/api`)

The API owns durable state and access control.

- Authentication and authorization.
- Users, projects, and registered targets.
- Scan lifecycle: create, cancel, and record status.
- Job coordination: the API will publish explicit instructions to Redis.
- Vulnerability records and reporting APIs.

The API is the only application process that connects to PostgreSQL.

### Python scanner (`services/scanner`)

The worker owns execution against an authorized target.

- Receive an explicit job from Redis.
- Enforce target scope before any outbound request.
- Run OWASP ZAP, Nuclei, and future checks.
- Normalize engine output into one finding shape.
- Publish results back to Redis for the API to store.

The worker does not receive database credentials and must not open a PostgreSQL connection. Scan execution is not implemented yet. The process starts, checks Redis, and waits.

### React UI (`apps/web`)

The console signs an operator in, then manages that operator's projects, registered targets, and scan jobs through the API. The access token stays in memory. The refresh token is an HttpOnly cookie. Saved jobs stay pending. Standard controls use shadcn. The application shell, empty and loading states, and status colors stay specific to this console. It does not talk to the scanner, Redis, or the database. Findings are not in the UI yet.

## Communication

```text
Browser  --HTTP-->  Nginx  --HTTP-->  React console
                           --HTTP-->  Go API  --SQL-->  PostgreSQL
                                      Go API  --Redis-->  Python scanner
```

Redis list names live in `configs/queues.yaml`:

- `sentinel:scan-jobs` for work the API will publish.
- `sentinel:scan-results` for results the worker will return.

Nothing reads or writes those lists yet. Leaving them unread avoids dropping jobs that no worker can complete. The scan job API stores a `pending` row in PostgreSQL and does not publish it.
