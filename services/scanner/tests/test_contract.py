from pathlib import Path

from scanner.settings import Settings


def test_queue_names_match_shared_config() -> None:
    root = Path(__file__).resolve().parents[3]
    contract = (root / "configs" / "queues.yaml").read_text(encoding="utf-8")
    settings = Settings(redis_url="redis://localhost:6379/0")
    assert f"jobs: {settings.jobs_queue}" in contract
    assert f"results: {settings.results_queue}" in contract
