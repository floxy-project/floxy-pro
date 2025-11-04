BEGIN;

CREATE SCHEMA IF NOT EXISTS workflows;
CREATE SCHEMA IF NOT EXISTS partman;
CREATE EXTENSION IF NOT EXISTS pg_partman WITH SCHEMA partman;

-- ============================================================
-- 0. not partitioned tables
-- ============================================================

CREATE TABLE IF NOT EXISTS workflows.workflow_definitions
(
    id         TEXT                     NOT NULL PRIMARY KEY,
    name       TEXT                     NOT NULL,
    version    INTEGER                  NOT NULL,
    definition JSONB                    NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now() NOT NULL,
    UNIQUE (name, version)
);

COMMENT ON TABLE workflows.workflow_definitions IS 'Workflow templates with definition of the execution graph';
COMMENT ON COLUMN workflows.workflow_definitions.definition IS 'JSONB graph with adjacency list structure';

CREATE INDEX IF NOT EXISTS idx_workflow_definitions_name ON workflows.workflow_definitions (name);

-- ============================================================
-- 1. workflow_instances (ПАРТИЦИОНИРОВАННАЯ)
-- ============================================================

CREATE TABLE workflows.workflow_instances
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

-- Создаем индекс на id для поиска (не уникальный, т.к. партиционированная таблица)
CREATE INDEX idx_workflow_instances_id ON workflows.workflow_instances(id);

SELECT partman.create_parent(
               p_parent_table => 'workflows.workflow_instances',
               p_control => 'created_at',
               p_interval => '1 day',
               p_premake => 30
       );

UPDATE partman.part_config
SET retention = '90 days',
    retention_keep_table = false,
    retention_keep_index = false,
    infinite_time_partitions = true
WHERE parent_table = 'workflows.workflow_instances';

-- ============================================================
-- 2. workflow_steps (НЕ ПАРТИЦИОНИРОВАННАЯ для простоты FK)
-- ============================================================

CREATE TABLE workflows.workflow_steps
(
    id                       BIGSERIAL PRIMARY KEY,
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
    idempotency_key          UUID NOT NULL DEFAULT gen_random_uuid()
);

-- Индексы для производительности
CREATE INDEX idx_workflow_steps_instance_id ON workflows.workflow_steps(instance_id);
CREATE INDEX idx_workflow_steps_created_at ON workflows.workflow_steps(created_at);
CREATE INDEX idx_workflow_steps_status ON workflows.workflow_steps(status);
CREATE INDEX idx_workflow_steps_step_name ON workflows.workflow_steps(step_name);

-- ============================================================
-- 3. workflow_events (ПАРТИЦИОНИРОВАННАЯ)
-- ============================================================

CREATE TABLE workflows.workflow_events
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

CREATE INDEX idx_workflow_events_id ON workflows.workflow_events(id);
CREATE INDEX idx_workflow_events_instance_id ON workflows.workflow_events(instance_id);

SELECT partman.create_parent(
               p_parent_table => 'workflows.workflow_events',
               p_control => 'created_at',
               p_interval => '1 day',
               p_premake => 30
       );

UPDATE partman.part_config
SET retention = '90 days',
    retention_keep_table = false,
    retention_keep_index = false,
    infinite_time_partitions = true
WHERE parent_table = 'workflows.workflow_events';

-- ============================================================
-- 4. workflow_dlq (ПАРТИЦИОНИРОВАННАЯ)
-- ============================================================

CREATE TABLE workflows.workflow_dlq
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

CREATE INDEX idx_workflow_dlq_id ON workflows.workflow_dlq(id);
CREATE INDEX idx_workflow_dlq_instance_id ON workflows.workflow_dlq(instance_id);

SELECT partman.create_parent(
               p_parent_table => 'workflows.workflow_dlq',
               p_control => 'created_at',
               p_interval => '1 day',
               p_premake => 30
       );

UPDATE partman.part_config
SET retention = '90 days',
    retention_keep_table = false,
    retention_keep_index = false,
    infinite_time_partitions = true
WHERE parent_table = 'workflows.workflow_dlq';

-- ============================================================
-- 5. workflow_human_decisions (НЕ ПАРТИЦИОНИРОВАННАЯ)
-- ============================================================

CREATE TABLE workflows.workflow_human_decisions
(
    id         BIGSERIAL PRIMARY KEY,
    instance_id BIGINT NOT NULL,
    step_id    BIGINT NOT NULL,
    decided_by TEXT NOT NULL,
    decision   TEXT NOT NULL CHECK (decision IN ('confirmed','rejected')),
    comment    TEXT,
    decided_at TIMESTAMPTZ DEFAULT now() NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now() NOT NULL,
    UNIQUE (step_id, decided_by)
);

CREATE INDEX idx_workflow_human_decisions_instance_id ON workflows.workflow_human_decisions(instance_id);
CREATE INDEX idx_workflow_human_decisions_step_id ON workflows.workflow_human_decisions(step_id);
CREATE INDEX idx_workflow_human_decisions_created_at ON workflows.workflow_human_decisions(created_at);

-- ============================================================
-- 6. workflow_join_state (НЕ ПАРТИЦИОНИРОВАННАЯ)
-- ============================================================

CREATE TABLE workflows.workflow_join_state
(
    id             BIGSERIAL PRIMARY KEY,
    instance_id    BIGINT NOT NULL,
    join_step_name TEXT NOT NULL,
    waiting_for    JSONB NOT NULL,
    completed      JSONB DEFAULT '[]'::jsonb NOT NULL,
    failed         JSONB DEFAULT '[]'::jsonb NOT NULL,
    join_strategy  TEXT DEFAULT 'all' NOT NULL,
    is_ready       BOOLEAN DEFAULT false NOT NULL,
    created_at     TIMESTAMPTZ DEFAULT now() NOT NULL,
    updated_at     TIMESTAMPTZ DEFAULT now() NOT NULL,
    UNIQUE (instance_id, join_step_name)
);

CREATE INDEX idx_workflow_join_state_instance ON workflows.workflow_join_state(instance_id);
CREATE INDEX idx_workflow_join_state_created_at ON workflows.workflow_join_state(created_at);

COMMIT;
