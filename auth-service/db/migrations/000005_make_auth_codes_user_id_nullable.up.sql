-- Make user_id nullable in auth_codes table
-- This allows temporary auth codes to be created without a user initially
ALTER TABLE auth_codes ALTER COLUMN user_id DROP NOT NULL;
