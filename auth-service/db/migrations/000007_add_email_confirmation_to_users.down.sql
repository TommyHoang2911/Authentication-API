-- +migrate Down
ALTER TABLE users DROP COLUMN email_confirmed;
ALTER TABLE users DROP COLUMN confirmation_token;
ALTER TABLE users DROP COLUMN confirmation_token_expiry;