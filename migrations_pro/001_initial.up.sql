BEGIN;

CREATE SCHEMA IF NOT EXISTS workflows;
CREATE SCHEMA IF NOT EXISTS partman;
CREATE EXTENSION IF NOT EXISTS pg_partman SCHEMA partman;

-- ============================================================
-- 0. not partitioned tables
-- ============================================================

create table if not exists workflows.workflow_definitions
(
    id         text                                   not null
        primary key,
    name       text                                   not null,
    version    integer                                not null,
    definition jsonb                                  not null,
    created_at timestamp with time zone default now() not null,
    unique (name, version)
);

comment on table workflows.workflow_definitions is 'Workflow templates with definition of the execution graph';
comment on column workflows.workflow_definitions.definition is 'JSONB graph with adjacency list structure';

create index if not exists idx_workflow_definitions_name on workflows.workflow_definitions (name);

---

create table if not exists workflows.workflow_queue
(
    id           bigserial
        primary key,
    instance_id  bigint                                 not null
        references workflows.workflow_instances
            on delete cascade,
    step_id      bigint
        references workflows.workflow_steps
            on delete cascade,
    scheduled_at timestamp with time zone default now() not null,
    attempted_at timestamp with time zone,
    attempted_by text,
    priority     integer                  default 0     not null
);

comment on table workflows.workflow_queue is 'Queue of steps for workers to complete';

create index if not exists idx_workflow_queue_scheduled
    on workflows.workflow_queue (scheduled_at asc, priority desc)
    where (attempted_at IS NULL);

create index if not exists idx_workflow_queue_instance_id on workflows.workflow_queue (instance_id);

---

create table if not exists workflows.workflow_cancel_requests
(
    id           bigserial
        primary key,
    instance_id  bigint                                 not null
        unique
        references workflows.workflow_instances
            on delete cascade,
    requested_by text                                   not null,
    cancel_type  text                                   not null
        constraint workflow_cancel_requests_type_check
            check (cancel_type = ANY (ARRAY ['cancel'::text, 'abort'::text])),
    reason       text,
    created_at   timestamp with time zone default now() not null
);

create index if not exists idx_workflow_cancel_requests_instance_id on workflows.workflow_cancel_requests (instance_id);

-- ============================================================
-- 1. workflow_instances
-- ============================================================

CREATE TABLE workflows.workflow_instances
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
               p_parent_table => 'workflows.workflow_instances',
               p_control => 'created_at',
               p_type => 'native',
               p_interval => 'monthly',
               p_premake => 1,
               p_retention := 12,
               p_retention_keep_table := true
       );

-- ============================================================
-- 2. workflow_steps
-- ============================================================

CREATE TABLE workflows.workflow_steps
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
               p_parent_table => 'workflows.workflow_steps',
               p_control => 'created_at',
               p_type => 'native',
               p_interval => 'monthly',
               p_premake => 1,
               p_retention := 12,
               p_retention_keep_table := true
       );

-- ============================================================
-- 3. workflow_events
-- ============================================================

CREATE TABLE workflows.workflow_events
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
               p_parent_table => 'workflows.workflow_events',
               p_control => 'created_at',
               p_type => 'native',
               p_interval => 'monthly',
               p_premake => 1,
               p_retention := 6,
               p_retention_keep_table := true
       );

-- ============================================================
-- 4. workflow_dlq
-- ============================================================

CREATE TABLE workflows.workflow_dlq
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
               p_parent_table => 'workflows.workflow_dlq',
               p_control => 'created_at',
               p_type => 'native',
               p_interval => 'monthly',
               p_premake => 1,
               p_retention := 3,
               p_retention_keep_table := true
       );

-- ============================================================
-- 5. workflow_human_decisions
-- ============================================================

CREATE TABLE workflows.workflow_human_decisions
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
               p_parent_table => 'workflows.workflow_human_decisions',
               p_control => 'created_at',
               p_type => 'native',
               p_interval => 'monthly',
               p_premake => 1,
               p_retention := 12,
               p_retention_keep_table := true
       );

-- ============================================================
-- 6. workflow_join_state
-- ============================================================

CREATE TABLE workflows.workflow_join_state
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
               p_parent_table => 'workflows.workflow_join_state',
               p_control => 'created_at',
               p_type => 'native',
               p_interval => 'monthly',
               p_premake => 1,
               p_retention := 12,
               p_retention_keep_table := true
       );

---

create or replace view active_workflows
            (id, workflow_id, status, created_at, updated_at, duration_seconds, total_steps, completed_steps,
             failed_steps, running_steps)
as
SELECT wi.id,
       wi.workflow_id,
       wi.status,
       wi.created_at,
       wi.updated_at,
       EXTRACT(epoch FROM now() - wi.created_at)                 AS duration_seconds,
       count(ws.id)                                              AS total_steps,
       count(ws.id) FILTER (WHERE ws.status = 'completed'::text) AS completed_steps,
       count(ws.id) FILTER (WHERE ws.status = 'failed'::text)    AS failed_steps,
       count(ws.id) FILTER (WHERE ws.status = 'running'::text)   AS running_steps
FROM workflows.workflow_instances wi
         LEFT JOIN workflows.workflow_steps ws ON wi.id = ws.instance_id
WHERE wi.status = ANY (ARRAY ['pending'::text, 'running'::text, 'dlq'::text])
GROUP BY wi.id, wi.workflow_id, wi.status, wi.created_at, wi.updated_at;

---

create or replace view workflow_stats (name, version, total_instances, completed, failed, running, avg_duration_seconds) as
SELECT wd.name,
       wd.version,
       count(wi.id)                                                                                          AS total_instances,
       count(wi.id) FILTER (WHERE wi.status = 'completed'::text)                                             AS completed,
       count(wi.id) FILTER (WHERE wi.status = 'failed'::text)                                                AS failed,
       count(wi.id) FILTER (WHERE wi.status = 'running'::text)                                               AS running,
       avg(EXTRACT(epoch FROM wi.completed_at - wi.created_at))
       FILTER (WHERE wi.status = 'completed'::text)                                                          AS avg_duration_seconds
FROM workflows.workflow_definitions wd
         LEFT JOIN workflows.workflow_instances wi ON wd.id = wi.workflow_id
GROUP BY wd.name, wd.version;

COMMIT;
