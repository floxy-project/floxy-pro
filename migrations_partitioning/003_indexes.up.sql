CREATE INDEX IF NOT EXISTS idx_workflow_instances_workflow_id
    ON workflows.workflow_instances (workflow_id);
CREATE INDEX IF NOT EXISTS idx_workflow_instances_status
    ON workflows.workflow_instances (status);
CREATE INDEX IF NOT EXISTS idx_workflow_events_event_type
    ON workflows.workflow_events (event_type);

CREATE INDEX IF NOT EXISTS idx_workflow_join_state_instance_join
    ON workflows.workflow_join_state (instance_id, join_step_name);

COMMENT ON INDEX workflows.idx_workflow_join_state_instance_join IS
    'Composite index for efficient lookups of join_state by (instance_id, join_step_name). Uniqueness is enforced at application level using advisory locks.';
