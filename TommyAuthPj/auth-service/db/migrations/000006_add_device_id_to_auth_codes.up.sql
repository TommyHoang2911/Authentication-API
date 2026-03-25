-- Add device_id column to auth_codes table
ALTER TABLE auth_codes ADD COLUMN device_id TEXT;