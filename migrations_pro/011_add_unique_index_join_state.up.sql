BEGIN;

-- ============================================================
-- Add composite index for workflow_join_state lookups
-- ============================================================
-- 
-- Note: We cannot create a UNIQUE index on (instance_id, join_step_name) 
-- for a partitioned table without including the partitioning key (created_at).
-- Since created_at is set to time.Now() on each INSERT, a unique index 
-- including created_at would not prevent duplicate (instance_id, join_step_name) 
-- combinations.
-- 
-- Instead, we create a regular (non-unique) composite index for query performance.
-- Uniqueness is enforced at the application level using PostgreSQL advisory locks
-- in the CreateJoinState function to prevent race conditions.

CREATE INDEX IF NOT EXISTS idx_workflow_join_state_instance_join
    ON workflows.workflow_join_state (instance_id, join_step_name);

COMMENT ON INDEX workflows.idx_workflow_join_state_instance_join IS 
    'Composite index for efficient lookups of join_state by (instance_id, join_step_name). Uniqueness is enforced at application level using advisory locks.';

COMMIT;
