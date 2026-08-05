CREATE TABLE sprints (
    id UUID PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    name_key VARCHAR(200) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    status VARCHAR(10) NOT NULL DEFAULT 'planned',
    version BIGINT NOT NULL DEFAULT 1,
    started_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT sprints_name_required CHECK (BTRIM(name) <> ''),
    CONSTRAINT sprints_name_unique UNIQUE (name_key),
    CONSTRAINT sprints_date_order CHECK (end_date >= start_date),
    CONSTRAINT sprints_status_valid CHECK (status IN ('planned', 'started')),
    CONSTRAINT sprints_version_positive CHECK (version > 0),
    CONSTRAINT sprints_started_at_consistent CHECK (
        (status = 'planned' AND started_at IS NULL)
        OR (status = 'started' AND started_at IS NOT NULL)
    )
);

CREATE INDEX sprints_list_order_idx
    ON sprints (start_date DESC, end_date DESC, name_key, id);

CREATE TABLE sprint_members (
    sprint_id UUID NOT NULL REFERENCES sprints(id) ON DELETE CASCADE,
    member_id UUID NOT NULL REFERENCES team_members(id),
    PRIMARY KEY (sprint_id, member_id)
);

CREATE INDEX sprint_members_member_sprint_idx
    ON sprint_members (member_id, sprint_id);

CREATE TABLE sprint_tasks (
    sprint_id UUID NOT NULL REFERENCES sprints(id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES wbs_nodes(id) ON DELETE CASCADE,
    PRIMARY KEY (sprint_id, task_id)
);

CREATE INDEX sprint_tasks_task_sprint_idx
    ON sprint_tasks (task_id, sprint_id);
