CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    run_at TIMESTAMPTZ NOT NULL,
    last_error TEXT,
    locked_by TEXT,
    locked_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT tasks_status_check CHECK (
        status IN ('pending', 'running', 'failed', 'completed', 'dead')
    ),
    CONSTRAINT tasks_attempts_not_negative CHECK (attempts >= 0),
    CONSTRAINT tasks_max_attempts_positive CHECK (max_attempts > 0)
);

CREATE INDEX IF NOT EXISTS idx_tasks_ready
    ON tasks (run_at, created_at)
    WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_tasks_lease_expired
    ON tasks (locked_until)
    WHERE status = 'running';

CREATE INDEX IF NOT EXISTS idx_tasks_status
    ON tasks (status);
