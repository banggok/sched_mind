CREATE TABLE task_dependencies (
    id UUID PRIMARY KEY,
    blocking_task_id UUID NOT NULL REFERENCES wbs_nodes(id),
    blocked_task_id UUID NOT NULL REFERENCES wbs_nodes(id),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT task_dependencies_distinct_tasks CHECK (blocking_task_id <> blocked_task_id),
    CONSTRAINT task_dependencies_pair_unique UNIQUE (blocking_task_id, blocked_task_id)
);

CREATE INDEX task_dependencies_blocked_task_id_idx
    ON task_dependencies (blocked_task_id, blocking_task_id);

CREATE INDEX task_dependencies_blocking_task_id_idx
    ON task_dependencies (blocking_task_id, blocked_task_id);

CREATE INDEX wbs_nodes_name_key_prefix_idx
    ON wbs_nodes (name_key text_pattern_ops);
