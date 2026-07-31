ALTER TABLE task_dependencies
    ADD COLUMN manual_owned BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN automatic_owned BOOLEAN NOT NULL DEFAULT FALSE,
    ADD CONSTRAINT task_dependencies_has_owner CHECK (manual_owned OR automatic_owned);
