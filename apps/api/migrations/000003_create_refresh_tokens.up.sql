-- Hashed refresh tokens. The raw token is never stored.

CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    replaced_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT refresh_tokens_hash_key UNIQUE (token_hash),
    CONSTRAINT refresh_tokens_hash_length CHECK (char_length(token_hash) = 64),
    CONSTRAINT refresh_tokens_hash_hex CHECK (token_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT refresh_tokens_replaced_by_fk FOREIGN KEY (replaced_by) REFERENCES refresh_tokens (id)
);

CREATE INDEX refresh_tokens_user_id_idx ON refresh_tokens (user_id);
