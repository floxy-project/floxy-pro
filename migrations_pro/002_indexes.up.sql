-- workflow_instances
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflow_instances_workflow_id
    ON ONLY workflows.workflow_instances (workflow_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflow_instances_status
    ON ONLY workflows.workflow_instances (status);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflow_instances_created_at
    ON ONLY workflows.workflow_instances (created_at DESC);

-- workflow_steps
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflow_steps_instance_id
    ON ONLY workflows.workflow_steps (instance_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflow_steps_status
    ON ONLY workflows.workflow_steps (status);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflow_steps_step_name
    ON ONLY workflows.workflow_steps (step_name);

-- workflow_events
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflow_events_instance_id
    ON ONLY workflows.workflow_events (instance_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflow_events_event_type
    ON ONLY workflows.workflow_events (event_type);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflow_events_created_at
    ON ONLY workflows.workflow_events (created_at DESC);

-- workflow_dlq
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflow_dlq_instance_id
    ON ONLY workflows.workflow_dlq (instance_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflow_dlq_created_at
    ON ONLY workflows.workflow_dlq (created_at DESC);

-- workflow_human_decisions
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflow_human_decisions_instance_id
    ON ONLY workflows.workflow_human_decisions (instance_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflow_human_decisions_step_id
    ON ONLY workflows.workflow_human_decisions (step_id);

-- workflow_join_state
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflow_join_state_instance
    ON ONLY workflows.workflow_join_state (instance_id);
