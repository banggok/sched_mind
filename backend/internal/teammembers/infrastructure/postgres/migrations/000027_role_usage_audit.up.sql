ALTER TABLE team_members
    ALTER COLUMN role_id DROP NOT NULL;

ALTER TABLE team_members
    ADD CONSTRAINT team_members_active_role_required
        CHECK (deleted_at IS NOT NULL OR role_id IS NOT NULL);

CREATE INDEX IF NOT EXISTS wbs_nodes_role_id_idx
    ON wbs_nodes(role_id);
