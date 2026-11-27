"""Redis heartbeat loop.

This module must not dequeue jobs or start a scanning engine. Consuming a job
before execution exists would drop authorized work.
"""

import json
import threading
from collections.abc import Callable

from redis import Redis

from scanner.settings import Settings


def log_event(event: str, **fields: object) -> None:
    """Write one JSON log record. Callers must not pass secrets."""
    print(json.dumps({"event": event, **fields}), flush=True)


def connect(settings: Settings) -> Redis:
    """Open a Redis client from the worker settings."""
    return Redis.from_url(
        str(settings.redis_url),
        socket_connect_timeout=5,
        socket_timeout=5,
        decode_responses=True,
    )


class Worker:
    """Keeps the worker process alive and confirms Redis is reachable."""

    def __init__(
        self,
        settings: Settings,
        client: Redis,
        wait: Callable[[float], None] | None = None,
    ) -> None:
        self._settings = settings
        self._client = client
        self._wait = wait

    def serve(self, stop: threading.Event) -> None:
        """Ping Redis until stop is set. Does not read queue contents."""
        self._ping()
        log_event(
            "scanner_ready",
            jobs_queue=self._settings.jobs_queue,
            results_queue=self._settings.results_queue,
        )
        while not stop.is_set():
            try:
                self._ping()
            except RuntimeError:
                log_event("redis_unreachable")
            if self._wait is not None:
                self._wait(float(self._settings.heartbeat_seconds))
            else:
                stop.wait(self._settings.heartbeat_seconds)
        log_event("scanner_stopped")

    def _ping(self) -> None:
        if not self._client.ping():
            raise RuntimeError("redis ping failed")
