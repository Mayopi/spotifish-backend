CREATE TABLE drive_credentials (
    user_id           UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    encrypted_access  BYTEA NOT NULL,
    encrypted_refresh BYTEA NOT NULL,
    expires_at        TIMESTAMPTZ NOT NULL,
    scope             TEXT NOT NULL,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE drive_folders (
    user_id        UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    folder_id      TEXT NOT NULL,
    folder_name    TEXT NOT NULL,
    last_synced_at TIMESTAMPTZ
);
