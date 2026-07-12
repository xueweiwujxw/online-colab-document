CREATE TABLE onlyoffice_saves (
    document_id UUID NOT NULL REFERENCES documents(id),
    document_key TEXT NOT NULL,
    version_id UUID REFERENCES document_versions(id),
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (document_id, document_key)
);
