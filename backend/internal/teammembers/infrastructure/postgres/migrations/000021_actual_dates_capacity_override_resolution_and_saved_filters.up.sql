ALTER TABLE wbs_nodes
    ADD COLUMN actual_start DATE NULL;

UPDATE wbs_nodes
SET actual_start = actual_end
WHERE actual_end IS NOT NULL;

ALTER TABLE wbs_nodes
    ADD CONSTRAINT wbs_actual_pair
        CHECK ((actual_start IS NULL) = (actual_end IS NULL)),
    ADD CONSTRAINT wbs_actual_order
        CHECK (actual_end IS NULL OR actual_end >= actual_start);

ALTER TABLE task_schedule_allocations
    DROP CONSTRAINT task_schedule_allocations_timeline_valid;

ALTER TABLE task_schedule_allocations
    ADD CONSTRAINT task_schedule_allocations_timeline_valid
        CHECK (timeline IN ('execution', 'commitment', 'actual'));

CREATE UNIQUE INDEX capacity_overrides_active_duplicate_idx
    ON capacity_overrides (team_member_id, start_date, end_date, capacity)
    WHERE deleted_at IS NULL;

CREATE TABLE portfolio_saved_filters (
    id UUID PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    name_key VARCHAR(100) NOT NULL,
    project_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT portfolio_saved_filters_name_required CHECK (BTRIM(name) <> ''),
    CONSTRAINT portfolio_saved_filters_name_unique UNIQUE (name_key),
    CONSTRAINT portfolio_saved_filters_version_positive CHECK (version > 0)
);

CREATE INDEX portfolio_saved_filters_order_idx
    ON portfolio_saved_filters (name_key, id);
