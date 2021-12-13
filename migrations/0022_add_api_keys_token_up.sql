CREATE EXTENSION IF NOT EXISTS pgcrypto;

ALTER TABLE api_keys ADD COLUMN token UUID NOT NULL DEFAULT gen_random_uuid();

CREATE INDEX IF NOT EXISTS token_idx ON api_keys (token);
