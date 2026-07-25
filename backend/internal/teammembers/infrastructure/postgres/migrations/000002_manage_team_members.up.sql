ALTER TABLE team_members
    ADD COLUMN name VARCHAR(100) NOT NULL DEFAULT 'Unknown',
    ADD COLUMN daily_capacity NUMERIC(4,1) NOT NULL DEFAULT 8,
    ADD COLUMN buffer_percentage NUMERIC(5,2) NOT NULL DEFAULT 20,
    ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD CONSTRAINT team_members_name_not_blank
        CHECK (LENGTH(BTRIM(name)) > 0),
    ADD CONSTRAINT team_members_name_max_length
        CHECK (CHAR_LENGTH(name) <= 100),
    ADD CONSTRAINT team_members_daily_capacity_range
        CHECK (daily_capacity > 0 AND daily_capacity <= 24),
    ADD CONSTRAINT team_members_daily_capacity_increment
        CHECK (MOD(daily_capacity * 2, 1) = 0),
    ADD CONSTRAINT team_members_buffer_range
        CHECK (buffer_percentage >= 0 AND buffer_percentage < 100);

ALTER TABLE team_members
    ALTER COLUMN name DROP DEFAULT,
    ALTER COLUMN daily_capacity DROP DEFAULT,
    ALTER COLUMN buffer_percentage DROP DEFAULT,
    ALTER COLUMN created_at DROP DEFAULT,
    ALTER COLUMN updated_at DROP DEFAULT;

CREATE INDEX team_members_name_ci_id_idx
    ON team_members (LOWER(name), id);

CREATE TABLE executable_leaves (
    id UUID PRIMARY KEY,
    assignee_id UUID,
    CONSTRAINT executable_leaves_assignee_fk
        FOREIGN KEY (assignee_id)
        REFERENCES team_members(id)
        ON DELETE RESTRICT
);

CREATE INDEX executable_leaves_assignee_id_idx
    ON executable_leaves(assignee_id);

CREATE TABLE capacity_overrides (
    id UUID PRIMARY KEY,
    team_member_id UUID NOT NULL,
    CONSTRAINT capacity_overrides_team_member_fk
        FOREIGN KEY (team_member_id)
        REFERENCES team_members(id)
        ON DELETE RESTRICT
);

CREATE INDEX capacity_overrides_team_member_id_idx
    ON capacity_overrides(team_member_id);
