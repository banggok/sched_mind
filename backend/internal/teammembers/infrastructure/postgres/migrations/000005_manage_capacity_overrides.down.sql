DROP INDEX IF EXISTS capacity_overrides_member_period_idx;
ALTER TABLE capacity_overrides
    DROP CONSTRAINT IF EXISTS capacity_overrides_capacity_increment,
    DROP CONSTRAINT IF EXISTS capacity_overrides_capacity_range,
    DROP CONSTRAINT IF EXISTS capacity_overrides_date_range,
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS created_at,
    DROP COLUMN IF EXISTS capacity,
    DROP COLUMN IF EXISTS end_date,
    DROP COLUMN IF EXISTS start_date;
