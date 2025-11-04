BEGIN;

-- ============================================================
-- workflow_queue (НЕ ПАРТИЦИОНИРОВАННАЯ)
-- ============================================================

CREATE TABLE IF NOT EXISTS workflows.workflow_queue
(
    id           BIGSERIAL PRIMARY KEY,
    instance_id  BIGINT NOT NULL,
    step_id      BIGINT REFERENCES workflows.workflow_steps(id) ON DELETE CASCADE,
    scheduled_at TIMESTAMPTZ DEFAULT now() NOT NULL,
    attempted_at TIMESTAMPTZ,
    attempted_by TEXT,
    priority     INTEGER DEFAULT 0 NOT NULL
);

COMMENT ON TABLE workflows.workflow_queue IS 'Queue of steps for workers to complete';

CREATE INDEX IF NOT EXISTS idx_workflow_queue_scheduled
    ON workflows.workflow_queue (scheduled_at ASC, priority DESC)
    WHERE (attempted_at IS NULL);

CREATE INDEX IF NOT EXISTS idx_workflow_queue_instance_id
    ON workflows.workflow_queue (instance_id);

-- ============================================================
-- workflow_cancel_requests (НЕ ПАРТИЦИОНИРОВАННАЯ)
-- ============================================================

CREATE TABLE IF NOT EXISTS workflows.workflow_cancel_requests
(
    id           BIGSERIAL PRIMARY KEY,
    instance_id  BIGINT NOT NULL UNIQUE,
    requested_by TEXT NOT NULL,
    cancel_type  TEXT NOT NULL CHECK (cancel_type IN ('cancel', 'abort')),
    reason       TEXT,
    created_at   TIMESTAMPTZ DEFAULT now() NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_workflow_cancel_requests_instance_id
    ON workflows.workflow_cancel_requests (instance_id);

-- ============================================================
-- VIEWS
-- ============================================================

CREATE OR REPLACE VIEW workflows.active_workflows AS
SELECT
    wi.id,
    wi.workflow_id,
    wi.status,
    wi.created_at,
    wi.updated_at,
    EXTRACT(epoch FROM now() - wi.created_at) AS duration_seconds,
    COUNT(ws.id) AS total_steps,
    COUNT(ws.id) FILTER (WHERE ws.status = 'completed') AS completed_steps,
    COUNT(ws.id) FILTER (WHERE ws.status = 'failed') AS failed_steps,
    COUNT(ws.id) FILTER (WHERE ws.status = 'running') AS running_steps
FROM workflows.workflow_instances wi
         LEFT JOIN workflows.workflow_steps ws ON wi.id = ws.instance_id
WHERE wi.status IN ('pending', 'running', 'dlq')
GROUP BY wi.id, wi.workflow_id, wi.status, wi.created_at, wi.updated_at;

CREATE OR REPLACE VIEW workflows.workflow_stats AS
SELECT
    wd.name,
    wd.version,
    COUNT(wi.id) AS total_instances,
    COUNT(wi.id) FILTER (WHERE wi.status = 'completed') AS completed,
    COUNT(wi.id) FILTER (WHERE wi.status = 'failed') AS failed,
    COUNT(wi.id) FILTER (WHERE wi.status = 'running') AS running,
    AVG(EXTRACT(epoch FROM wi.completed_at - wi.created_at))
    FILTER (WHERE wi.status = 'completed') AS avg_duration_seconds
FROM workflows.workflow_definitions wd
         LEFT JOIN workflows.workflow_instances wi ON wd.id = wi.workflow_id
GROUP BY wd.name, wd.version;

COMMIT;
