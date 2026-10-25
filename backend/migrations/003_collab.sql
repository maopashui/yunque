-- +goose Up
CREATE TABLE document_yjs_updates (
    seq          INTEGER PRIMARY KEY AUTOINCREMENT,
    document_id  TEXT NOT NULL,
    operation_id TEXT NOT NULL,
    actor_id     TEXT NOT NULL,
    update_blob  BLOB NOT NULL,
    created_at   TEXT NOT NULL,
    FOREIGN KEY (document_id) REFERENCES documents (id) ON DELETE CASCADE,
    FOREIGN KEY (actor_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX idx_document_yjs_updates_op
    ON document_yjs_updates (document_id, operation_id);
CREATE INDEX idx_document_yjs_updates_doc_seq
    ON document_yjs_updates (document_id, seq);

CREATE TABLE document_yjs_snapshots (
    document_id TEXT PRIMARY KEY,
    snapshot    BLOB NOT NULL,
    update_seq  INTEGER NOT NULL DEFAULT 0,
    updated_at  TEXT NOT NULL,
    FOREIGN KEY (document_id) REFERENCES documents (id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE IF EXISTS document_yjs_snapshots;
DROP TABLE IF EXISTS document_yjs_updates;
