ALTER TABLE helm_releases ADD COLUMN tiller_namespace VARCHAR(512) DEFAULT 'kube-system';

ALTER TABLE kubernetes_environments ADD COLUMN tiller_addr TEXT;