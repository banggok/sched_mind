ALTER TABLE wbs_nodes
    DROP CONSTRAINT IF EXISTS wbs_lag_days_non_negative,
    DROP COLUMN IF EXISTS lag_days;
