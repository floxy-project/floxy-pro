BEGIN;

CREATE SCHEMA IF NOT EXISTS workflows;
CREATE SCHEMA IF NOT EXISTS partman;
CREATE EXTENSION IF NOT EXISTS pg_partman WITH SCHEMA partman;

-- ============================================================
-- 1. workflow_instances_p
-- ============================================================

CREATE TABLE workflows.workflow_instances_p
(
    id           BIGSERIAL,
    workflow_id  TEXT NOT NULL REFERENCES workflows.workflow_definitions(id),
    status       TEXT NOT NULL CHECK (status IN ('pending','running','completed','failed','rolling_back','cancelled','cancelling','aborted','dlq')),
    input        JSONB,
    output       JSONB,
    error        TEXT,
    started_at   TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (id, created_at)
)
    PARTITION BY RANGE (created_at);

CREATE INDEX idx_workflow_instances_id ON workflows.workflow_instances_p(id);

SELECT partman.create_parent(
               p_parent_table => 'workflows.workflow_instances_p',
               p_control => 'created_at',
               p_interval => '1 day',
               p_premake => 30
       );

UPDATE partman.part_config
SET retention = '90 days',
    retention_keep_table = false,
    retention_keep_index = false,
    infinite_time_partitions = true
WHERE parent_table = 'workflows.workflow_instances_p';

INSERT INTO workflows.workflow_instances_p
SELECT * FROM workflows.workflow_instances;

ANALYZE workflows.workflow_instances_p;

-- ============================================================
-- 2. workflow_steps_p
-- ============================================================
CREATE TABLE workflows.workflow_steps_p
(
    id                       BIGSERIAL,
    instance_id              BIGINT NOT NULL,
    step_name                TEXT NOT NULL,
    step_type                TEXT NOT NULL CHECK (step_type IN ('task','parallel','condition','fork','join','save_point','human')),
    status                   TEXT NOT NULL CHECK (status IN ('pending','running','completed','failed','skipped','compensation','rolled_back','waiting_decision','confirmed','rejected','paused')),
    input                    JSONB,
    output                   JSONB,
    error                    TEXT,
    retry_count              INTEGER NOT NULL DEFAULT 0,
    max_retries              INTEGER NOT NULL DEFAULT 3,
    started_at               TIMESTAMPTZ,
    completed_at             TIMESTAMPTZ,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    compensation_retry_count INTEGER NOT NULL DEFAULT 0,
    idempotency_key          UUID NOT NULL DEFAULT gen_random_uuid(),
    PRIMARY KEY (id, created_at)
)
    PARTITION BY RANGE (created_at);

CREATE INDEX idx_workflow_steps_id ON workflows.workflow_steps_p (id);
CREATE INDEX idx_workflow_steps_instance_id ON workflows.workflow_steps_p (instance_id);
CREATE INDEX idx_workflow_steps_status ON workflows.workflow_steps_p (status);
CREATE INDEX idx_workflow_steps_step_name ON workflows.workflow_steps_p (step_name);
CREATE INDEX idx_workflow_steps_created_at ON workflows.workflow_steps_p (created_at);

SELECT partman.create_parent(
               p_parent_table => 'workflows.workflow_steps_p',
               p_control => 'created_at',
               p_interval => '1 day',
               p_premake => 30
       );

UPDATE partman.part_config
SET retention = '90 days',
    retention_keep_table = false,
    retention_keep_index = false,
    infinite_time_partitions = true
WHERE parent_table = 'workflows.workflow_steps_p';

INSERT INTO workflows.workflow_steps_p
SELECT * FROM workflows.workflow_steps;

ANALYZE workflows.workflow_steps_p;

-- ============================================================
-- 3. workflow_events_p
-- ============================================================

CREATE TABLE workflows.workflow_events_p
(
    id          BIGSERIAL,
    instance_id BIGINT NOT NULL,
    step_id     BIGINT,
    event_type  TEXT NOT NULL,
    payload     JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (id, created_at)
)
    PARTITION BY RANGE (created_at);

CREATE INDEX idx_workflow_events_id ON workflows.workflow_events_p(id);
CREATE INDEX idx_workflow_events_instance_id ON workflows.workflow_events_p(instance_id);

SELECT partman.create_parent(
               p_parent_table => 'workflows.workflow_events_p',
               p_control => 'created_at',
               p_interval => '1 day',
               p_premake => 30
       );

UPDATE partman.part_config
SET retention = '90 days',
    retention_keep_table = false,
    retention_keep_index = false,
    infinite_time_partitions = true
WHERE parent_table = 'workflows.workflow_events_p';

INSERT INTO workflows.workflow_events_p
SELECT * FROM workflows.workflow_events;

ANALYZE workflows.workflow_events_p;

-- ============================================================
-- 4. workflow_dlq_p
-- ============================================================

CREATE TABLE workflows.workflow_dlq_p
(
    id          BIGSERIAL,
    instance_id BIGINT NOT NULL,
    workflow_id TEXT NOT NULL,
    step_id     BIGINT NOT NULL,
    step_name   TEXT NOT NULL,
    step_type   TEXT NOT NULL,
    input       JSONB,
    error       TEXT,
    reason      TEXT,
    created_at  TIMESTAMPTZ DEFAULT now() NOT NULL,
    PRIMARY KEY (id, created_at)
)
    PARTITION BY RANGE (created_at);

CREATE INDEX idx_workflow_dlq_id ON workflows.workflow_dlq_p(id);
CREATE INDEX idx_workflow_dlq_instance_id ON workflows.workflow_dlq_p(instance_id);

SELECT partman.create_parent(
               p_parent_table => 'workflows.workflow_dlq_p',
               p_control => 'created_at',
               p_interval => '1 day',
               p_premake => 30
       );

UPDATE partman.part_config
SET retention = '90 days',
    retention_keep_table = false,
    retention_keep_index = false,
    infinite_time_partitions = true
WHERE parent_table = 'workflows.workflow_dlq_p';

INSERT INTO workflows.workflow_dlq_p
SELECT * FROM workflows.workflow_dlq;

ANALYZE workflows.workflow_dlq_p;

-- ============================================================
-- 5. workflow_join_state_p
-- ============================================================

CREATE TABLE workflows.workflow_join_state_p
(
    id             BIGSERIAL,
    instance_id    BIGINT NOT NULL,
    join_step_name TEXT NOT NULL,
    waiting_for    JSONB NOT NULL,
    completed      JSONB DEFAULT '[]'::jsonb NOT NULL,
    failed         JSONB DEFAULT '[]'::jsonb NOT NULL,
    join_strategy  TEXT DEFAULT 'all' NOT NULL,
    is_ready       BOOLEAN DEFAULT false NOT NULL,
    created_at     TIMESTAMPTZ DEFAULT now() NOT NULL,
    updated_at     TIMESTAMPTZ DEFAULT now() NOT NULL,
    PRIMARY KEY (id, created_at)
)
    PARTITION BY RANGE (created_at);

COMMENT ON TABLE workflows.workflow_join_state_p IS 'Join synchronization state for parallel branches';

CREATE INDEX idx_workflow_join_state_instance_id
    ON workflows.workflow_join_state_p (instance_id);
CREATE INDEX idx_workflow_join_state_join_name
    ON workflows.workflow_join_state_p (join_step_name);
CREATE INDEX idx_workflow_join_state_created_at
    ON workflows.workflow_join_state_p (created_at);

SELECT partman.create_parent(
               p_parent_table => 'workflows.workflow_join_state_p',
               p_control => 'created_at',
               p_interval => '1 day',
               p_premake => 30
       );

UPDATE partman.part_config
SET retention = '90 days',
    retention_keep_table = false,
    retention_keep_index = false,
    infinite_time_partitions = true
WHERE parent_table = 'workflows.workflow_join_state_p';

INSERT INTO workflows.workflow_join_state_p
SELECT * FROM workflows.workflow_join_state;

ANALYZE workflows.workflow_join_state_p;

COMMIT;
