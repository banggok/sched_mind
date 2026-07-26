DROP INDEX projects_active_order_idx;

CREATE INDEX projects_active_priority_idx
    ON projects (priority, start_date, end_date, id)
    WHERE status IN ('open', 'locked');
