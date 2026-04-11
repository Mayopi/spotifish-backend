-- Re-tighten the state CHECK constraint and drop the audit column. Any rows
-- currently in 'paused' state must be migrated to 'failed' first or this will
-- error.
ALTER TABLE sync_jobs DROP CONSTRAINT IF EXISTS sync_jobs_state_check;
ALTER TABLE sync_jobs ADD CONSTRAINT sync_jobs_state_check
    CHECK (state IN ('queued', 'running', 'succeeded', 'failed'));

ALTER TABLE sync_jobs DROP COLUMN IF EXISTS paused_at;
