CREATE INDEX team_members_deleted_at_idx
    ON team_members (deleted_at);

CREATE INDEX capacity_overrides_deleted_at_idx
    ON capacity_overrides (deleted_at);

CREATE INDEX capacity_overrides_team_member_id_idx
    ON capacity_overrides (team_member_id);
