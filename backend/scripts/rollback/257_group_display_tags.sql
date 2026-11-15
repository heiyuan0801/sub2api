-- Stop the new application before removing the display metadata.
BEGIN;
ALTER TABLE groups DROP COLUMN display_tags;
DELETE FROM schema_migrations WHERE filename = '257_group_display_tags.sql';
COMMIT;
