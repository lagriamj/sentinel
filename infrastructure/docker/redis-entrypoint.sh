#!/bin/sh
# Start Redis as the unprivileged redis user with a password that never
# appears in the process arguments.
set -eu

case "${REDIS_PASSWORD:-}" in
  "" | change-me-local-only)
    echo "Replace REDIS_PASSWORD in .env before starting." >&2
    exit 1
    ;;
  *[!A-Za-z0-9_-]*)
    echo "REDIS_PASSWORD must contain only letters, numbers, underscores, and hyphens." >&2
    exit 1
    ;;
esac

if [ "${#REDIS_PASSWORD}" -lt 16 ]; then
  echo "REDIS_PASSWORD must be at least 16 characters." >&2
  exit 1
fi

umask 077
rm -f /tmp/redis.conf
cat > /tmp/redis.conf <<EOF
requirepass ${REDIS_PASSWORD}
appendonly yes
protected-mode yes
rename-command FLUSHALL ""
rename-command FLUSHDB ""
rename-command DEBUG ""
EOF

if [ "$(id -u)" = "0" ]; then
  chown redis:redis /tmp/redis.conf
fi

exec /usr/local/bin/docker-entrypoint.sh redis-server /tmp/redis.conf
