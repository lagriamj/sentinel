-- Assessment records owned by the API. A stored job stays pending until coordination is added.

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT users_email_not_blank CHECK (length(btrim(email)) > 0),
    CONSTRAINT users_email_length CHECK (char_length(email) <= 320),
    CONSTRAINT users_email_format CHECK (email ~* '^[^[:space:]@]+@[^[:space:]@]+\.[^[:space:]@]+$'),
    CONSTRAINT users_password_hash_prefix CHECK (password_hash LIKE '$argon2id$%'),
    CONSTRAINT users_password_hash_length CHECK (char_length(password_hash) BETWEEN 16 AND 512)
);

CREATE UNIQUE INDEX users_email_lower_key ON users (lower(email));

CREATE TABLE scan_targets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    url TEXT NOT NULL,
    scheme TEXT NOT NULL,
    host TEXT NOT NULL,
    port INTEGER NOT NULL,
    authorized BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT scan_targets_url_key UNIQUE (url),
    CONSTRAINT scan_targets_url_not_blank CHECK (length(btrim(url)) > 0),
    CONSTRAINT scan_targets_url_length CHECK (char_length(url) <= 2048),
    CONSTRAINT scan_targets_scheme_allowed CHECK (scheme IN ('http', 'https')),
    CONSTRAINT scan_targets_host_not_blank CHECK (length(btrim(host)) > 0),
    CONSTRAINT scan_targets_host_length CHECK (char_length(host) <= 253),
    CONSTRAINT scan_targets_port_range CHECK (port BETWEEN 1 AND 65535),
    CONSTRAINT scan_targets_authorized CHECK (authorized)
);

CREATE INDEX scan_targets_host_port_idx ON scan_targets (host, port);

CREATE TABLE scan_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_id UUID NOT NULL REFERENCES scan_targets (id) ON DELETE RESTRICT,
    created_by UUID REFERENCES users (id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    error_message TEXT,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT scan_jobs_name_not_blank CHECK (length(btrim(name)) > 0),
    CONSTRAINT scan_jobs_name_length CHECK (char_length(name) <= 200),
    CONSTRAINT scan_jobs_status_allowed CHECK (
        status IN ('pending', 'queued', 'running', 'completed', 'failed', 'cancelled')
    ),
    CONSTRAINT scan_jobs_error_message_length CHECK (
        error_message IS NULL OR char_length(error_message) <= 1000
    ),
    CONSTRAINT scan_jobs_finished_at_matches_status CHECK (
        (
            status IN ('completed', 'failed', 'cancelled')
            AND finished_at IS NOT NULL
        )
        OR (
            status IN ('pending', 'queued', 'running')
            AND finished_at IS NULL
        )
    )
);

CREATE INDEX scan_jobs_created_at_id_idx ON scan_jobs (created_at DESC, id DESC);
CREATE INDEX scan_jobs_status_created_at_id_idx ON scan_jobs (status, created_at DESC, id DESC);
CREATE INDEX scan_jobs_target_created_at_id_idx ON scan_jobs (target_id, created_at DESC, id DESC);
CREATE INDEX scan_jobs_created_by_idx ON scan_jobs (created_by);

CREATE TABLE scan_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scan_job_id UUID NOT NULL REFERENCES scan_jobs (id) ON DELETE CASCADE,
    engine TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT scan_results_engine_not_blank CHECK (length(btrim(engine)) > 0),
    CONSTRAINT scan_results_engine_length CHECK (char_length(engine) <= 64),
    CONSTRAINT scan_results_summary_length CHECK (char_length(summary) <= 4000),
    CONSTRAINT scan_results_job_engine_key UNIQUE (scan_job_id, engine)
);

CREATE TABLE vulnerabilities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scan_job_id UUID NOT NULL REFERENCES scan_jobs (id) ON DELETE CASCADE,
    scan_result_id UUID REFERENCES scan_results (id) ON DELETE SET NULL,
    target_id UUID NOT NULL REFERENCES scan_targets (id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    severity TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    location TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT vulnerabilities_name_not_blank CHECK (length(btrim(name)) > 0),
    CONSTRAINT vulnerabilities_name_length CHECK (char_length(name) <= 300),
    CONSTRAINT vulnerabilities_severity_allowed CHECK (
        severity IN ('info', 'low', 'medium', 'high', 'critical')
    ),
    CONSTRAINT vulnerabilities_description_length CHECK (char_length(description) <= 8000),
    CONSTRAINT vulnerabilities_location_length CHECK (char_length(location) <= 2048)
);

CREATE INDEX vulnerabilities_scan_job_id_idx ON vulnerabilities (scan_job_id);
CREATE INDEX vulnerabilities_target_id_idx ON vulnerabilities (target_id);
CREATE INDEX vulnerabilities_severity_idx ON vulnerabilities (severity);
