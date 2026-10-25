-- Additive HA operations state. No user balances or existing usage rows are changed.
CREATE TABLE IF NOT EXISTS cluster_nodes (
    node_id VARCHAR(64) PRIMARY KEY,
    instance_id VARCHAR(64) NOT NULL,
    role VARCHAR(16) NOT NULL,
    version VARCHAR(128) NOT NULL DEFAULT '',
    heartbeat_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    snapshot JSONB NOT NULL DEFAULT '{}'
);
CREATE TABLE IF NOT EXISTS cluster_node_controls (
    node_id VARCHAR(64) PRIMARY KEY REFERENCES cluster_nodes(node_id),
    maintenance BOOLEAN NOT NULL DEFAULT FALSE,
    max_concurrency INTEGER NOT NULL DEFAULT 1000 CHECK (max_concurrency BETWEEN 1 AND 20000),
    weight INTEGER NOT NULL DEFAULT 100 CHECK (weight BETWEEN 1 AND 256),
    revision BIGINT NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by BIGINT,
    reason VARCHAR(500) NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS cluster_events (
    id BIGSERIAL PRIMARY KEY,
    node_id VARCHAR(64) NOT NULL DEFAULT '',
    kind VARCHAR(64) NOT NULL,
    severity VARCHAR(16) NOT NULL DEFAULT 'info',
    message VARCHAR(1000) NOT NULL,
    actor_id BIGINT,
    notification_done BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS cluster_events_recent ON cluster_events(created_at DESC, id DESC);
CREATE TABLE IF NOT EXISTS cluster_alerts (
    alert_key VARCHAR(160) PRIMARY KEY,
    node_id VARCHAR(64) NOT NULL DEFAULT '',
    severity VARCHAR(16) NOT NULL,
    message VARCHAR(1000) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    raised_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS cluster_request_journal (
    attempt_id UUID PRIMARY KEY,
    request_id VARCHAR(256) NOT NULL,
    node_id VARCHAR(64) NOT NULL,
    instance_id VARCHAR(64) NOT NULL,
    user_id BIGINT NOT NULL,
    api_key_id BIGINT NOT NULL,
    route VARCHAR(256) NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    http_status INTEGER,
    state VARCHAR(32) NOT NULL DEFAULT 'running',
    review_note VARCHAR(1000) NOT NULL DEFAULT '',
    reviewed_by BIGINT,
    reviewed_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS cluster_requests_pending ON cluster_request_journal(started_at) WHERE state IN ('running','needs_review');
CREATE INDEX IF NOT EXISTS cluster_requests_billing ON cluster_request_journal(request_id, api_key_id);
CREATE INDEX IF NOT EXISTS cluster_requests_recent ON cluster_request_journal(started_at DESC, attempt_id);
CREATE TABLE IF NOT EXISTS cluster_billing_evidence (
    request_id VARCHAR(256) NOT NULL,
    api_key_id BIGINT NOT NULL,
    attempt_id UUID,
    user_id BIGINT NOT NULL,
    fingerprint VARCHAR(128) NOT NULL,
    expected_cost NUMERIC(20,10) NOT NULL,
    usage_snapshot JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY(request_id, api_key_id)
);
CREATE INDEX IF NOT EXISTS cluster_billing_evidence_attempt ON cluster_billing_evidence(attempt_id);
CREATE TABLE IF NOT EXISTS cluster_image_tasks (
    task_id VARCHAR(128) PRIMARY KEY,
    node_id VARCHAR(64) NOT NULL,
    instance_id VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL,
    payload JSONB NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS cluster_image_tasks_recovery ON cluster_image_tasks(status, updated_at);
