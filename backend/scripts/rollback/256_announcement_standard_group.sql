-- Stop new application workers before running this script. Export matching
-- publication/delivery rows beforehand if standard group delivery history is needed.
BEGIN;
UPDATE announcements SET status = 'archived', updated_at = now()
WHERE id IN (SELECT announcement_id FROM announcement_publications WHERE standard_group_id IS NOT NULL);
DELETE FROM announcement_publications WHERE standard_group_id IS NOT NULL;
ALTER TABLE announcement_publications
    DROP CONSTRAINT announcement_publications_standard_group_check,
    DROP CONSTRAINT announcement_publications_kind_check,
    DROP COLUMN standard_group_id,
    ADD CONSTRAINT announcement_publications_kind_check CHECK (kind IN (
        'price_increase', 'price_decrease', 'new_plan', 'plan_unlisted', 'system', 'activity'
    ));
DELETE FROM schema_migrations WHERE filename = '256_announcement_standard_group.sql';
COMMIT;
