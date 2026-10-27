-- Aggregate keys retain a stable client identity and reference ordinary keys owned by the same user/group.
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS is_aggregate boolean NOT NULL DEFAULT false;
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS aggregate_key_ids jsonb;
