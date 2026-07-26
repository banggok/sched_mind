ALTER TABLE projects
    ADD COLUMN automatic_scheduling BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN project_buffer INTEGER NOT NULL DEFAULT 20,
    ADD CONSTRAINT projects_project_buffer_range CHECK (project_buffer BETWEEN 0 AND 100);
