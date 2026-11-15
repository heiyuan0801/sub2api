-- A direct balance group target is separate from the historical group snapshot
-- of a subscription plan. Keep both targets readable after their source is removed.
ALTER TABLE announcement_publications
    ADD COLUMN standard_group_id BIGINT,
    DROP CONSTRAINT announcement_publications_kind_check,
    ADD CONSTRAINT announcement_publications_kind_check CHECK (kind IN (
        'price_increase', 'price_decrease', 'new_plan', 'plan_unlisted',
        'new_group', 'group_unlisted', 'system', 'activity'
    )),
    ADD CONSTRAINT announcement_publications_standard_group_check CHECK (
        (standard_group_id IS NULL AND kind NOT IN ('new_group', 'group_unlisted'))
        OR (standard_group_id IS NOT NULL AND standard_group_id > 0 AND plan_id IS NULL AND group_id IS NULL
            AND kind IN ('price_increase', 'price_decrease', 'new_group', 'group_unlisted'))
    );
