CREATE TABLE markdown_snapshots (
    id UUID PRIMARY KEY,
    document_id UUID NOT NULL REFERENCES documents(id),
    version_no BIGINT NOT NULL,
    content TEXT NOT NULL,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE markdown_updates (
    id UUID PRIMARY KEY,
    document_id UUID NOT NULL REFERENCES documents(id),
    update_seq BIGINT NOT NULL,
    update_data BYTEA NOT NULL,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX markdown_snapshots_document_version_idx
    ON markdown_snapshots (document_id, version_no DESC);

CREATE UNIQUE INDEX markdown_updates_document_seq_unique
    ON markdown_updates (document_id, update_seq);
