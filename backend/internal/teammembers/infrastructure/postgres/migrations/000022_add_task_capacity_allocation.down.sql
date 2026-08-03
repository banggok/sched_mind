DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM wbs_nodes
        WHERE capacity_allocation_percentage <> 100
    ) THEN
        RAISE EXCEPTION 'cannot roll back task capacity allocation while custom values exist';
    END IF;
END $$;

ALTER TABLE wbs_nodes
    DROP CONSTRAINT IF EXISTS wbs_nodes_capacity_allocation_percentage_chk,
    DROP COLUMN IF EXISTS capacity_allocation_percentage;
