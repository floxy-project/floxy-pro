BEGIN;

CREATE SCHEMA IF NOT EXISTS partman;
CREATE EXTENSION IF NOT EXISTS pg_partman SCHEMA partman;

-- ============================================================
-- 1. workflow_instances_p
-- ============================================================

CREATE TABLE workflows.workflow_instances_p
(
    id           bigserial PRIMARY KEY,
    workflow_id  text NOT NULL REFERENCES workflows.workflow_definitions(id),
    status       text NOT NULL CHECK (status IN ('pending','running','completed','failed','rolling_back','cancelled','cancelling','aborted','dlq')),
    input        jsonb,
    output       jsonb,
    error        text,
    started_at   timestamptz,
    completed_at timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
)
    PARTITION BY RANGE (created_at);

SELECT partman.create_parent(
    p_parent_table => 'workflows.workflow_instances_p',
    p_control => 'created_at',
    p_type => 'native',
    p_interval => 'monthly',
    p_premake => 1,
    p_retention := 12,
    p_retention_keep_table := true
);

INSERT INTO workflows.workflow_instances_p
SELECT * FROM workflows.workflow_instances;

ANALYZE workflows.workflow_instances_p;

-- ============================================================
-- 2. workflow_steps_p
-- ============================================================

CREATE TABLE workflows.workflow_steps_p
(
    id bigserial PRIMARY KEY,
    instance_id bigint NOT NULL,
    step_name text NOT NULL,
    step_type text NOT NULL CHECK (step_type IN ('task','parallel','condition','fork','join','save_point','human')),
    status text NOT NULL CHECK (status IN ('pending','running','completed','failed','skipped','compensation','rolled_back','waiting_decision','confirmed','rejected','paused')),
    input jsonb,
    output jsonb,
    error text,
    retry_count integer NOT NULL DEFAULT 0,
    max_retries integer NOT NULL DEFAULT 3,
    started_at timestamptz,
    completed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    compensation_retry_count integer NOT NULL DEFAULT 0,
    idempotency_key uuid NOT NULL DEFAULT gen_random_uuid()
)
    PARTITION BY RANGE (created_at);

SELECT partman.create_parent(
    p_parent_table => 'workflows.workflow_steps_p',
    p_control => 'created_at',
    p_type => 'native',
    p_interval => 'monthly',
    p_premake => 1,
    p_retention := 12,
    p_retention_keep_table := true
);

INSERT INTO workflows.workflow_steps_p
SELECT * FROM workflows.workflow_steps;

ANALYZE workflows.workflow_steps_p;

-- ============================================================
-- 3. workflow_events_p
-- ============================================================

CREATE TABLE workflows.workflow_events_p
(
    id bigserial PRIMARY KEY,
    instance_id bigint NOT NULL,
    step_id bigint,
    event_type text NOT NULL,
    payload jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
)
    PARTITION BY RANGE (created_at);

SELECT partman.create_parent(
    p_parent_table => 'workflows.workflow_events_p',
    p_control => 'created_at',
    p_type => 'native',
    p_interval => 'monthly',
    p_premake => 1,
    p_retention := 6,
    p_retention_keep_table := true
);

INSERT INTO workflows.workflow_events_p
SELECT * FROM workflows.workflow_events;

ANALYZE workflows.workflow_events_p;

-- ============================================================
-- 4. workflow_dlq_p
-- ============================================================

CREATE TABLE workflows.workflow_dlq_p
(
    id bigserial PRIMARY KEY,
    instance_id bigint NOT NULL,
    workflow_id text NOT NULL,
    step_id bigint NOT NULL,
    step_name text NOT NULL,
    step_type text NOT NULL,
    input jsonb,
    error text,
    reason text,
    created_at timestamptz DEFAULT now() NOT NULL
)
    PARTITION BY RANGE (created_at);

SELECT partman.create_parent(
    p_parent_table => 'workflows.workflow_dlq_p',
    p_control => 'created_at',
    p_type => 'native',
    p_interval => 'monthly',
    p_premake => 1,
    p_retention := 3,
    p_retention_keep_table := true
);

INSERT INTO workflows.workflow_dlq_p
SELECT * FROM workflows.workflow_dlq;

ANALYZE workflows.workflow_dlq_p;

-- ============================================================
-- 5. workflow_human_decisions_p
-- ============================================================

CREATE TABLE workflows.workflow_human_decisions_p
(
    id bigserial PRIMARY KEY,
    instance_id bigint NOT NULL,
    step_id bigint NOT NULL,
    decided_by text NOT NULL,
    decision text NOT NULL CHECK (decision IN ('confirmed','rejected')),
    comment text,
    decided_at timestamptz DEFAULT now() NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    UNIQUE (step_id, decided_by)
)
    PARTITION BY RANGE (created_at);

SELECT partman.create_parent(
    p_parent_table => 'workflows.workflow_human_decisions_p',
    p_control => 'created_at',
    p_type => 'native',
    p_interval => 'monthly',
    p_premake => 1,
    p_retention := 12,
    p_retention_keep_table := true
);

INSERT INTO workflows.workflow_human_decisions_p
SELECT * FROM workflows.workflow_human_decisions;

ANALYZE workflows.workflow_human_decisions_p;

-- ============================================================
-- 6. workflow_join_state_p
-- ============================================================

CREATE TABLE workflows.workflow_join_state_p
(
    id bigserial PRIMARY KEY,
    instance_id bigint NOT NULL,
    join_step_name text NOT NULL,
    waiting_for jsonb NOT NULL,
    completed jsonb DEFAULT '[]'::jsonb NOT NULL,
    failed jsonb DEFAULT '[]'::jsonb NOT NULL,
    join_strategy text DEFAULT 'all' NOT NULL,
    is_ready boolean DEFAULT false NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    UNIQUE (instance_id, join_step_name)
)
    PARTITION BY RANGE (created_at);

SELECT partman.create_parent(
    p_parent_table => 'workflows.workflow_join_state_p',
    p_control => 'created_at',
    p_type => 'native',
    p_interval => 'monthly',
    p_premake => 1,
    p_retention := 12,
    p_retention_keep_table := true
);

INSERT INTO workflows.workflow_join_state_p
SELECT * FROM workflows.workflow_join_state;

ANALYZE workflows.workflow_join_state_p;

COMMIT;
