DROP TABLE IF EXISTS wbs_nodes;

CREATE TABLE executable_leaves (
    id UUID PRIMARY KEY,
    assignee_id UUID NOT NULL REFERENCES team_members(id) ON DELETE RESTRICT
);
CREATE INDEX executable_leaves_assignee_id_idx ON executable_leaves(assignee_id);
