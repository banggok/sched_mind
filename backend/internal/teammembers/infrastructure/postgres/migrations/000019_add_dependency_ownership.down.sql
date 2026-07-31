DELETE FROM task_dependencies WHERE manual_owned = FALSE;
ALTER TABLE task_dependencies
    DROP CONSTRAINT IF EXISTS task_dependencies_has_owner,
    DROP COLUMN IF EXISTS automatic_owned,
    DROP COLUMN IF EXISTS manual_owned;
