CREATE TABLE auth_refresh_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ
);

CREATE INDEX auth_refresh_tokens_user_id_idx ON auth_refresh_tokens (user_id);
CREATE INDEX auth_refresh_tokens_token_hash_idx ON auth_refresh_tokens (token_hash);
