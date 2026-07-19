CREATE TABLE IF NOT EXISTS share_links (
    id UUID PRIMARY KEY,
    document_id UUID NOT NULL REFERENCES documents(id),
    token_hash TEXT NOT NULL UNIQUE,
    permission TEXT NOT NULL,
    expires_at TIMESTAMPTZ,
    disabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS share_links_document_created_idx
    ON share_links (document_id, created_at DESC);
