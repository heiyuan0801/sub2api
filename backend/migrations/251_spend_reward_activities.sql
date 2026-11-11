CREATE TABLE IF NOT EXISTS reward_activity_config (
    id SMALLINT PRIMARY KEY CHECK (id = 1),
    config JSONB NOT NULL
);

CREATE TABLE IF NOT EXISTS reward_activity_spend (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    spend_day DATE NOT NULL,
    amount NUMERIC(24,8) NOT NULL CHECK (amount >= 0)
);

CREATE TABLE IF NOT EXISTS reward_activity_claims (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    request_id UUID NOT NULL,
    activity_day DATE NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('check_in', 'wheel')),
    reward_cents BIGINT NOT NULL CHECK (reward_cents >= 0),
    consumed_cents BIGINT NOT NULL CHECK (consumed_cents >= 0),
    prize_index INTEGER NOT NULL DEFAULT -1,
    prize_name TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (user_id, request_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_reward_activity_check_in
    ON reward_activity_claims (user_id, activity_day) WHERE kind = 'check_in';
CREATE INDEX IF NOT EXISTS idx_reward_activity_claims_user_day
    ON reward_activity_claims (user_id, activity_day);
CREATE INDEX IF NOT EXISTS idx_reward_activity_claims_user_history
    ON reward_activity_claims (user_id, id DESC);
