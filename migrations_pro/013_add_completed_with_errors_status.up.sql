-- Add 'completed_with_errors' workflow instance status (partial success failure policy).
--
-- workflow_instances is partitioned: constraints added/dropped on the parent table
-- are propagated to all existing partitions and inherited by partitions created by pg_partman.
--
-- The constraint name depends on how the schema was created:
--   * migrations_pro/001_initial         -> workflow_instances_status_check
--   * migrations_partitioning (_p rename) -> workflow_instances_p_status_check

BEGIN;

ALTER TABLE workflows.workflow_instances DROP CONSTRAINT IF EXISTS workflow_instances_status_check;
ALTER TABLE workflows.workflow_instances DROP CONSTRAINT IF EXISTS workflow_instances_p_status_check;
ALTER TABLE workflows.workflow_instances ADD CONSTRAINT workflow_instances_status_check
    CHECK (status IN ('pending', 'running', 'completed', 'completed_with_errors', 'failed', 'rolling_back', 'cancelled', 'cancelling', 'aborted', 'dlq'));

COMMENT ON COLUMN workflows.workflow_instances.status IS 'pending | running | completed | completed_with_errors | failed | rolling_back | cancelled | cancelling | aborted | dlq';

COMMIT;
