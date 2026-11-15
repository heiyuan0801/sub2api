-- Marketing attribution is immutable and separate from users.signup_source,
-- which records the authentication provider and controls signup grants.
CREATE TABLE IF NOT EXISTS user_registration_attributions (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    selected_source VARCHAR(20) NOT NULL DEFAULT '' CHECK (selected_source IN (
        '', 'google', 'bing', 'friend', 'telegram', 'discord', 'github',
        'youtube', 'reddit', 'x', 'blog_forum', 'other'
    )),
    source_note VARCHAR(240) NOT NULL DEFAULT '' CHECK (source_note = '' OR selected_source = 'other'),
    utm_source VARCHAR(120) NOT NULL DEFAULT '',
    utm_medium VARCHAR(120) NOT NULL DEFAULT '',
    utm_campaign VARCHAR(120) NOT NULL DEFAULT '',
    utm_content VARCHAR(120) NOT NULL DEFAULT '',
    utm_term VARCHAR(120) NOT NULL DEFAULT ''
);
