-- metadata table and schema_version row are bootstrapped by ledger.migrate()
-- before any migration runs. They are not re-created here.

CREATE TABLE IF NOT EXISTS upload_state (
    skill_name   TEXT NOT NULL,
    target       TEXT NOT NULL,  -- 'claudeai' | 'claude_code'
    version      TEXT NOT NULL,  -- git tag or short SHA
    content_hash TEXT NOT NULL,  -- SHA256 of zip (recorded for future use)
    uploaded_at  TEXT NOT NULL,  -- ISO 8601
    PRIMARY KEY (skill_name, target)
);

CREATE TABLE IF NOT EXISTS events (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    otel_event_id TEXT UNIQUE,
    timestamp     TEXT NOT NULL,
    skill_name    TEXT NOT NULL,
    event_type    TEXT NOT NULL,
    payload       TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_events_skill     ON events(skill_name);
CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
CREATE INDEX IF NOT EXISTS idx_events_type      ON events(event_type);
