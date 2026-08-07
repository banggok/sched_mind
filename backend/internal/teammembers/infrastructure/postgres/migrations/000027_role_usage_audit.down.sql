DROP INDEX IF EXISTS wbs_nodes_role_id_idx;

ALTER TABLE team_members
    DROP CONSTRAINT IF EXISTS team_members_active_role_required;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM team_members WHERE role_id IS NULL) THEN
        RAISE EXCEPTION 'cannot restore team_members.role_id NOT NULL after historical Role references were detached';
    END IF;

    ALTER TABLE team_members
        ALTER COLUMN role_id SET NOT NULL;
END
$$;
