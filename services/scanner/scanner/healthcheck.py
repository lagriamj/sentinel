"""Container health check. Exits 0 when Redis answers."""

from scanner.settings import Settings
from scanner.worker import connect, log_event


def main() -> int:
    try:
        client = connect(Settings())
        try:
            if not client.ping():
                raise RuntimeError("redis ping failed")
        finally:
            client.close()
    except Exception:
        log_event("redis_ping_failed")
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
