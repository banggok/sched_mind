ALTER TABLE projects
    DROP CONSTRAINT projects_project_buffer_range,
    DROP COLUMN project_buffer,
    DROP COLUMN automatic_scheduling;
