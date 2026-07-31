ALTER TABLE projects
    ADD COLUMN auto_dependency_by_assignee BOOLEAN NOT NULL DEFAULT TRUE;
