package collab

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrDuplicateOperation = errors.New("duplicate operation")

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

type StoredUpdate struct {
	Seq         int64
	DocumentID  string
	OperationID string
	ActorID     string
	Blob        []byte
	CreatedAt   time.Time
}

type Snapshot struct {
	DocumentID string
	Blob       []byte
	UpdateSeq  int64
	UpdatedAt  time.Time
}

func (s *Store) CountUpdates(ctx context.Context, documentID string) (int64, error) {
	var n int64
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM document_yjs_updates WHERE document_id = ?`, documentID,
	).Scan(&n)
	return n, err
}

func (s *Store) MaxSeq(ctx context.Context, documentID string) (int64, error) {
	var seq sql.NullInt64
	err := s.db.QueryRowContext(ctx,
		`SELECT MAX(seq) FROM document_yjs_updates WHERE document_id = ?`, documentID,
	).Scan(&seq)
	if err != nil {
		return 0, err
	}
	if !seq.Valid {
		return 0, nil
	}
	return seq.Int64, nil
}

func (s *Store) GetSnapshot(ctx context.Context, documentID string) (*Snapshot, error) {
	var snap Snapshot
	var updated string
	err := s.db.QueryRowContext(ctx, `
		SELECT document_id, snapshot, update_seq, updated_at
		FROM document_yjs_snapshots WHERE document_id = ?
	`, documentID).Scan(&snap.DocumentID, &snap.Blob, &snap.UpdateSeq, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	snap.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return &snap, nil
}

func (s *Store) UpsertSnapshot(ctx context.Context, documentID string, blob []byte, updateSeq int64) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO document_yjs_snapshots (document_id, snapshot, update_seq, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(document_id) DO UPDATE SET
			snapshot = excluded.snapshot,
			update_seq = excluded.update_seq,
			updated_at = excluded.updated_at
	`, documentID, blob, updateSeq, now)
	return err
}

func (s *Store) AppendUpdate(ctx context.Context, documentID, operationID, actorID string, blob []byte) (StoredUpdate, error) {
	if operationID == "" {
		return StoredUpdate{}, fmt.Errorf("operation_id required")
	}
	if len(blob) == 0 {
		return StoredUpdate{}, fmt.Errorf("empty update")
	}
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO document_yjs_updates (document_id, operation_id, actor_id, update_blob, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, documentID, operationID, actorID, blob, now.Format(time.RFC3339Nano))
	if err != nil {
		if isUniqueViolation(err) {
			return StoredUpdate{}, ErrDuplicateOperation
		}
		return StoredUpdate{}, err
	}
	seq, err := res.LastInsertId()
	if err != nil {
		return StoredUpdate{}, err
	}
	return StoredUpdate{
		Seq:         seq,
		DocumentID:  documentID,
		OperationID: operationID,
		ActorID:     actorID,
		Blob:        blob,
		CreatedAt:   now,
	}, nil
}

func (s *Store) ListUpdatesAfter(ctx context.Context, documentID string, afterSeq int64) ([]StoredUpdate, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT seq, document_id, operation_id, actor_id, update_blob, created_at
		FROM document_yjs_updates
		WHERE document_id = ? AND seq > ?
		ORDER BY seq ASC
	`, documentID, afterSeq)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []StoredUpdate
	for rows.Next() {
		var u StoredUpdate
		var created string
		if err := rows.Scan(&u.Seq, &u.DocumentID, &u.OperationID, &u.ActorID, &u.Blob, &created); err != nil {
			return nil, err
		}
		u.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) LoadSyncState(ctx context.Context, documentID string) (snap *Snapshot, updates []StoredUpdate, empty bool, err error) {
	snap, err = s.GetSnapshot(ctx, documentID)
	if err != nil {
		return nil, nil, false, err
	}
	after := int64(0)
	if snap != nil {
		after = snap.UpdateSeq
	}
	updates, err = s.ListUpdatesAfter(ctx, documentID, after)
	if err != nil {
		return nil, nil, false, err
	}
	count, err := s.CountUpdates(ctx, documentID)
	if err != nil {
		return nil, nil, false, err
	}
	empty = snap == nil && count == 0
	return snap, updates, empty, nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "constraint")
}
