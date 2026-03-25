-- +migrate Down
DROP INDEX IF EXISTS idx_users_oauth_identity;

ALTER TABLE users DROP COLUMN IF EXISTS oauth_provider_id;
ALTER TABLE users DROP COLUMN IF EXISTS oauth_provider;
