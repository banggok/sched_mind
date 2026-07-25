CREATE INDEX IF NOT EXISTS team_members_name_prefix_search_idx
    ON team_members (LOWER(name) text_pattern_ops);
