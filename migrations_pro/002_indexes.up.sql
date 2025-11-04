CREATE INDEX IF NOT EXISTS idx_workflow_instances_workflow_id
    ON workflows.workflow_instances (workflow_id);
CREATE INDEX IF NOT EXISTS idx_workflow_instances_status
    ON workflows.workflow_instances (status);
CREATE INDEX IF NOT EXISTS idx_workflow_events_event_type
    ON workflows.workflow_events (event_type);
