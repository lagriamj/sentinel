import pytest
from pydantic import ValidationError

from scanner.settings import Settings


def test_redis_url_is_required(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.delenv("SCANNER_REDIS_URL", raising=False)
    with pytest.raises(ValidationError):
        Settings()


def test_database_url_is_not_a_setting() -> None:
    assert "database_url" not in Settings.model_fields


def test_postgres_url_is_rejected() -> None:
    with pytest.raises(ValidationError):
        Settings(redis_url="postgres://localhost/sentinel")


def test_queue_defaults_match_shared_contract() -> None:
    settings = Settings(redis_url="redis://localhost:6379/0")
    assert settings.jobs_queue == "sentinel:scan-jobs"
    assert settings.results_queue == "sentinel:scan-results"
