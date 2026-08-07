ALTER TABLE wbs_nodes
    ADD COLUMN group_scheduling_source VARCHAR(10) NOT NULL DEFAULT 'inherit',
    ADD COLUMN group_automatic_scheduling BOOLEAN NULL,
    ADD COLUMN group_scheduling_start_date DATE NULL,
    ADD COLUMN group_local_status VARCHAR(10) NOT NULL DEFAULT 'open',
    ADD COLUMN group_scheduling_version BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN group_locked_automatic_scheduling BOOLEAN NULL,
    ADD COLUMN group_locked_scheduling_start_date DATE NULL;

ALTER TABLE wbs_nodes
    ADD CONSTRAINT wbs_nodes_group_scheduling_source_valid
        CHECK (group_scheduling_source IN ('inherit', 'override')),
    ADD CONSTRAINT wbs_nodes_group_local_status_valid
        CHECK (group_local_status IN ('open', 'locked')),
    ADD CONSTRAINT wbs_nodes_group_override_auto_consistent
        CHECK (group_scheduling_source <> 'override' OR group_automatic_scheduling IS NOT NULL),
    ADD CONSTRAINT wbs_nodes_group_lock_snapshot_consistent
        CHECK (group_local_status <> 'locked' OR group_locked_automatic_scheduling IS NOT NULL);

