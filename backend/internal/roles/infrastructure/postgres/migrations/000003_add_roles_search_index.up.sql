CREATE INDEX IF NOT EXISTS roles_name_prefix_search_idx
    ON roles (LOWER(name) text_pattern_ops);
