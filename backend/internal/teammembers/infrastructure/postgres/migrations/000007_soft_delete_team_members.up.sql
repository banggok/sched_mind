ALTER TABLE team_members
    ADD COLUMN deleted_at TIMESTAMPTZ;

ALTER TABLE capacity_overrides
    ADD COLUMN deleted_at TIMESTAMPTZ;

DROP INDEX IF EXISTS team_members_name_ci_id_idx;
DROP INDEX IF EXISTS team_members_name_prefix_search_idx;
DROP INDEX IF EXISTS capacity_overrides_member_period_idx;

CREATE INDEX team_members_active_name_ci_idx
    ON team_members (LOWER(name), id)
    WHERE deleted_at IS NULL;

CREATE INDEX team_members_active_name_prefix_search_idx
    ON team_members (LOWER(name) text_pattern_ops, id)
    WHERE deleted_at IS NULL;

CREATE INDEX capacity_overrides_active_member_period_idx
    ON capacity_overrides (team_member_id, start_date, end_date, id)
    WHERE deleted_at IS NULL;

CREATE INDEX team_members_deleted_at_idx
    ON team_members (deleted_at);

CREATE INDEX capacity_overrides_deleted_at_idx
    ON capacity_overrides (deleted_at);
