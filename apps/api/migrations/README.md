# Migrations

SQL migrations for the API database live in this directory and run through golang-migrate.

Do not edit a migration after it has been applied. Add a new version instead. The API does not migrate on startup. `cmd/migrate` applies this directory, and Compose runs `/app/migrate up` before starting the API.

Each file may contain multiple statements. They are split on semicolons. Do not put PL/pgSQL function bodies in a migration, because the splitter does not understand dollar quotes.

- `000001_enable_pgcrypto` enables UUID generation.
- `000002_create_assessment_schema` creates users, scan targets, scan jobs, scan results, and vulnerabilities.
