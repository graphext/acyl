DROP INDEX IF EXISTS idx_qa_environments_name;
ALTER TABLE qa_environments DROP CONSTRAINT qa_environments_pkey;

DROP INDEX IF EXISTS idx_event_logs_env_name;

ALTER TABLE helm_releases
    DROP CONSTRAINT helm_releases_env_name_fkey;
ALTER TABLE kubernetes_environments
    DROP CONSTRAINT kubernetes_environments_env_name_fkey;
ALTER TABLE event_logs
    DROP CONSTRAINT event_logs_env_name_fkey;

UPDATE event_logs SET env_name = '' WHERE env_name = null;
INSERT INTO event_logs (SELECT * FROM event_logs_orphans);

DROP TABLE event_logs_orphans;

DROP INDEX IF EXISTS idx_helm_releases_envname;
ALTER TABLE helm_releases
    ADD CONSTRAINT helm_releases_env_name_fkey FOREIGN KEY (env_name) REFERENCES qa_environments (name);

DROP INDEX IF EXISTS idx_kubernetes_environments_envname;
ALTER TABLE kubernetes_environments
    ADD CONSTRAINT kubernetes_environments_env_name_fkey FOREIGN KEY (env_name) REFERENCES qa_environments (name);

ALTER TABLE qa_environments DROP COLUMN id;
