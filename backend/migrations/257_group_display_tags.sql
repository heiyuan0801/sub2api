ALTER TABLE groups
    ADD COLUMN display_tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD CONSTRAINT groups_display_tags_valid CHECK (
        jsonb_typeof(display_tags) = 'array'
        AND display_tags <@ '["recommended", "popular", "new"]'::jsonb
    );

COMMENT ON COLUMN groups.display_tags IS 'Presentation tags; do not affect group access or billing.';
