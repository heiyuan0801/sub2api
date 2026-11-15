CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_registration_report
    ON users (created_at DESC, id DESC) WHERE deleted_at IS NULL;
