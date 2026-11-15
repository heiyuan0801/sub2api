-- Stop the new application and export the price history before applying this optional rollback.
-- Keep effective plan prices and every order snapshot. Retire restricted offers before an older
-- application takes over so it cannot accidentally sell them as public offers.
BEGIN;
UPDATE subscription_plans SET for_sale = false WHERE visibility_scope <> 'all';
DROP TABLE subscription_plan_price_changes;
ALTER TABLE subscription_plans
    DROP COLUMN display_tags,
    DROP COLUMN visibility_scope,
    DROP COLUMN visibility_user_ids,
    DROP COLUMN visibility_group_ids;
DELETE FROM schema_migrations WHERE filename = '253_subscription_plan_operations.sql';
COMMIT;
