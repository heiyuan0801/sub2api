-- Run manually after deploying the previous application. Export the two
-- new tables beforehand if delivery, click history and preferences must be retained.
-- Managed announcement content is kept as archived, with no legacy audience.
BEGIN;
UPDATE announcements
SET status = 'archived', updated_at = now()
WHERE id IN (SELECT announcement_id FROM announcement_publications);
DROP TABLE announcement_deliveries;
DROP TABLE announcement_publications;
DROP TABLE announcement_group_mutes;
DELETE FROM schema_migrations WHERE filename = '254_announcement_delivery.sql';
COMMIT;
