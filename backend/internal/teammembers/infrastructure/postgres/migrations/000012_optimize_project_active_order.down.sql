DROP INDEX IF EXISTS projects_active_priority_idx;

CREATE INDEX projects_active_order_idx
    ON projects (status, priority, start_date, end_date, id);
