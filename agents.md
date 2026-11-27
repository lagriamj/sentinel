
# Sentinel Scanner - Engineering Guidelines

## Project Overview
Sentinel Scanner is an authorized web application
and API security assessment platform.

## Architecture
- Go + Gin for the main backend.
- Python for scanning workers.
- React + TypeScript for the frontend.
- PostgreSQL for persistent storage.
- Redis for background job coordination.
- Docker Compose for local infrastructure.

## Backend Guidelines
- Follow idiomatic Go conventions.
- Use modular architecture.
- Keep handlers thin.
- Separate business logic from database operations.
- Use context.Context for cancellation and timeouts.
- Use dependency injection where appropriate.
- Use structured logging.
- Validate all incoming requests.
- Use database migrations.
- Write unit and integration tests.

## Python Guidelines
- Use Python type hints.
- Follow PEP 8.
- Use async where appropriate.
- Separate scanning engines from orchestration.
- Normalize scanner output.
- Handle timeouts and failures.
- Never trust scanner input.

## Frontend Guidelines
- React with TypeScript.
- Use reusable components.
- Keep API communication in service modules.
- Use consistent error handling.
- Implement loading, empty, and error states.

## Security Guidelines
- Only scan explicitly authorized targets.
- Enforce target allowlists.
- Block prohibited network destinations.
- Revalidate DNS and redirects.
- Never expose secrets in logs.
- Run scanner processes in isolation.
- Apply resource and execution limits.

## Testing
- Every business-critical feature must have tests.
- Validate authorization boundaries.
- Test invalid and malicious input.
- Do not consider code complete without tests.

## Development Rules
- Do not generate the entire application at once.
- Work on one module at a time.
- Explain architectural decisions.
- Avoid unnecessary abstractions.
- Keep documentation updated.
