ALTER TABLE kubernetes_environments DROP COLUMN IF EXISTS tiller_addr;

ALTER TABLE helm_releases DROP COLUMN IF EXISTS tiller_namespace;