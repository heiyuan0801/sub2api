ALTER TABLE groups
    ADD COLUMN custom_display_tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD CONSTRAINT groups_custom_display_tags_valid CHECK (
        CASE WHEN jsonb_typeof(custom_display_tags) = 'array'
            THEN jsonb_array_length(custom_display_tags) <= 5
            ELSE false
        END
    );

COMMENT ON COLUMN groups.custom_display_tags IS 'Custom presentation badges; do not affect group access or billing.';
