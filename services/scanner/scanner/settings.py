"""Worker settings. Database credentials are intentionally not accepted."""

from typing import Literal

from pydantic import Field, RedisDsn, field_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Runtime settings loaded from SCANNER_* environment variables."""

    model_config = SettingsConfigDict(
        env_prefix="SCANNER_",
        extra="ignore",
        env_file=None,
    )

    environment: Literal["development", "test", "production"] = "development"
    redis_url: RedisDsn
    jobs_queue: str = "sentinel:scan-jobs"
    results_queue: str = "sentinel:scan-results"
    heartbeat_seconds: int = Field(default=5, ge=1, le=60)

    @field_validator("jobs_queue", "results_queue")
    @classmethod
    def queue_name_has_no_whitespace(cls, value: str) -> str:
        if not value or any(character.isspace() for character in value):
            raise ValueError("queue name must be non-empty and contain no whitespace")
        return value
