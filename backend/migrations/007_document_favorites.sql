-- +goose Up
CREATE TABLE document_favorites (
    user_id     TEXT NOT NULL,
    document_id TEXT NOT NULL,
    created_at  TEXT NOT NULL,
    PRIMARY KEY (user_id, document_id),
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (document_id) REFERENCES documents (id) ON DELETE CASCADE
);

CREATE INDEX idx_document_favorites_document ON document_favorites (document_id);

-- Migrate legacy per-document favorites onto the owner.
INSERT INTO document_favorites (user_id, document_id, created_at)
SELECT d.owner_id, d.id, d.updated_at
FROM documents d
WHERE d.is_favorite = 1
ON CONFLICT(user_id, document_id) DO NOTHING;

UPDATE documents SET is_favorite = 0 WHERE is_favorite != 0;

-- +goose Down
DROP TABLE IF EXISTS document_favorites;
