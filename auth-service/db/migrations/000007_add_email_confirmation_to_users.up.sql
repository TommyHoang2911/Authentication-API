-- +migrate Up
ALTER TABLE users ADD COLUMN email_confirmed BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN confirmation_token VARCHAR(255);
ALTER TABLE users ADD COLUMN confirmation_token_expiry TIMESTAMP;