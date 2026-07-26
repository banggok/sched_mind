ALTER TABLE capacity_overrides
    ADD COLUMN description VARCHAR(100) NOT NULL DEFAULT 'Capacity override';

ALTER TABLE capacity_overrides
    ALTER COLUMN description DROP DEFAULT;
