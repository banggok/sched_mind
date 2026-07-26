ALTER TABLE capacity_overrides
    ADD COLUMN start_date DATE NOT NULL DEFAULT CURRENT_DATE,
    ADD COLUMN end_date DATE NOT NULL DEFAULT CURRENT_DATE,
    ADD COLUMN capacity NUMERIC(4,1) NOT NULL DEFAULT 0,
    ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD CONSTRAINT capacity_overrides_date_range
        CHECK (end_date >= start_date),
    ADD CONSTRAINT capacity_overrides_capacity_range
        CHECK (capacity >= 0 AND capacity <= 24),
    ADD CONSTRAINT capacity_overrides_capacity_increment
        CHECK (MOD(capacity * 2, 1) = 0);

ALTER TABLE capacity_overrides
    ALTER COLUMN start_date DROP DEFAULT,
    ALTER COLUMN end_date DROP DEFAULT,
    ALTER COLUMN capacity DROP DEFAULT,
    ALTER COLUMN created_at DROP DEFAULT,
    ALTER COLUMN updated_at DROP DEFAULT;

CREATE INDEX capacity_overrides_member_period_idx
    ON capacity_overrides(team_member_id, start_date, end_date);
