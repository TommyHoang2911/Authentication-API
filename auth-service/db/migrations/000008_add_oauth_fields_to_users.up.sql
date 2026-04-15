-- +migrate Up
ALTER TABLE users ADD COLUMN oauth_provider VARCHAR(50);
ALTER TABLE users ADD COLUMN oauth_provider_id VARCHAR(255);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_oauth_identity
ON users(oauth_provider, oauth_provider_id)
WHERE oauth_provider IS NOT NULL AND oauth_provider_id IS NOT NULL;
