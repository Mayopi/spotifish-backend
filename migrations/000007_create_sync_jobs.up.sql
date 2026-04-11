CREATE TABLE sync_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    state           TEXT NOT NULL CHECK (state IN ('queued', 'running', 'succeeded', 'failed')),
    processed_count INTEGER NOT NULL DEFAULT 0,
    total_count     INTEGER,
    last_error      TEXT,
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX sync_jobs_user_id_idx ON sync_jobs (user_id, created_at DESC);
