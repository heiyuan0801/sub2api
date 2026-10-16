-- Keep a model's first catalog appearance independent of price row replacement.
-- Historical entries have no reliable per-model timestamp. Seed their identities
-- with NULL so editing an old price cannot advertise that model as newly added.
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '10min';

CREATE TABLE IF NOT EXISTS channel_model_listings (
    platform        TEXT        NOT NULL,
    model_name      TEXT        NOT NULL,
    first_listed_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (platform, model_name)
);

COMMENT ON TABLE channel_model_listings IS '渠道模型首次加入目录的历史；不随渠道或定价删除';
COMMENT ON COLUMN channel_model_listings.model_name IS '与 SupportedModels 一致的小写模型身份';
COMMENT ON COLUMN channel_model_listings.first_listed_at IS '首次加入时间；NULL 表示迁移前已存在且历史时间未知';

-- SupportedModels is the union of concrete price names and exact mapping keys.
-- Wildcard mappings only expand concrete price names already covered here.
WITH configured_models AS (
    SELECT pricing.platform, model.name
    FROM channel_model_pricing AS pricing
    CROSS JOIN LATERAL jsonb_array_elements_text(
        CASE WHEN jsonb_typeof(pricing.models) = 'array' THEN pricing.models ELSE '[]'::jsonb END
    ) AS model(name)
    UNION
    SELECT platform.key, model.key
    FROM channels AS channel
    CROSS JOIN LATERAL jsonb_each(
        CASE WHEN jsonb_typeof(channel.model_mapping) = 'object' THEN channel.model_mapping ELSE '{}'::jsonb END
    ) AS platform(key, mappings)
    CROSS JOIN LATERAL jsonb_each(
        CASE WHEN jsonb_typeof(platform.mappings) = 'object' THEN platform.mappings ELSE '{}'::jsonb END
    ) AS model(key, target)
)
INSERT INTO channel_model_listings (platform, model_name, first_listed_at)
SELECT DISTINCT platform, lower(name), NULL::TIMESTAMPTZ
FROM configured_models
WHERE name <> '' AND right(name, 1) <> '*'
ON CONFLICT (platform, model_name) DO NOTHING;
