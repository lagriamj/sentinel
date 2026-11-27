# Infrastructure

Local infrastructure is defined in the root `docker-compose.yml`.

- `infrastructure/nginx/` is the reverse proxy in front of the API. It is the only service published to the host, and it binds to loopback.
- `infrastructure/docker/redis-entrypoint.sh` writes a Redis config at startup and hands off to the image entrypoint, which drops to the `redis` user. The password is not passed as a process argument.
- PostgreSQL and Redis stay on the internal Compose network and are not published to the host.
- Service Dockerfiles live next to the service they build: `apps/api/Dockerfile` and `services/scanner/Dockerfile`.

OWASP ZAP and Nuclei are not part of this Compose file. They will run in isolated containers when scan execution is implemented.

Production configuration for the VPS will live separately from this local stack. Do not put production credentials in this directory.
