DROP TABLE IF EXISTS task_schedule_allocations;
ALTER TABLE wbs_nodes
    DROP COLUMN IF EXISTS commitment_unscheduled_reason,
    DROP COLUMN IF EXISTS execution_unscheduled_reason;
ALTER TABLE projects
    DROP COLUMN IF EXISTS schedule_version;
