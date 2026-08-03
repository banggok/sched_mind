ALTER TABLE wbs_nodes
    ADD COLUMN capacity_allocation_percentage INTEGER;

UPDATE wbs_nodes
SET capacity_allocation_percentage = 100
WHERE capacity_allocation_percentage IS NULL;

ALTER TABLE wbs_nodes
    ALTER COLUMN capacity_allocation_percentage SET DEFAULT 100,
    ALTER COLUMN capacity_allocation_percentage SET NOT NULL,
    ADD CONSTRAINT wbs_nodes_capacity_allocation_percentage_chk
        CHECK (capacity_allocation_percentage BETWEEN 1 AND 100);
