-- Keep the previous column for rollback compatibility. Never transfer member
-- credentials or quota: existing aggregates become a one-group aggregate.
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS aggregate_group_ids jsonb;
UPDATE api_keys
SET aggregate_group_ids = jsonb_build_array(group_id), group_id = NULL
WHERE is_aggregate = TRUE AND aggregate_group_ids IS NULL AND group_id IS NOT NULL;
