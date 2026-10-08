CREATE TABLE console_sessions (
    session_hash TEXT PRIMARY KEY CHECK (session_hash ~ '^[0-9a-f]{64}$'),
    credential_hash TEXT NOT NULL CHECK (credential_hash ~ '^[0-9a-f]{64}$'),
    subject TEXT NOT NULL CHECK (length(subject) BETWEEN 1 AND 128),
    role TEXT NOT NULL CHECK (role IN ('analyst', 'administrator')),
    created_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    CHECK (expires_at > created_at AND expires_at <= created_at + INTERVAL '8 hours'),
    CHECK (last_seen_at >= created_at)
);
CREATE INDEX console_sessions_subject_idx ON console_sessions (subject);
