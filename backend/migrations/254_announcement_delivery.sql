-- Reuse announcement content and existing read receipts. Managed announcements
-- take a recipient snapshot at publication; legacy announcements remain dynamic.
CREATE TABLE IF NOT EXISTS announcement_publications (
    announcement_id BIGINT PRIMARY KEY REFERENCES announcements(id) ON DELETE CASCADE,
    kind VARCHAR(24) NOT NULL CHECK (kind IN ('price_increase', 'price_decrease', 'new_plan', 'plan_unlisted', 'system', 'activity')),
    plan_id BIGINT REFERENCES subscription_plans(id) ON DELETE SET NULL,
    -- Keep the original group even after a plan is removed.
    group_id BIGINT,
    audience JSONB NOT NULL,
    delivery_status VARCHAR(16) NOT NULL DEFAULT 'draft' CHECK (delivery_status IN ('draft', 'queued', 'published', 'failed', 'withdrawn')),
    next_attempt_at TIMESTAMPTZ,
    published_at TIMESTAMPTZ,
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    consecutive_failures INTEGER NOT NULL DEFAULT 0 CHECK (consecutive_failures >= 0),
    last_error VARCHAR(120) NOT NULL DEFAULT '',
    published_by BIGINT REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_announcement_publications_due
    ON announcement_publications(next_attempt_at, announcement_id) WHERE delivery_status = 'queued';

CREATE TABLE IF NOT EXISTS announcement_deliveries (
    announcement_id BIGINT NOT NULL REFERENCES announcement_publications(announcement_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    delivered_at TIMESTAMPTZ NOT NULL,
    click_count BIGINT NOT NULL DEFAULT 0 CHECK (click_count >= 0),
    PRIMARY KEY (announcement_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_announcement_deliveries_user
    ON announcement_deliveries(user_id, announcement_id DESC);

CREATE TABLE IF NOT EXISTS announcement_group_mutes (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, group_id)
);
