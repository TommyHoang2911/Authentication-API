-- Revert: Make user_id non-nullable again
ALTER TABLE auth_codes ALTER COLUMN user_id SET NOT NULL;
