CREATE TABLE env_created_events (
    id SERIAL PRIMARY KEY,
    environment_id INTEGER REFERENCES environments (id) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    name VARCHAR(512) UNIQUE NOT NULL,
    kube_namespace VARCHAR(512) NOT NULL
);

CREATE TABLE env_destroyed_events (
    id SERIAL PRIMARY KEY,
    environment_id INTEGER REFERENCES environments (id) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);

CREATE TABLE helm_releases (
    id SERIAL PRIMARY KEY,
    environment_id INTEGER REFERENCES environments (id) NOT NULL,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    release_name VARCHAR(512) NOT NULL,
    tiller_namespace VARCHAR(512) DEFAULT 'kube-system'
);

