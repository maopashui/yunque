package docs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	SnapshotLabelAuto   = "auto"
	SnapshotLabelManual = "manual"
	SnapshotLabelBeforeRestore = "before_restore"
	MaxSnapshotsPerDoc  = 100
)

type SnapshotMeta struct {
	ID          string    `json:"id"`
	DocumentID  string    `json:"document_id"`
	Revision    int64     `json:"revision"`
	Label       string    `json:"label"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	BlockCount  int       `json:"block_count"`
	Title       string    `json:"title"`
	CreatorName string    `json:"creator_name,omitempty"`
}

type SnapshotPayload struct {
	Title   string  `json:"title"`
	Blocks  []Block `json:"blocks"`
}

type SnapshotDetail struct {
	SnapshotMeta
	Blocks []Block `json:"blocks"`
}

// DiffEntry is one block in a snapshot comparison (identity + short preview).
type DiffEntry struct {
	ID        string `json:"id"`
	BlockType string `json:"block_type"`
	Preview   string `json:"preview"`
}

// DiffChangeEntry is a block present in both snapshots with different content.
type DiffChangeEntry struct {
	ID          string `json:"id"`
	BlockType   string `json:"block_type"`
	FromPreview string `json:"from_preview"`
	ToPreview   string `json:"to_preview"`
}

type SnapshotDiff struct {
	FromRevision int64             `json:"from_revision"`
	ToRevision   int64             `json:"to_revision"`
	TitleChanged bool              `json:"title_changed"`
	FromTitle    string            `json:"from_title"`
	ToTitle      string            `json:"to_title"`
	Added        []DiffEntry       `json:"added"`
	Removed      []DiffEntry       `json:"removed"`
	Changed      []DiffChangeEntry `json:"changed"`
	Unchanged    int               `json:"unchanged"`
}

func (s *Service) ListSnapshots(ctx context.Context, actorID, documentID string) ([]SnapshotMeta, error) {
	if _, _, err := s.requireRole(ctx, documentID, actorID, Role.CanRead); err != nil {
		return nil, err
	}
	return s.store.ListSnapshots(ctx, documentID, 100)
}

func (s *Service) GetSnapshot(ctx context.Context, actorID, documentID, snapshotID string) (SnapshotDetail, error) {
	if _, _, err := s.requireRole(ctx, documentID, actorID, Role.CanRead); err != nil {
		return SnapshotDetail{}, err
	}
	meta, payload, err := s.store.GetSnapshot(ctx, documentID, snapshotID)
	if err != nil {
		return SnapshotDetail{}, err
	}
	return SnapshotDetail{
		SnapshotMeta: meta,
		Blocks:       payload.Blocks,
	}, nil
}

func (s *Service) CreateNamedSnapshot(ctx context.Context, actorID, documentID, label string) (SnapshotMeta, error) {
	doc, _, err := s.requireRole(ctx, documentID, actorID, Role.CanEdit)
	if err != nil {
		return SnapshotMeta{}, err
	}
	if doc.DeletedAt != nil {
		return SnapshotMeta{}, ErrNotFound
	}
	label = strings.TrimSpace(label)
	if label == "" {
		label = SnapshotLabelManual
	}
	if len(label) > 80 {
		return SnapshotMeta{}, fmt.Errorf("%w: label too long", ErrInvalidInput)
	}
	return s.captureSnapshot(ctx, doc, actorID, label)
}

func (s *Service) captureSnapshot(ctx context.Context, doc Document, actorID, label string) (SnapshotMeta, error) {
	blocks, err := s.store.ListBlocks(ctx, doc.ID, false)
	if err != nil {
		return SnapshotMeta{}, err
	}
	payload := SnapshotPayload{Title: doc.Title, Blocks: blocks}
	raw, err := json.Marshal(payload)
	if err != nil {
		return SnapshotMeta{}, err
	}
	now := time.Now().UTC()
	meta := SnapshotMeta{
		ID:         uuid.NewString(),
		DocumentID: doc.ID,
		Revision:   doc.Revision,
		Label:      label,
		CreatedBy:  actorID,
		CreatedAt:  now,
		BlockCount: len(blocks),
		Title:      doc.Title,
	}
	err = s.store.WithTx(ctx, func(tx *sql.Tx) error {
		if err := s.store.InsertSnapshot(ctx, tx, meta, raw); err != nil {
			return err
		}
		payloadOp, _ := json.Marshal(map[string]any{
			"snapshot_id": meta.ID,
			"revision":    meta.Revision,
			"label":       label,
		})
		if err := s.store.AppendOperation(ctx, tx, Operation{
			DocumentID:    doc.ID,
			OperationID:   uuid.NewString(),
			ActorID:       actorID,
			OperationType: "document.snapshot",
			Payload:       payloadOp,
			CreatedAt:     now,
		}); err != nil {
			return err
		}
		_, err := s.store.TrimSnapshots(ctx, tx, doc.ID, MaxSnapshotsPerDoc)
		return err
	})
	if err != nil {
		return SnapshotMeta{}, err
	}
	return meta, nil
}

func (s *Service) maybeAutoSnapshot(ctx context.Context, doc Document, actorID string, blocks []Block) {
	// Throttle: skip if latest auto snapshot already covers this revision.
	latest, err := s.store.LatestSnapshot(ctx, doc.ID)
	if err == nil && latest != nil && latest.Revision == doc.Revision && latest.Label == SnapshotLabelAuto {
		return
	}
	payload := SnapshotPayload{Title: doc.Title, Blocks: blocks}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	now := time.Now().UTC()
	meta := SnapshotMeta{
		ID:         uuid.NewString(),
		DocumentID: doc.ID,
		Revision:   doc.Revision,
		Label:      SnapshotLabelAuto,
		CreatedBy:  actorID,
		CreatedAt:  now,
		BlockCount: len(blocks),
		Title:      doc.Title,
	}
	_ = s.store.WithTx(ctx, func(tx *sql.Tx) error {
		if err := s.store.InsertSnapshot(ctx, tx, meta, raw); err != nil {
			return err
		}
		_, err := s.store.TrimSnapshots(ctx, tx, doc.ID, MaxSnapshotsPerDoc)
		return err
	})
}

func (s *Service) CompareSnapshot(ctx context.Context, actorID, documentID, snapshotID string) (SnapshotDiff, error) {
	doc, _, err := s.requireRole(ctx, documentID, actorID, Role.CanRead)
	if err != nil {
		return SnapshotDiff{}, err
	}
	meta, payload, err := s.store.GetSnapshot(ctx, documentID, snapshotID)
	if err != nil {
		return SnapshotDiff{}, err
	}
	current, err := s.store.ListBlocks(ctx, documentID, false)
	if err != nil {
		return SnapshotDiff{}, err
	}
	diff := diffSnapshots(payload, SnapshotPayload{Title: doc.Title, Blocks: current})
	diff.FromRevision = meta.Revision
	diff.ToRevision = doc.Revision
	return diff, nil
}

func blockFingerprint(b Block) string {
	return string(b.BlockType) + "\n" + blockPlainText(b)
}

// lcsPairs returns matched index pairs between fingerprint sequences (content-based).
// This keeps diffs stable even when materialize regenerates block UUIDs.
func lcsPairs(a, b []string) [][2]int {
	n, m := len(a), len(b)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	var pairs [][2]int
	i, j := 0, 0
	for i < n && j < m {
		if a[i] == b[j] {
			pairs = append(pairs, [2]int{i, j})
			i++
			j++
		} else if dp[i+1][j] >= dp[i][j+1] {
			i++
		} else {
			j++
		}
	}
	return pairs
}

func diffSnapshots(from, to SnapshotPayload) SnapshotDiff {
	diff := SnapshotDiff{
		FromTitle:    from.Title,
		ToTitle:      to.Title,
		TitleChanged: from.Title != to.Title,
		Added:        []DiffEntry{},
		Removed:      []DiffEntry{},
		Changed:      []DiffChangeEntry{},
	}

	fromFP := make([]string, len(from.Blocks))
	for i, b := range from.Blocks {
		fromFP[i] = blockFingerprint(b)
	}
	toFP := make([]string, len(to.Blocks))
	for i, b := range to.Blocks {
		toFP[i] = blockFingerprint(b)
	}
	pairs := lcsPairs(fromFP, toFP)
	matchedFrom := map[int]bool{}
	matchedTo := map[int]bool{}
	for _, p := range pairs {
		matchedFrom[p[0]] = true
		matchedTo[p[1]] = true
	}
	diff.Unchanged = len(pairs)

	for i, fb := range from.Blocks {
		if matchedFrom[i] {
			continue
		}
		diff.Removed = append(diff.Removed, makeDiffEntry(fb))
	}
	for j, tb := range to.Blocks {
		if matchedTo[j] {
			continue
		}
		diff.Added = append(diff.Added, makeDiffEntry(tb))
	}
	return diff
}

func makeDiffEntry(b Block) DiffEntry {
	return DiffEntry{
		ID:        b.ID,
		BlockType: string(b.BlockType),
		Preview:   blockPreviewText(b),
	}
}

func blockPreviewText(b Block) string {
	text := blockPlainText(b)
	if len(text) > 120 {
		text = text[:120] + "…"
	}
	return text
}

func blockPlainText(b Block) string {
	var m map[string]any
	if err := json.Unmarshal(b.Content, &m); err != nil {
		return ""
	}
	if t, ok := m["text"].(string); ok {
		return t
	}
	if inlines, ok := m["inlines"].([]any); ok {
		var sb strings.Builder
		for _, raw := range inlines {
			im, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if t, ok := im["text"].(string); ok {
				sb.WriteString(t)
			}
		}
		return sb.String()
	}
	return ""
}

func (s *Service) RestoreSnapshot(ctx context.Context, actorID, documentID, snapshotID string) (Document, []Block, SnapshotMeta, error) {
	doc, _, err := s.requireRole(ctx, documentID, actorID, Role.CanEdit)
	if err != nil {
		return Document{}, nil, SnapshotMeta{}, err
	}
	if doc.DeletedAt != nil {
		return Document{}, nil, SnapshotMeta{}, ErrNotFound
	}
	snapMeta, payload, err := s.store.GetSnapshot(ctx, documentID, snapshotID)
	if err != nil {
		return Document{}, nil, SnapshotMeta{}, err
	}

	// Preserve history: snapshot current state before restore.
	if _, err := s.captureSnapshot(ctx, doc, actorID, SnapshotLabelBeforeRestore); err != nil {
		return Document{}, nil, SnapshotMeta{}, err
	}
	// Reload doc after before_restore snapshot (revision unchanged).
	doc, err = s.store.GetDocument(ctx, documentID)
	if err != nil {
		return Document{}, nil, SnapshotMeta{}, err
	}

	now := time.Now().UTC()
	blocks := make([]Block, 0, len(payload.Blocks))
	orderKey := ""
	for _, b := range payload.Blocks {
		id := b.ID
		if id == "" {
			id = uuid.NewString()
		}
		orderKey = OrderKeyAfter(orderKey)
		blocks = append(blocks, Block{
			ID:         id,
			DocumentID: documentID,
			OrderKey:   orderKey,
			BlockType:  b.BlockType,
			Content:    normalizeJSON(b.Content),
			Attrs:      normalizeJSON(b.Attrs),
			Revision:   1,
			CreatedAt:  now,
			UpdatedAt:  now,
		})
	}

	title := strings.TrimSpace(payload.Title)
	if title == "" {
		title = doc.Title
	}

	err = s.store.WithTx(ctx, func(tx *sql.Tx) error {
		if err := s.store.ReplaceAllBlocks(ctx, tx, documentID, blocks, now); err != nil {
			return err
		}
		doc.Title = title
		doc.BlockCount = len(blocks)
		doc.Revision++
		doc.UpdatedAt = now
		if err := s.store.UpdateDocument(ctx, tx, doc); err != nil {
			return err
		}
		payloadOp, _ := json.Marshal(map[string]any{
			"restored_snapshot_id": snapMeta.ID,
			"restored_revision":    snapMeta.Revision,
		})
		return s.store.AppendOperation(ctx, tx, Operation{
			DocumentID:    documentID,
			OperationID:   uuid.NewString(),
			ActorID:       actorID,
			OperationType: "document.restore_snapshot",
			Payload:       payloadOp,
			CreatedAt:     now,
		})
	})
	if err != nil {
		return Document{}, nil, SnapshotMeta{}, err
	}

	// Record the restored state as a new snapshot entry (history preserved).
	newSnap, _ := s.captureSnapshot(ctx, doc, actorID, "restored")
	return doc, blocks, newSnap, nil
}

func (s *Service) CleanupSnapshots(ctx context.Context, actorID, documentID string, keep int) (int64, error) {
	if _, _, err := s.requireRole(ctx, documentID, actorID, Role.CanManage); err != nil {
		return 0, err
	}
	if keep <= 0 {
		keep = MaxSnapshotsPerDoc
	}
	var deleted int64
	err := s.store.WithTx(ctx, func(tx *sql.Tx) error {
		n, err := s.store.TrimSnapshots(ctx, tx, documentID, keep)
		if err != nil {
			return err
		}
		deleted = n
		return nil
	})
	return deleted, err
}
