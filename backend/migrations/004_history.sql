-- +goose Up
CREATE TABLE document_snapshots (
    id            TEXT PRIMARY KEY NOT NULL,
    document_id   TEXT NOT NULL,
    revision      INTEGER NOT NULL,
    label         TEXT NOT NULL DEFAULT '',
    snapshot_json TEXT NOT NULL,
    created_by    TEXT NOT NULL,
    created_at    TEXT NOT NULL,
    FOREIGN KEY (document_id) REFERENCES documents (id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_document_snapshots_doc_created
    ON document_snapshots (document_id, created_at DESC);
CREATE INDEX idx_document_snapshots_doc_rev
    ON document_snapshots (document_id, revision);

-- +goose Down
DROP TABLE IF EXISTS document_snapshots;
