package docs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrConflict      = errors.New("conflict")
	ErrForbidden     = errors.New("forbidden")
	ErrInvalidInput  = errors.New("invalid input")
	ErrVersionConflict = errors.New("version conflict")
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) CreateDocument(ctx context.Context, tx *sql.Tx, doc Document) error {
	exec := s.exec(tx)
	_, err := exec.ExecContext(ctx, `
		INSERT INTO documents (
			id, owner_id, folder_id, title, format_version, revision, block_count,
			is_favorite, created_at, updated_at, deleted_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)`,
		doc.ID, doc.OwnerID, nullStringPtr(doc.FolderID), doc.Title, doc.FormatVersion,
		doc.Revision, doc.BlockCount, boolToInt(doc.IsFavorite),
		fmtTime(doc.CreatedAt), fmtTime(doc.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("insert document: %w", err)
	}
	_, err = exec.ExecContext(ctx, `
		INSERT INTO document_members (document_id, user_id, role, created_at)
		VALUES (?, ?, ?, ?)`,
		doc.ID, doc.OwnerID, RoleOwner, fmtTime(doc.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("insert document member: %w", err)
	}
	return nil
}

func (s *Store) GetDocument(ctx context.Context, id string) (Document, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, owner_id, folder_id, title, format_version, revision, block_count,
		       is_favorite, created_at, updated_at, deleted_at
		FROM documents WHERE id = ?`, id)
	return scanDocument(row)
}

func (s *Store) GetDocumentRole(ctx context.Context, documentID, userID string) (Role, error) {
	var role string
	err := s.db.QueryRowContext(ctx, `
		SELECT role FROM document_members
		WHERE document_id = ? AND user_id = ?`, documentID, userID).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		// Fallback: owner_id match for legacy/consistency
		var ownerID string
		err2 := s.db.QueryRowContext(ctx, `SELECT owner_id FROM documents WHERE id = ?`, documentID).Scan(&ownerID)
		if errors.Is(err2, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		if err2 != nil {
			return "", err2
		}
		if ownerID == userID {
			return RoleOwner, nil
		}
		return "", ErrForbidden
	}
	if err != nil {
		return "", err
	}
	return Role(role), nil
}

func (s *Store) ListMembers(ctx context.Context, documentID string) ([]Member, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT m.user_id, u.username, u.display_name, u.avatar_color, m.role, m.created_at
		FROM document_members m
		INNER JOIN users u ON u.id = m.user_id
		WHERE m.document_id = ?
		ORDER BY
			CASE m.role WHEN 'owner' THEN 0 WHEN 'editor' THEN 1 ELSE 2 END,
			u.username COLLATE NOCASE ASC
	`, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Member
	for rows.Next() {
		var m Member
		var created string
		if err := rows.Scan(&m.UserID, &m.Username, &m.DisplayName, &m.AvatarColor, &m.Role, &created); err != nil {
			return nil, err
		}
		m.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) UpsertMember(ctx context.Context, documentID, userID string, role Role, createdAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO document_members (document_id, user_id, role, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(document_id, user_id) DO UPDATE SET role = excluded.role
	`, documentID, userID, role, fmtTime(createdAt))
	if err != nil {
		return fmt.Errorf("upsert member: %w", err)
	}
	return nil
}

func (s *Store) DeleteMember(ctx context.Context, documentID, userID string) error {
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM document_members WHERE document_id = ? AND user_id = ?
	`, documentID, userID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func listDocumentsWhere(q ListDocumentsQuery) (where string, args []any) {
	// First ? is favorites JOIN user; second is membership filter.
	args = []any{q.UserID, q.UserID}
	parts := []string{"m.user_id = ?"}
	if q.Trash {
		parts = append(parts, "d.deleted_at IS NOT NULL")
	} else {
		parts = append(parts, "d.deleted_at IS NULL")
	}
	if q.Favorite {
		parts = append(parts, "f.user_id IS NOT NULL")
	}
	if strings.TrimSpace(q.Query) != "" {
		parts = append(parts, "d.title LIKE ?")
		args = append(args, "%"+strings.TrimSpace(q.Query)+"%")
	}
	return strings.Join(parts, " AND "), args
}

func (s *Store) CountDocuments(ctx context.Context, q ListDocumentsQuery) (int, error) {
	where, args := listDocumentsWhere(q)
	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM documents d
		INNER JOIN document_members m ON m.document_id = d.id
		INNER JOIN users u ON u.id = d.owner_id
		LEFT JOIN document_favorites f ON f.document_id = d.id AND f.user_id = ?
		WHERE %s`, where)
	var n int
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func (s *Store) ListDocuments(ctx context.Context, q ListDocumentsQuery) ([]Document, error) {
	if q.Limit <= 0 || q.Limit > 200 {
		q.Limit = 20
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
	sortCol := "d.updated_at"
	switch q.Sort {
	case "title":
		sortCol = "d.title COLLATE NOCASE"
	case "created_at":
		sortCol = "d.created_at"
	case "updated_at", "":
		sortCol = "d.updated_at"
	default:
		return nil, fmt.Errorf("%w: invalid sort", ErrInvalidInput)
	}
	order := "DESC"
	if strings.EqualFold(q.Order, "asc") {
		order = "ASC"
	}

	where, args := listDocumentsWhere(q)
	query := fmt.Sprintf(`
		SELECT d.id, d.owner_id, d.folder_id, d.title, d.format_version, d.revision, d.block_count,
		       CASE WHEN f.user_id IS NOT NULL THEN 1 ELSE 0 END AS is_favorite,
		       d.created_at, d.updated_at, d.deleted_at, m.role,
		       COALESCE(NULLIF(TRIM(u.display_name), ''), u.username) AS owner_name
		FROM documents d
		INNER JOIN document_members m ON m.document_id = d.id
		INNER JOIN users u ON u.id = d.owner_id
		LEFT JOIN document_favorites f ON f.document_id = d.id AND f.user_id = ?
		WHERE %s
		ORDER BY %s %s
		LIMIT ? OFFSET ?`, where, sortCol, order)
	args = append(args, q.Limit, q.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Document
	for rows.Next() {
		doc, err := scanDocumentWithRole(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, doc)
	}
	return out, rows.Err()
}

func (s *Store) HasFavorite(ctx context.Context, userID, documentID string) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `
		SELECT 1 FROM document_favorites WHERE user_id = ? AND document_id = ?`,
		userID, documentID,
	).Scan(&n)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) SetFavorite(ctx context.Context, userID, documentID string, favorite bool) error {
	if favorite {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO document_favorites (user_id, document_id, created_at)
			VALUES (?, ?, ?)
			ON CONFLICT(user_id, document_id) DO NOTHING`,
			userID, documentID, fmtTime(time.Now().UTC()),
		)
		return err
	}
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM document_favorites WHERE user_id = ? AND document_id = ?`,
		userID, documentID,
	)
	return err
}

func (s *Store) UpdateDocument(ctx context.Context, tx *sql.Tx, doc Document) error {
	exec := s.exec(tx)
	// is_favorite on documents is legacy; per-user favorites live in document_favorites.
	res, err := exec.ExecContext(ctx, `
		UPDATE documents SET
			title = ?, folder_id = ?, revision = ?, block_count = ?, updated_at = ?, deleted_at = ?
		WHERE id = ?`,
		doc.Title, nullStringPtr(doc.FolderID),
		doc.Revision, doc.BlockCount, fmtTime(doc.UpdatedAt), nullTimePtr(doc.DeletedAt), doc.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) SoftDeleteDocument(ctx context.Context, tx *sql.Tx, id string, at time.Time) error {
	exec := s.exec(tx)
	res, err := exec.ExecContext(ctx, `
		UPDATE documents SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
		fmtTime(at), fmtTime(at), id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) RestoreDocument(ctx context.Context, tx *sql.Tx, id string, at time.Time) error {
	exec := s.exec(tx)
	res, err := exec.ExecContext(ctx, `
		UPDATE documents SET deleted_at = NULL, updated_at = ? WHERE id = ? AND deleted_at IS NOT NULL`,
		fmtTime(at), id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) InsertBlock(ctx context.Context, tx *sql.Tx, b Block) error {
	exec := s.exec(tx)
	_, err := exec.ExecContext(ctx, `
		INSERT INTO document_blocks (
			id, document_id, order_key, block_type, content_json, attrs_json,
			revision, created_at, updated_at, deleted_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)`,
		b.ID, b.DocumentID, b.OrderKey, b.BlockType,
		rawOrEmpty(b.Content), rawOrEmpty(b.Attrs), b.Revision,
		fmtTime(b.CreatedAt), fmtTime(b.UpdatedAt),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return ErrConflict
		}
		return err
	}
	return nil
}

func (s *Store) GetBlock(ctx context.Context, documentID, blockID string) (Block, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, document_id, order_key, block_type, content_json, attrs_json,
		       revision, created_at, updated_at, deleted_at
		FROM document_blocks
		WHERE document_id = ? AND id = ?`, documentID, blockID)
	return scanBlock(row)
}

func (s *Store) ListBlocks(ctx context.Context, documentID string, includeDeleted bool) ([]Block, error) {
	query := `
		SELECT id, document_id, order_key, block_type, content_json, attrs_json,
		       revision, created_at, updated_at, deleted_at
		FROM document_blocks
		WHERE document_id = ?`
	if !includeDeleted {
		query += ` AND deleted_at IS NULL`
	}
	query += ` ORDER BY order_key ASC`
	rows, err := s.db.QueryContext(ctx, query, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBlocks(rows)
}

func (s *Store) ListBlockIndex(ctx context.Context, documentID string) ([]BlockMeta, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, order_key, block_type, attrs_json
		FROM document_blocks
		WHERE document_id = ? AND deleted_at IS NULL
		ORDER BY order_key ASC`, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []BlockMeta
	i := 0
	for rows.Next() {
		var m BlockMeta
		var blockType, attrs string
		if err := rows.Scan(&m.ID, &m.OrderKey, &blockType, &attrs); err != nil {
			return nil, err
		}
		m.BlockType = BlockType(blockType)
		m.Attrs = json.RawMessage(attrs)
		m.Index = i
		i++
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) ListHeadingBlocks(ctx context.Context, documentID string) ([]Block, []int, error) {
	index, err := s.ListBlockIndex(ctx, documentID)
	if err != nil {
		return nil, nil, err
	}
	indexByID := make(map[string]int, len(index))
	for _, m := range index {
		indexByID[m.ID] = m.Index
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, document_id, order_key, block_type, content_json, attrs_json,
		       revision, created_at, updated_at, deleted_at
		FROM document_blocks
		WHERE document_id = ? AND deleted_at IS NULL AND block_type = ?
		ORDER BY order_key ASC`, documentID, BlockHeading)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	blocks, err := scanBlocks(rows)
	if err != nil {
		return nil, nil, err
	}
	indexes := make([]int, len(blocks))
	for i, b := range blocks {
		indexes[i] = indexByID[b.ID]
	}
	return blocks, indexes, nil
}

func (s *Store) ListBlocksByIDs(ctx context.Context, documentID string, ids []string) ([]Block, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, 0, len(ids)+1)
	args = append(args, documentID)
	for i, id := range ids {
		placeholders[i] = "?"
		args = append(args, id)
	}
	query := fmt.Sprintf(`
		SELECT id, document_id, order_key, block_type, content_json, attrs_json,
		       revision, created_at, updated_at, deleted_at
		FROM document_blocks
		WHERE document_id = ? AND deleted_at IS NULL AND id IN (%s)
		ORDER BY order_key ASC`, strings.Join(placeholders, ","))
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBlocks(rows)
}

func (s *Store) ListBlocksWindow(ctx context.Context, documentID string, afterKey string, limit int) ([]Block, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, document_id, order_key, block_type, content_json, attrs_json,
		       revision, created_at, updated_at, deleted_at
		FROM document_blocks
		WHERE document_id = ? AND deleted_at IS NULL AND order_key > ?
		ORDER BY order_key ASC
		LIMIT ?`, documentID, afterKey, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBlocks(rows)
}

func (s *Store) ListBlocksAround(ctx context.Context, documentID, anchorID string, before, after int) ([]Block, error) {
	anchor, err := s.GetBlock(ctx, documentID, anchorID)
	if err != nil {
		return nil, err
	}
	if before < 0 {
		before = 0
	}
	if after < 0 {
		after = 0
	}
	if before > 200 {
		before = 200
	}
	if after > 200 {
		after = 200
	}

	beforeRows, err := s.db.QueryContext(ctx, `
		SELECT id, document_id, order_key, block_type, content_json, attrs_json,
		       revision, created_at, updated_at, deleted_at
		FROM document_blocks
		WHERE document_id = ? AND deleted_at IS NULL AND order_key < ?
		ORDER BY order_key DESC
		LIMIT ?`, documentID, anchor.OrderKey, before)
	if err != nil {
		return nil, err
	}
	beforeBlocks, err := scanBlocks(beforeRows)
	beforeRows.Close()
	if err != nil {
		return nil, err
	}
	// reverse before blocks
	for i, j := 0, len(beforeBlocks)-1; i < j; i, j = i+1, j-1 {
		beforeBlocks[i], beforeBlocks[j] = beforeBlocks[j], beforeBlocks[i]
	}

	afterRows, err := s.db.QueryContext(ctx, `
		SELECT id, document_id, order_key, block_type, content_json, attrs_json,
		       revision, created_at, updated_at, deleted_at
		FROM document_blocks
		WHERE document_id = ? AND deleted_at IS NULL AND order_key > ?
		ORDER BY order_key ASC
		LIMIT ?`, documentID, anchor.OrderKey, after)
	if err != nil {
		return nil, err
	}
	afterBlocks, err := scanBlocks(afterRows)
	afterRows.Close()
	if err != nil {
		return nil, err
	}

	out := make([]Block, 0, len(beforeBlocks)+1+len(afterBlocks))
	out = append(out, beforeBlocks...)
	out = append(out, anchor)
	out = append(out, afterBlocks...)
	return out, nil
}

func (s *Store) UpdateBlock(ctx context.Context, tx *sql.Tx, b Block) error {
	exec := s.exec(tx)
	res, err := exec.ExecContext(ctx, `
		UPDATE document_blocks SET
			order_key = ?, block_type = ?, content_json = ?, attrs_json = ?,
			revision = ?, updated_at = ?, deleted_at = ?
		WHERE id = ? AND document_id = ?`,
		b.OrderKey, b.BlockType, rawOrEmpty(b.Content), rawOrEmpty(b.Attrs),
		b.Revision, fmtTime(b.UpdatedAt), nullTimePtr(b.DeletedAt), b.ID, b.DocumentID,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return ErrConflict
		}
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) SoftDeleteBlock(ctx context.Context, tx *sql.Tx, documentID, blockID string, at time.Time) error {
	exec := s.exec(tx)
	res, err := exec.ExecContext(ctx, `
		UPDATE document_blocks SET deleted_at = ?, updated_at = ?
		WHERE document_id = ? AND id = ? AND deleted_at IS NULL`,
		fmtTime(at), fmtTime(at), documentID, blockID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) NeighborOrderKeys(ctx context.Context, tx *sql.Tx, documentID, orderKey string) (prev, next string, err error) {
	q := s.query(tx)
	_ = q.QueryRowContext(ctx, `
		SELECT order_key FROM document_blocks
		WHERE document_id = ? AND deleted_at IS NULL AND order_key < ?
		ORDER BY order_key DESC LIMIT 1`, documentID, orderKey).Scan(&prev)
	_ = q.QueryRowContext(ctx, `
		SELECT order_key FROM document_blocks
		WHERE document_id = ? AND deleted_at IS NULL AND order_key > ?
		ORDER BY order_key ASC LIMIT 1`, documentID, orderKey).Scan(&next)
	return prev, next, nil
}

func (s *Store) LastOrderKey(ctx context.Context, tx *sql.Tx, documentID string) (string, error) {
	q := s.query(tx)
	var key sql.NullString
	err := q.QueryRowContext(ctx, `
		SELECT order_key FROM document_blocks
		WHERE document_id = ? AND deleted_at IS NULL
		ORDER BY order_key DESC LIMIT 1`, documentID).Scan(&key)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !key.Valid {
		return "", nil
	}
	return key.String, nil
}

func (s *Store) CountAliveBlocks(ctx context.Context, tx *sql.Tx, documentID string) (int, error) {
	q := s.query(tx)
	var n int
	err := q.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM document_blocks
		WHERE document_id = ? AND deleted_at IS NULL`, documentID).Scan(&n)
	return n, err
}

func (s *Store) AppendOperation(ctx context.Context, tx *sql.Tx, op Operation) error {
	exec := s.exec(tx)
	_, err := exec.ExecContext(ctx, `
		INSERT INTO document_operations (
			document_id, operation_id, actor_id, operation_type, payload_json, created_at
		) VALUES (?, ?, ?, ?, ?, ?)`,
		op.DocumentID, op.OperationID, op.ActorID, op.OperationType,
		rawOrEmpty(op.Payload), fmtTime(op.CreatedAt),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return ErrConflict
		}
		return err
	}
	return nil
}

func (s *Store) ListOperationsAfter(ctx context.Context, documentID string, afterSeq int64, limit int) ([]Operation, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT seq, document_id, operation_id, actor_id, operation_type, payload_json, created_at
		FROM document_operations
		WHERE document_id = ? AND seq > ?
		ORDER BY seq ASC
		LIMIT ?`, documentID, afterSeq, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Operation
	for rows.Next() {
		var op Operation
		var created string
		var payload string
		if err := rows.Scan(&op.Seq, &op.DocumentID, &op.OperationID, &op.ActorID, &op.OperationType, &payload, &created); err != nil {
			return nil, err
		}
		op.Payload = json.RawMessage(payload)
		op.CreatedAt, err = parseTime(created)
		if err != nil {
			return nil, err
		}
		out = append(out, op)
	}
	return out, rows.Err()
}

func (s *Store) InsertSnapshot(ctx context.Context, tx *sql.Tx, meta SnapshotMeta, snapshotJSON []byte) error {
	exec := s.exec(tx)
	_, err := exec.ExecContext(ctx, `
		INSERT INTO document_snapshots (
			id, document_id, revision, label, snapshot_json, created_by, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		meta.ID, meta.DocumentID, meta.Revision, meta.Label, string(snapshotJSON),
		meta.CreatedBy, fmtTime(meta.CreatedAt),
	)
	return err
}

func (s *Store) ListSnapshots(ctx context.Context, documentID string, limit int) ([]SnapshotMeta, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.id, s.document_id, s.revision, s.label, s.created_by, s.created_at, s.snapshot_json,
		       u.display_name, u.username
		FROM document_snapshots s
		LEFT JOIN users u ON u.id = s.created_by
		WHERE s.document_id = ?
		ORDER BY s.created_at DESC
		LIMIT ?`, documentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SnapshotMeta
	for rows.Next() {
		var m SnapshotMeta
		var created, raw string
		var display, username sql.NullString
		if err := rows.Scan(
			&m.ID, &m.DocumentID, &m.Revision, &m.Label, &m.CreatedBy, &created, &raw,
			&display, &username,
		); err != nil {
			return nil, err
		}
		m.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		if display.Valid && display.String != "" {
			m.CreatorName = display.String
		} else if username.Valid {
			m.CreatorName = username.String
		}
		var p SnapshotPayload
		if json.Unmarshal([]byte(raw), &p) == nil {
			m.Title = p.Title
			m.BlockCount = len(p.Blocks)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) GetSnapshot(ctx context.Context, documentID, snapshotID string) (SnapshotMeta, SnapshotPayload, error) {
	var m SnapshotMeta
	var created, raw string
	var display, username sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT s.id, s.document_id, s.revision, s.label, s.created_by, s.created_at, s.snapshot_json,
		       u.display_name, u.username
		FROM document_snapshots s
		LEFT JOIN users u ON u.id = s.created_by
		WHERE s.id = ? AND s.document_id = ?
	`, snapshotID, documentID).Scan(
		&m.ID, &m.DocumentID, &m.Revision, &m.Label, &m.CreatedBy, &created, &raw,
		&display, &username,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return SnapshotMeta{}, SnapshotPayload{}, ErrNotFound
	}
	if err != nil {
		return SnapshotMeta{}, SnapshotPayload{}, err
	}
	m.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	if display.Valid && display.String != "" {
		m.CreatorName = display.String
	} else if username.Valid {
		m.CreatorName = username.String
	}
	var payload SnapshotPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return SnapshotMeta{}, SnapshotPayload{}, fmt.Errorf("%w: invalid snapshot json", ErrInvalidInput)
	}
	m.Title = payload.Title
	m.BlockCount = len(payload.Blocks)
	return m, payload, nil
}

func (s *Store) LatestSnapshot(ctx context.Context, documentID string) (*SnapshotMeta, error) {
	list, err := s.ListSnapshots(ctx, documentID, 1)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return &list[0], nil
}

func (s *Store) TrimSnapshots(ctx context.Context, tx *sql.Tx, documentID string, keep int) (int64, error) {
	if keep < 1 {
		keep = MaxSnapshotsPerDoc
	}
	exec := s.exec(tx)
	res, err := exec.ExecContext(ctx, `
		DELETE FROM document_snapshots
		WHERE document_id = ?
		  AND id NOT IN (
			SELECT id FROM document_snapshots
			WHERE document_id = ?
			ORDER BY created_at DESC
			LIMIT ?
		  )`, documentID, documentID, keep)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) ReplaceAllBlocks(ctx context.Context, tx *sql.Tx, documentID string, blocks []Block, at time.Time) error {
	exec := s.exec(tx)
	// Hard-delete previous rows so stable block IDs can be re-inserted by the editor sync.
	if _, err := exec.ExecContext(ctx, `
		DELETE FROM document_blocks WHERE document_id = ?`, documentID); err != nil {
		return err
	}
	_ = at
	for _, b := range blocks {
		if err := s.InsertBlock(ctx, tx, b); err != nil {
			return err
		}
	}
	return nil
}

type dbExec interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type dbQuery interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (s *Store) exec(tx *sql.Tx) dbExec {
	if tx != nil {
		return tx
	}
	return s.db
}

func (s *Store) query(tx *sql.Tx) dbQuery {
	if tx != nil {
		return tx
	}
	return s.db
}

func scanDocument(row *sql.Row) (Document, error) {
	var d Document
	var folderID sql.NullString
	var created, updated string
	var deleted sql.NullString
	var fav int
	if err := row.Scan(
		&d.ID, &d.OwnerID, &folderID, &d.Title, &d.FormatVersion, &d.Revision, &d.BlockCount,
		&fav, &created, &updated, &deleted,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Document{}, ErrNotFound
		}
		return Document{}, err
	}
	d.IsFavorite = fav == 1
	if folderID.Valid {
		d.FolderID = &folderID.String
	}
	var err error
	d.CreatedAt, err = parseTime(created)
	if err != nil {
		return Document{}, err
	}
	d.UpdatedAt, err = parseTime(updated)
	if err != nil {
		return Document{}, err
	}
	if deleted.Valid {
		t, err := parseTime(deleted.String)
		if err != nil {
			return Document{}, err
		}
		d.DeletedAt = &t
	}
	return d, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanDocumentWithRole(row scannable) (Document, error) {
	var d Document
	var folderID sql.NullString
	var created, updated string
	var deleted sql.NullString
	var fav int
	var role string
	var ownerName string
	if err := row.Scan(
		&d.ID, &d.OwnerID, &folderID, &d.Title, &d.FormatVersion, &d.Revision, &d.BlockCount,
		&fav, &created, &updated, &deleted, &role, &ownerName,
	); err != nil {
		return Document{}, err
	}
	d.IsFavorite = fav == 1
	d.Role = Role(role)
	d.OwnerName = ownerName
	if folderID.Valid {
		d.FolderID = &folderID.String
	}
	var err error
	d.CreatedAt, err = parseTime(created)
	if err != nil {
		return Document{}, err
	}
	d.UpdatedAt, err = parseTime(updated)
	if err != nil {
		return Document{}, err
	}
	if deleted.Valid {
		t, err := parseTime(deleted.String)
		if err != nil {
			return Document{}, err
		}
		d.DeletedAt = &t
	}
	return d, nil
}

func scanBlock(row *sql.Row) (Block, error) {
	var b Block
	var content, attrs, created, updated string
	var deleted sql.NullString
	var blockType string
	if err := row.Scan(
		&b.ID, &b.DocumentID, &b.OrderKey, &blockType, &content, &attrs,
		&b.Revision, &created, &updated, &deleted,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Block{}, ErrNotFound
		}
		return Block{}, err
	}
	b.BlockType = BlockType(blockType)
	b.Content = json.RawMessage(content)
	b.Attrs = json.RawMessage(attrs)
	var err error
	b.CreatedAt, err = parseTime(created)
	if err != nil {
		return Block{}, err
	}
	b.UpdatedAt, err = parseTime(updated)
	if err != nil {
		return Block{}, err
	}
	if deleted.Valid {
		t, err := parseTime(deleted.String)
		if err != nil {
			return Block{}, err
		}
		b.DeletedAt = &t
	}
	return b, nil
}

func scanBlocks(rows *sql.Rows) ([]Block, error) {
	var out []Block
	for rows.Next() {
		var b Block
		var content, attrs, created, updated string
		var deleted sql.NullString
		var blockType string
		if err := rows.Scan(
			&b.ID, &b.DocumentID, &b.OrderKey, &blockType, &content, &attrs,
			&b.Revision, &created, &updated, &deleted,
		); err != nil {
			return nil, err
		}
		b.BlockType = BlockType(blockType)
		b.Content = json.RawMessage(content)
		b.Attrs = json.RawMessage(attrs)
		var err error
		b.CreatedAt, err = parseTime(created)
		if err != nil {
			return nil, err
		}
		b.UpdatedAt, err = parseTime(updated)
		if err != nil {
			return nil, err
		}
		if deleted.Valid {
			t, err := parseTime(deleted.String)
			if err != nil {
				return nil, err
			}
			b.DeletedAt = &t
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *Store) InsertAsset(ctx context.Context, a Asset) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO document_assets (
			id, document_id, uploader_id, filename, mime, size_bytes, ext, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, a.ID, a.DocumentID, a.UploaderID, a.Filename, a.MIME, a.SizeBytes, a.Ext, fmtTime(a.CreatedAt))
	return err
}

func (s *Store) GetAsset(ctx context.Context, documentID, assetID string) (Asset, error) {
	var a Asset
	var created string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, document_id, uploader_id, filename, mime, size_bytes, ext, created_at
		FROM document_assets
		WHERE document_id = ? AND id = ?
	`, documentID, assetID).Scan(
		&a.ID, &a.DocumentID, &a.UploaderID, &a.Filename, &a.MIME, &a.SizeBytes, &a.Ext, &created,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Asset{}, ErrNotFound
		}
		return Asset{}, err
	}
	t, err := parseTime(created)
	if err != nil {
		return Asset{}, err
	}
	a.CreatedAt = t
	return a, nil
}

func fmtTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, s)
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nullStringPtr(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

func nullTimePtr(p *time.Time) any {
	if p == nil {
		return nil
	}
	return fmtTime(*p)
}

func rawOrEmpty(r json.RawMessage) string {
	if len(r) == 0 {
		return "{}"
	}
	return string(r)
}
