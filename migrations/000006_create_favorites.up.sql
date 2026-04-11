CREATE TABLE favorites (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    song_id    UUID NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, song_id)
);
