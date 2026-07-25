DROP TABLE IF EXISTS capacity_overrides;
DROP TABLE IF EXISTS executable_leaves;
DROP INDEX IF EXISTS team_members_name_ci_id_idx;
ALTER TABLE team_members
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS created_at,
    DROP COLUMN IF EXISTS buffer_percentage,
    DROP COLUMN IF EXISTS daily_capacity,
    DROP COLUMN IF EXISTS name;
