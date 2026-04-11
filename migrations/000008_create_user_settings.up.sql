CREATE TABLE user_settings (
    user_id            UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    theme              TEXT NOT NULL DEFAULT 'system',
    default_sort_field TEXT NOT NULL DEFAULT 'title',
    default_sort_dir   TEXT NOT NULL DEFAULT 'asc',
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE playback_events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    song_id    UUID NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL CHECK (event_type IN ('started', 'completed', 'skipped')),
    position_ms BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX playback_events_user_id_idx ON playback_events (user_id, created_at DESC);
