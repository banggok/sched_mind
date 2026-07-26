DROP INDEX IF EXISTS capacity_overrides_member_period_idx;

CREATE INDEX capacity_overrides_member_period_idx
    ON capacity_overrides(team_member_id, start_date, end_date, id);
