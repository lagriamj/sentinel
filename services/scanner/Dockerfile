FROM python:3.12.16-slim-bookworm

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1 \
    PIP_NO_CACHE_DIR=1

WORKDIR /app

RUN useradd --create-home --uid 10001 --shell /usr/sbin/nologin scanner

COPY pyproject.toml ./
COPY scanner ./scanner
RUN pip install --no-cache-dir .

USER scanner
ENTRYPOINT ["python", "-m", "scanner"]
