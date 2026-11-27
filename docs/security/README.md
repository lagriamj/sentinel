# Security constraints

Sentinel Scanner may only assess systems the operator owns or is explicitly authorized to test.

These controls are required before scan execution is added.

Target registration currently:

- Requires explicit authorization before a target is stored.
- Rejects literal loopback, private, link-local, and cloud metadata addresses.
- Resolves a hostname when DNS answers. A private, loopback, link-local, or metadata answer is rejected, as is a name that does not exist.
- A hostname is stored when the resolver cannot be reached. Scan execution must resolve it again before any request.
- Refuses target URLs that contain credentials, a query, or a fragment.
- Checks the stored target again when a scan job is created.

Accounts and access:

- Passwords are hashed with Argon2id. The users table rejects any other hash prefix.
- Refresh tokens are random, stored as SHA-256 hashes, and rotated when presented. A reused refresh token revokes that account's refresh tokens.
- Project, target, and scan-job routes require a bearer token. A caller who is not a project member receives 404. A scan job can only use a target registered on that project.

Still required before a scanner runs:

- Revalidate DNS answers and every redirect while the job is executing.
- Restrict every request to the stored host and port.
- Run ZAP and Nuclei in isolated containers without the Docker socket, with dropped capabilities and CPU, memory, and egress limits.
- Keep scanner logs free of credentials and raw secrets.

PostgreSQL, Redis, the API, and the scanner share an internal Compose network with no route to the internet or the host. Nginx is the only published service, bound to `127.0.0.1` and `API_HOST_PORT` (8080 by default). Redis requires a password and refuses the placeholder from `.env.example`.
