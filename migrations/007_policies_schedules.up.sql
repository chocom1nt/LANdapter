-- Declarative software group policies and durable interval-based schedules.
CREATE TABLE IF NOT EXISTS group_policies (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 120),
    description TEXT NOT NULL DEFAULT '',
    os_filter TEXT NOT NULL DEFAULT '',
    files JSONB NOT NULL,
    mode TEXT NOT NULL DEFAULT 'quiet' CHECK (mode IN ('quiet', 'normal', 'interactive')),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_group_policies_enabled ON group_policies(enabled);

CREATE TABLE IF NOT EXISTS schedules (
    id UUID PRIMARY KEY,
    policy_id UUID NOT NULL REFERENCES group_policies(id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 120),
    interval_minutes INTEGER NOT NULL CHECK (interval_minutes BETWEEN 5 AND 43200),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    last_run TIMESTAMPTZ,
    next_run TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_schedules_due ON schedules(enabled, next_run);
