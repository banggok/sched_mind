ALTER TABLE wbs_nodes
    DROP CONSTRAINT IF EXISTS wbs_nodes_group_lock_snapshot_consistent,
    DROP CONSTRAINT IF EXISTS wbs_nodes_group_override_auto_consistent,
    DROP CONSTRAINT IF EXISTS wbs_nodes_group_local_status_valid,
    DROP CONSTRAINT IF EXISTS wbs_nodes_group_scheduling_source_valid,
    DROP COLUMN IF EXISTS group_locked_scheduling_start_date,
    DROP COLUMN IF EXISTS group_locked_automatic_scheduling,
    DROP COLUMN IF EXISTS group_scheduling_version,
    DROP COLUMN IF EXISTS group_local_status,
    DROP COLUMN IF EXISTS group_scheduling_start_date,
    DROP COLUMN IF EXISTS group_automatic_scheduling,
    DROP COLUMN IF EXISTS group_scheduling_source;
