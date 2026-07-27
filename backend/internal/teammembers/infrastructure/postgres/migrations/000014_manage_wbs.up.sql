DROP TABLE IF EXISTS executable_leaves;

CREATE TABLE wbs_nodes (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    parent_id UUID NULL REFERENCES wbs_nodes(id) ON DELETE RESTRICT,
    parent_key VARCHAR(36) NOT NULL DEFAULT '',
    name VARCHAR(100) NOT NULL,
    name_key VARCHAR(100) NOT NULL,
    position INTEGER NOT NULL CHECK (position > 0),
    role_id UUID NULL REFERENCES roles(id) ON DELETE RESTRICT,
    assignee_id UUID NULL REFERENCES team_members(id) ON DELETE RESTRICT,
    effort_minutes INTEGER NULL CHECK (effort_minutes >= 30 AND effort_minutes % 30 = 0),
    execution_start DATE NULL,
    execution_end DATE NULL,
    commitment_start DATE NULL,
    commitment_end DATE NULL,
    actual_end DATE NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT wbs_execution_pair CHECK ((execution_start IS NULL) = (execution_end IS NULL)),
    CONSTRAINT wbs_commitment_pair CHECK ((commitment_start IS NULL) = (commitment_end IS NULL)),
    CONSTRAINT wbs_execution_order CHECK (execution_end IS NULL OR execution_end >= execution_start),
    CONSTRAINT wbs_commitment_order CHECK (commitment_end IS NULL OR commitment_end >= commitment_start),
    CONSTRAINT wbs_parent_key_consistent CHECK (parent_key = COALESCE(parent_id::text, '')),
    CONSTRAINT wbs_sibling_name_unique UNIQUE (project_id, parent_key, name_key),
    CONSTRAINT wbs_sibling_position_unique UNIQUE (project_id, parent_key, position)
);

CREATE INDEX wbs_nodes_project_tree_idx ON wbs_nodes(project_id, parent_key, position, id);
CREATE INDEX wbs_nodes_parent_id_idx ON wbs_nodes(parent_id);
CREATE INDEX wbs_nodes_assignee_id_idx ON wbs_nodes(assignee_id);
