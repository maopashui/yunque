-- +goose Up
CREATE TABLE document_assets (
    id           TEXT PRIMARY KEY NOT NULL,
    document_id  TEXT NOT NULL,
    uploader_id  TEXT NOT NULL,
    filename     TEXT NOT NULL DEFAULT '',
    mime         TEXT NOT NULL,
    size_bytes   INTEGER NOT NULL,
    ext          TEXT NOT NULL,
    created_at   TEXT NOT NULL,
    FOREIGN KEY (document_id) REFERENCES documents (id) ON DELETE CASCADE,
    FOREIGN KEY (uploader_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_document_assets_document ON document_assets (document_id);

-- +goose Down
DROP TABLE IF EXISTS document_assets;
