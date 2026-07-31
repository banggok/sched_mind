ALTER TABLE projects
    ADD COLUMN schedule_version BIGINT NOT NULL DEFAULT 0;

ALTER TABLE wbs_nodes
    ADD COLUMN execution_unscheduled_reason VARCHAR(160),
    ADD COLUMN commitment_unscheduled_reason VARCHAR(160);

CREATE TABLE task_schedule_allocations (
    task_id UUID NOT NULL REFERENCES wbs_nodes(id) ON DELETE CASCADE,
    assignee_id UUID NOT NULL REFERENCES team_members(id),
    timeline VARCHAR(10) NOT NULL,
    allocation_date DATE NOT NULL,
    allocated_minutes NUMERIC(12,6) NOT NULL,
    remaining_capacity_minutes NUMERIC(12,6) NOT NULL,
    sequence INTEGER NOT NULL,
    PRIMARY KEY (task_id, timeline, allocation_date),
    CONSTRAINT task_schedule_allocations_timeline_valid CHECK (timeline IN ('execution', 'commitment')),
    CONSTRAINT task_schedule_allocations_minutes_positive CHECK (allocated_minutes > 0),
    CONSTRAINT task_schedule_allocations_remaining_non_negative CHECK (remaining_capacity_minutes >= 0),
    CONSTRAINT task_schedule_allocations_sequence_positive CHECK (sequence > 0)
);

CREATE INDEX task_schedule_allocations_member_date_idx
    ON task_schedule_allocations (assignee_id, timeline, allocation_date, sequence, task_id);
