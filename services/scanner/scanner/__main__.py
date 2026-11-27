"""Process entrypoint. Connects to Redis and waits for a later execution milestone."""

import signal
import threading

from pydantic import ValidationError

from scanner.settings import Settings
from scanner.worker import Worker, connect, log_event


def main() -> None:
    try:
        settings = Settings()
    except ValidationError:
        log_event("configuration_invalid")
        raise SystemExit(1) from None

    stop = threading.Event()

    def request_stop(_signum: int, _frame: object) -> None:
        stop.set()

    signal.signal(signal.SIGTERM, request_stop)
    signal.signal(signal.SIGINT, request_stop)

    client = connect(settings)
    try:
        Worker(settings, client).serve(stop)
    finally:
        client.close()


if __name__ == "__main__":
    main()
