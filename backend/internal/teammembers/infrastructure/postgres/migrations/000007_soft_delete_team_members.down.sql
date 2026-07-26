DROP INDEX IF EXISTS capacity_overrides_deleted_at_idx;
DROP INDEX IF EXISTS team_members_deleted_at_idx;
DROP INDEX IF EXISTS capacity_overrides_active_member_period_idx;
DROP INDEX IF EXISTS team_members_active_name_prefix_search_idx;
DROP INDEX IF EXISTS team_members_active_name_ci_idx;

CREATE INDEX team_members_name_ci_id_idx
    ON team_members (LOWER(name), id);

CREATE INDEX team_members_name_prefix_search_idx
    ON team_members (LOWER(name) text_pattern_ops);

CREATE INDEX capacity_overrides_member_period_idx
    ON capacity_overrides (team_member_id, start_date, end_date, id);

ALTER TABLE capacity_overrides
    DROP COLUMN deleted_at;

ALTER TABLE team_members
    DROP COLUMN deleted_at;
