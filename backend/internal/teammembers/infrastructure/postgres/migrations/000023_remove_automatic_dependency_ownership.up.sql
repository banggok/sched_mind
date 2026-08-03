DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM task_dependencies
        WHERE NOT manual_owned AND NOT automatic_owned
    ) THEN
        RAISE EXCEPTION 'task_dependencies contains a legacy relation without an owner';
    END IF;
END $$;

DELETE FROM task_dependencies
WHERE automatic_owned AND NOT manual_owned;

ALTER TABLE task_dependencies
    DROP CONSTRAINT task_dependencies_has_owner,
    DROP COLUMN manual_owned,
    DROP COLUMN automatic_owned;
