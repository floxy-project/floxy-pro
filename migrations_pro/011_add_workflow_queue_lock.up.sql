ALTER TABLE workflows.workflow_queue
    ADD COLUMN IF NOT EXISTS locked_until TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_workflow_queue_locked_until
    ON workflows.workflow_queue(locked_until)
    WHERE locked_until IS NOT NULL;

COMMENT ON COLUMN workflows.workflow_queue.locked_until IS
    'Queue item lease expiration; workers may reclaim attempted items after this timestamp';
