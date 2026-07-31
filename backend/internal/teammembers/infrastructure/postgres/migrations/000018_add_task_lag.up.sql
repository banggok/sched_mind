ALTER TABLE wbs_nodes
    ADD COLUMN lag_days INTEGER NOT NULL DEFAULT 0,
    ADD CONSTRAINT wbs_lag_days_non_negative CHECK (lag_days >= 0);
