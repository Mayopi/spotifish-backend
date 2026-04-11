CREATE TABLE users (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    google_sub   TEXT UNIQUE NOT NULL,
    email        TEXT NOT NULL,
    display_name TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
