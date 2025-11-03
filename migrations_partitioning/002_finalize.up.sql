BEGIN;

ALTER TABLE IF EXISTS workflows.workflow_steps DROP CONSTRAINT IF EXISTS workflow_steps_instance_id_fkey;
ALTER TABLE IF EXISTS workflows.workflow_events DROP CONSTRAINT IF EXISTS workflow_events_instance_id_fkey;

ALTER TABLE workflows.workflow_instances RENAME TO workflow_instances_old;
ALTER TABLE workflows.workflow_steps RENAME TO workflow_steps_old;
ALTER TABLE workflows.workflow_events RENAME TO workflow_events_old;
ALTER TABLE workflows.workflow_dlq RENAME TO workflow_dlq_old;
ALTER TABLE workflows.workflow_human_decisions RENAME TO workflow_human_decisions_old;
ALTER TABLE workflows.workflow_join_state RENAME TO workflow_join_state_old;

ALTER TABLE workflows.workflow_instances_p RENAME TO workflow_instances;
ALTER TABLE workflows.workflow_steps_p RENAME TO workflow_steps;
ALTER TABLE workflows.workflow_events_p RENAME TO workflow_events;
ALTER TABLE workflows.workflow_dlq_p RENAME TO workflow_dlq;
ALTER TABLE workflows.workflow_human_decisions_p RENAME TO workflow_human_decisions;
ALTER TABLE workflows.workflow_join_state_p RENAME TO workflow_join_state;

ALTER TABLE workflows.workflow_steps
    ADD CONSTRAINT workflow_steps_instance_id_fkey
        FOREIGN KEY (instance_id) REFERENCES workflows.workflow_instances(id) NOT VALID;

ALTER TABLE workflows.workflow_events
    ADD CONSTRAINT workflow_events_instance_id_fkey
        FOREIGN KEY (instance_id) REFERENCES workflows.workflow_instances(id) NOT VALID;

UPDATE partman.part_config
SET parent_table = 'workflows.workflow_instances'
WHERE parent_table = 'workflows.workflow_instances_p';

UPDATE partman.part_config
SET parent_table = 'workflows.workflow_steps'
WHERE parent_table = 'workflows.workflow_steps_p';

UPDATE partman.part_config
SET parent_table = 'workflows.workflow_events'
WHERE parent_table = 'workflows.workflow_events_p';

UPDATE partman.part_config
SET parent_table = 'workflows.workflow_dlq'
WHERE parent_table = 'workflows.workflow_dlq_p';

UPDATE partman.part_config
SET parent_table = 'workflows.workflow_human_decisions'
WHERE parent_table = 'workflows.workflow_human_decisions_p';

UPDATE partman.part_config
SET parent_table = 'workflows.workflow_join_state'
WHERE parent_table = 'workflows.workflow_join_state_p';

COMMIT;
