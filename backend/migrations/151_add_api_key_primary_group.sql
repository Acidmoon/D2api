-- Add primary group support for API keys.
-- Existing group_id remains the secondary group so legacy keys keep their current group as fallback.
ALTER TABLE api_keys
  ADD COLUMN IF NOT EXISTS primary_group_id BIGINT REFERENCES groups(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_api_keys_primary_group_id ON api_keys(primary_group_id);
