# ADR 0001: API and scanner boundaries

## Status

Accepted

## Decision

The Go API is a modular monolith and the only writer of application data. The Python scanner is an independent process. Redis carries explicit job and result messages between them. The scanner does not connect to PostgreSQL.

## Consequences

- A scanner compromise does not come with database credentials.
- Scan execution can be scaled or isolated without moving user, project, or finding ownership out of the API.
- Result delivery is asynchronous. The API remains responsible for what is stored and shown to users.
- Introducing another network service needs a concrete operational reason. ZAP and Nuclei will be engine integrations inside the worker's isolation boundary, not new application services, until that reason exists.
