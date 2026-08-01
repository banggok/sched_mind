DROP INDEX IF EXISTS portfolio_saved_filters_order_idx;
DROP TABLE IF EXISTS portfolio_saved_filters;
DROP INDEX IF EXISTS capacity_overrides_active_duplicate_idx;

ALTER TABLE task_schedule_allocations
    DROP CONSTRAINT IF EXISTS task_schedule_allocations_timeline_valid;

ALTER TABLE task_schedule_allocations
    ADD CONSTRAINT task_schedule_allocations_timeline_valid
        CHECK (timeline IN ('execution', 'commitment'));

ALTER TABLE wbs_nodes
    DROP CONSTRAINT IF EXISTS wbs_actual_order,
    DROP CONSTRAINT IF EXISTS wbs_actual_pair,
    DROP COLUMN IF EXISTS actual_start;
