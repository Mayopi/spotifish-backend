-- Allow sync jobs to enter a paused state. The original CHECK constraint only
-- accepted (queued, running, succeeded, failed); we drop and re-add it so the
-- sync worker can persist a job as 'paused' when the user pauses an in-flight
-- sync from the Settings screen.
ALTER TABLE sync_jobs DROP CONSTRAINT IF EXISTS sync_jobs_state_check;
ALTER TABLE sync_jobs ADD CONSTRAINT sync_jobs_state_check
    CHECK (state IN ('queued', 'running', 'paused', 'succeeded', 'failed'));

-- Optional audit column. Nullable so existing rows don't need a default.
ALTER TABLE sync_jobs ADD COLUMN IF NOT EXISTS paused_at TIMESTAMPTZ;
