CREATE TABLE audit_logs (
    id UUID PRIMARY KEY,
    actor_user_id UUID REFERENCES users(id),
    action TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id UUID NOT NULL,
    ip_addr TEXT,
    user_agent TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX audit_logs_created_idx
    ON audit_logs (created_at DESC);

CREATE INDEX audit_logs_actor_idx
    ON audit_logs (actor_user_id, created_at DESC);

CREATE INDEX audit_logs_target_idx
    ON audit_logs (target_type, target_id, created_at DESC);
