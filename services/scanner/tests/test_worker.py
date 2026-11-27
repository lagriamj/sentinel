import inspect
import threading
from pathlib import Path
from unittest.mock import MagicMock

from scanner.settings import Settings
from scanner.worker import Worker
import scanner.worker as worker_module


def test_serve_pings_without_reading_a_queue() -> None:
    client = MagicMock()
    client.ping.return_value = True
    stop = threading.Event()

    def stop_after_wait(_seconds: float) -> None:
        stop.set()

    settings = Settings(redis_url="redis://localhost:6379/0", heartbeat_seconds=1)
    Worker(settings, client, wait=stop_after_wait).serve(stop)

    assert client.ping.call_count >= 1
    assert client.blpop.call_count == 0
    assert client.brpop.call_count == 0


def test_worker_does_not_invoke_scanners() -> None:
    source = Path(inspect.getfile(worker_module)).read_text(encoding="utf-8").lower()
    for term in ("subprocess", "nuclei", "zap", "blpop", "brpop", "psycopg", "sqlalchemy"):
        assert term not in source
