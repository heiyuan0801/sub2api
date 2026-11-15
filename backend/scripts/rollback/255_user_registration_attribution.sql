-- Roll back application binaries first. Export registration attribution data
-- before this rollback if it is needed later. Users and balances are preserved.
BEGIN;
DROP INDEX IF EXISTS idx_users_registration_report;
DROP TABLE IF EXISTS user_registration_attributions;
DELETE FROM schema_migrations WHERE filename IN (
    '255_user_registration_attribution.sql',
    '255_user_registration_attribution_index_notx.sql'
);
COMMIT;
