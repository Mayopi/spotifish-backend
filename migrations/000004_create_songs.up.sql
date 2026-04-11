CREATE TABLE songs (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source               TEXT NOT NULL CHECK (source IN ('drive', 'local')),
    source_file_id       TEXT NOT NULL,
    title                TEXT NOT NULL,
    artist               TEXT NOT NULL,
    album                TEXT NOT NULL,
    duration_ms          BIGINT NOT NULL DEFAULT 0,
    mime_type            TEXT,
    album_art_object_key TEXT,
    drive_modified_at    TIMESTAMPTZ,
    added_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, source, source_file_id)
);

CREATE INDEX songs_user_title_idx  ON songs (user_id, lower(title));
CREATE INDEX songs_user_artist_idx ON songs (user_id, lower(artist));
CREATE INDEX songs_user_album_idx  ON songs (user_id, lower(album));
CREATE INDEX songs_fts_idx         ON songs USING gin (to_tsvector('simple', title || ' ' || artist || ' ' || album));
