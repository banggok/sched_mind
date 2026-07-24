CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT roles_name_not_blank CHECK (LENGTH(BTRIM(name)) > 0),
    CONSTRAINT roles_name_max_length CHECK (CHAR_LENGTH(name) <= 100)
);

CREATE UNIQUE INDEX IF NOT EXISTS roles_name_ci_unique
    ON roles (LOWER(name));

CREATE TABLE IF NOT EXISTS team_members (
    id UUID PRIMARY KEY,
    role_id UUID NOT NULL,
    CONSTRAINT team_members_role_fk
        FOREIGN KEY (role_id)
        REFERENCES roles(id)
        ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS team_members_role_id_idx
    ON team_members(role_id);
