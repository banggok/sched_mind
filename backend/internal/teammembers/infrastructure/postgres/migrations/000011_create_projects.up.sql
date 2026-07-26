CREATE TABLE projects (
    id UUID PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    name_key VARCHAR(100) NOT NULL,
    status VARCHAR(10) NOT NULL DEFAULT 'open',
    start_date DATE,
    end_date DATE,
    auto_calculate_date BOOLEAN NOT NULL DEFAULT TRUE,
    auto_dependency_by_assignee BOOLEAN NOT NULL DEFAULT TRUE,
    priority INTEGER NOT NULL,
    closed_at TIMESTAMPTZ,
    locked_execution_snapshot TEXT,
    locked_commitment_snapshot TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT projects_name_not_blank CHECK (LENGTH(BTRIM(name)) > 0),
    CONSTRAINT projects_name_max_length CHECK (CHAR_LENGTH(name) <= 100),
    CONSTRAINT projects_name_key_unique UNIQUE (name_key),
    CONSTRAINT projects_status_valid CHECK (status IN ('open', 'locked', 'closed')),
    CONSTRAINT projects_priority_positive CHECK (priority > 0),
    CONSTRAINT projects_priority_unique UNIQUE (priority),
    CONSTRAINT projects_dates_valid CHECK (start_date IS NULL OR end_date IS NULL OR start_date <= end_date),
    CONSTRAINT projects_closed_at_valid CHECK ((status = 'closed' AND closed_at IS NOT NULL) OR (status <> 'closed' AND closed_at IS NULL))
);

CREATE INDEX projects_name_key_prefix_idx
    ON projects (name_key text_pattern_ops);

CREATE INDEX projects_active_order_idx
    ON projects (status, priority, start_date, end_date, id);

CREATE INDEX projects_closed_order_idx
    ON projects (status, closed_at DESC, id);
