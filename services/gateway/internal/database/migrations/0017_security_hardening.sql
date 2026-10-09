CREATE TABLE access_audit (
    id BIGSERIAL PRIMARY KEY,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    trace_id TEXT NOT NULL CHECK (trace_id ~ '^[a-f0-9]{32}$'),
    method TEXT NOT NULL CHECK (length(method) <= 16),
    route TEXT NOT NULL CHECK (length(route) <= 160),
    subject TEXT NOT NULL CHECK (length(subject) <= 128),
    role TEXT NOT NULL CHECK (role IN ('', 'analyst', 'administrator', 'collector')),
    outcome TEXT NOT NULL CHECK (length(outcome) <= 40),
    status INTEGER NOT NULL CHECK (status BETWEEN 0 AND 599)
);
CREATE INDEX access_audit_retention_idx ON access_audit (occurred_at, id);
CREATE TABLE collector_nonces (
    subject TEXT NOT NULL,
    nonce TEXT NOT NULL CHECK (nonce ~ '^[a-f0-9]{32}$'),
    body_sha256 TEXT NOT NULL CHECK (body_sha256 ~ '^[a-f0-9]{64}$'),
    expires_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (subject, nonce)
);
CREATE INDEX collector_nonces_expiry_idx ON collector_nonces (expires_at);
