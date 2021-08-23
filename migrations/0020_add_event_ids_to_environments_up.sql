ALTER TABLE qa_environments ADD COLUMN event_ids uuid[] DEFAULT '{}';

UPDATE qa_environments
SET event_ids = array(
    SELECT id FROM event_logs WHERE qa_environments.name = event_logs.env_name
);