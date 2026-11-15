-- Stop the new application before removing the custom display metadata.
BEGIN;
ALTER TABLE groups DROP COLUMN custom_display_tags;
DELETE FROM schema_migrations WHERE filename = '258_group_custom_display_tags.sql';
COMMIT;
