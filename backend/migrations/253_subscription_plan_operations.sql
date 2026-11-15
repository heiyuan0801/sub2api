ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS display_tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS visibility_scope VARCHAR(12) NOT NULL DEFAULT 'all',
    ADD COLUMN IF NOT EXISTS visibility_user_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS visibility_group_ids JSONB NOT NULL DEFAULT '[]'::jsonb;

CREATE TABLE IF NOT EXISTS subscription_plan_price_changes (
    id BIGSERIAL PRIMARY KEY,
    plan_id BIGINT NOT NULL,
    plan_name VARCHAR(100) NOT NULL,
    group_id BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT '',
    old_price DECIMAL(20,2) NOT NULL CHECK (old_price >= 0),
    new_price DECIMAL(20,2) NOT NULL CHECK (new_price > 0),
    reason VARCHAR(1000) NOT NULL DEFAULT '',
    actor_id BIGINT NOT NULL,
    cancelled_by BIGINT,
    status VARCHAR(16) NOT NULL CHECK (status IN ('pending', 'applied', 'cancelled', 'conflict')),
    effective_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_plan_price_single_pending
    ON subscription_plan_price_changes(plan_id) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_plan_price_due
    ON subscription_plan_price_changes(effective_at, plan_id) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_plan_price_history
    ON subscription_plan_price_changes(plan_id, id DESC);
