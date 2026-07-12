CREATE TABLE document_permissions (
    id UUID PRIMARY KEY,
    document_id UUID NOT NULL REFERENCES documents(id),
    subject_type TEXT NOT NULL,
    subject_id UUID NOT NULL,
    permission TEXT NOT NULL,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX document_permissions_unique_subject
    ON document_permissions (document_id, subject_type, subject_id);

CREATE INDEX document_permissions_subject_idx
    ON document_permissions (subject_type, subject_id);
