-- +goose Up
CREATE TABLE folders (
    id          TEXT PRIMARY KEY NOT NULL,
    owner_id    TEXT NOT NULL,
    parent_id   TEXT,
    name        TEXT NOT NULL,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL,
    deleted_at  TEXT,
    FOREIGN KEY (owner_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (parent_id) REFERENCES folders (id) ON DELETE SET NULL
);

CREATE INDEX idx_folders_owner ON folders (owner_id);
CREATE INDEX idx_folders_parent ON folders (parent_id);

CREATE TABLE documents (
    id              TEXT PRIMARY KEY NOT NULL,
    owner_id        TEXT NOT NULL,
    folder_id       TEXT,
    title           TEXT NOT NULL DEFAULT '未命名文档',
    format_version  INTEGER NOT NULL DEFAULT 1,
    revision        INTEGER NOT NULL DEFAULT 0,
    block_count     INTEGER NOT NULL DEFAULT 0,
    is_favorite     INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL,
    deleted_at      TEXT,
    FOREIGN KEY (owner_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (folder_id) REFERENCES folders (id) ON DELETE SET NULL
);

CREATE INDEX idx_documents_owner ON documents (owner_id);
CREATE INDEX idx_documents_folder ON documents (folder_id);
CREATE INDEX idx_documents_updated ON documents (updated_at);
CREATE INDEX idx_documents_title ON documents (title);
CREATE INDEX idx_documents_deleted ON documents (deleted_at);

CREATE TABLE document_members (
    document_id TEXT NOT NULL,
    user_id     TEXT NOT NULL,
    role        TEXT NOT NULL CHECK (role IN ('owner', 'editor', 'viewer')),
    created_at  TEXT NOT NULL,
    PRIMARY KEY (document_id, user_id),
    FOREIGN KEY (document_id) REFERENCES documents (id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_document_members_user ON document_members (user_id);

CREATE TABLE document_blocks (
    id           TEXT PRIMARY KEY NOT NULL,
    document_id  TEXT NOT NULL,
    order_key    TEXT NOT NULL,
    block_type   TEXT NOT NULL,
    content_json TEXT NOT NULL DEFAULT '{}',
    attrs_json   TEXT NOT NULL DEFAULT '{}',
    revision     INTEGER NOT NULL DEFAULT 1,
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL,
    deleted_at   TEXT,
    FOREIGN KEY (document_id) REFERENCES documents (id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX idx_document_blocks_order
    ON document_blocks (document_id, order_key)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_document_blocks_doc ON document_blocks (document_id);
CREATE INDEX idx_document_blocks_doc_alive
    ON document_blocks (document_id, order_key)
    WHERE deleted_at IS NULL;

CREATE TABLE document_operations (
    seq            INTEGER PRIMARY KEY AUTOINCREMENT,
    document_id    TEXT NOT NULL,
    operation_id   TEXT NOT NULL,
    actor_id       TEXT NOT NULL,
    operation_type TEXT NOT NULL,
    payload_json   TEXT NOT NULL DEFAULT '{}',
    created_at     TEXT NOT NULL,
    FOREIGN KEY (document_id) REFERENCES documents (id) ON DELETE CASCADE,
    FOREIGN KEY (actor_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX idx_document_operations_id
    ON document_operations (document_id, operation_id);
CREATE INDEX idx_document_operations_doc_seq
    ON document_operations (document_id, seq);

-- +goose Down
DROP TABLE IF EXISTS document_operations;
DROP TABLE IF EXISTS document_blocks;
DROP TABLE IF EXISTS document_members;
DROP TABLE IF EXISTS documents;
DROP TABLE IF EXISTS folders;
