-- stop using environment name as primary key for environments instead use numeric id
-- we have to also change all foreign key relationships

ALTER TABLE qa_environments ADD COLUMN id bigserial UNIQUE NOT NULL;

--drop constraint on foreign key tables
ALTER TABLE kubernetes_environments DROP CONSTRAINT kubernetes_environments_env_name_fkey;
ALTER TABLE helm_releases DROP CONSTRAINT helm_releases_env_name_fkey;

-- -- change primary key on qa_environments
ALTER TABLE qa_environments DROP CONSTRAINT qa_environments_pkey;
ALTER TABLE qa_environments ADD PRIMARY KEY (id);
ALTER TABLE qa_environments ADD UNIQUE (name);
CREATE UNIQUE INDEX IF NOT EXISTS idx_qa_environments_name ON qa_environments
    ( name );

-- add back in foreign keys on name so inserts with unknown envs fail and name changes cascade
ALTER TABLE helm_releases
    ADD CONSTRAINT helm_releases_env_name_fkey
        FOREIGN KEY (env_name) REFERENCES qa_environments (name)
        ON UPDATE CASCADE
        ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_helm_releases_envname ON helm_releases
    ( env_name );
ALTER TABLE kubernetes_environments
    ADD CONSTRAINT kubernetes_environments_env_name_fkey
        FOREIGN KEY (env_name) REFERENCES qa_environments (name)
        ON UPDATE CASCADE
        ON DELETE CASCADE;
CREATE UNIQUE INDEX IF NOT EXISTS idx_kubernetes_environments_envname ON kubernetes_environments
    ( env_name );

-- clean up event logs for the fk constraint
-- save dropped rows in new table so we can reverse this migration if needed
CREATE TABLE event_logs_orphans AS (
  SELECT * FROM event_logs WHERE env_name != '' AND (env_name NOT IN (SELECT name FROM qa_environments))
);
DELETE FROM event_logs WHERE env_name != '' AND (env_name NOT IN (SELECT name FROM qa_environments));
UPDATE event_logs SET env_name = null WHERE env_name = '';

-- add additional fk constraint on event logs so it cascades on update
ALTER TABLE event_logs
    ADD CONSTRAINT event_logs_env_name_fkey
        FOREIGN KEY (env_name) REFERENCES qa_environments (name)
        ON UPDATE CASCADE
        ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_event_logs_env_name ON event_logs
    ( env_name );
