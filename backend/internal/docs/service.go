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

type Service struct {
	store         *Store
	users         UserLookup
	assetsDir     string
	maxAssetBytes int64
}

func NewService(store *Store, users UserLookup) *Service {
	return &Service{
		store:         store,
		users:         users,
		assetsDir:     "data/assets",
		maxAssetBytes: DefaultMaxAssetBytes,
	}
}

func (s *Service) requireRole(ctx context.Context, documentID, userID string, need func(Role) bool) (Document, Role, error) {
	doc, err := s.store.GetDocument(ctx, documentID)
	if err != nil {
		return Document{}, "", err
	}
	role, err := s.store.GetDocumentRole(ctx, documentID, userID)
	if err != nil {
		return Document{}, "", err
	}
	if !need(role) {
		return Document{}, "", ErrForbidden
	}
	doc.Role = role
	fav, err := s.store.HasFavorite(ctx, userID, documentID)
	if err != nil {
		return Document{}, "", err
	}
	doc.IsFavorite = fav
	return doc, role, nil
}

type CreateDocumentInput struct {
	ActorID  string
	Title    string
	FolderID *string
}

func (s *Service) CreateDocument(ctx context.Context, in CreateDocumentInput) (Document, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = "未命名文档"
	}
	if len(title) > 200 {
		return Document{}, fmt.Errorf("%w: title too long", ErrInvalidInput)
	}
	now := time.Now().UTC()
	doc := Document{
		ID:            uuid.NewString(),
		OwnerID:       in.ActorID,
		FolderID:      in.FolderID,
		Title:         title,
		FormatVersion: FormatVersion,
		Revision:      0,
		BlockCount:    0,
		CreatedAt:     now,
		UpdatedAt:     now,
		Role:          RoleOwner,
	}
	if err := s.store.WithTx(ctx, func(tx *sql.Tx) error {
		return s.store.CreateDocument(ctx, tx, doc)
	}); err != nil {
		return Document{}, err
	}
	return doc, nil
}

func (s *Service) GetDocument(ctx context.Context, actorID, documentID string) (Document, error) {
	doc, _, err := s.requireRole(ctx, documentID, actorID, Role.CanRead)
	return doc, err
}

type DocumentListResult struct {
	Documents []Document
	Total     int
	Limit     int
	Offset    int
}

func (s *Service) ListDocuments(ctx context.Context, q ListDocumentsQuery) (DocumentListResult, error) {
	if q.Limit <= 0 || q.Limit > 200 {
		q.Limit = 20
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
	total, err := s.store.CountDocuments(ctx, q)
	if err != nil {
		return DocumentListResult{}, err
	}
	list, err := s.store.ListDocuments(ctx, q)
	if err != nil {
		return DocumentListResult{}, err
	}
	if list == nil {
		list = []Document{}
	}
	return DocumentListResult{
		Documents: list,
		Total:     total,
		Limit:     q.Limit,
		Offset:    q.Offset,
	}, nil
}

type PatchDocumentInput struct {
	ActorID    string
	DocumentID string
	Title      *string
	IsFavorite *bool
	FolderID   *string
	ClearFolder bool
}

func (s *Service) PatchDocument(ctx context.Context, in PatchDocumentInput) (Document, error) {
	mutatesDoc := in.Title != nil || in.FolderID != nil || in.ClearFolder
	need := Role.CanRead
	if mutatesDoc {
		need = Role.CanEdit
	}
	doc, role, err := s.requireRole(ctx, in.DocumentID, in.ActorID, need)
	if err != nil {
		return Document{}, err
	}
	if doc.DeletedAt != nil {
		return Document{}, ErrNotFound
	}
	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		if title == "" {
			return Document{}, fmt.Errorf("%w: title required", ErrInvalidInput)
		}
		if len(title) > 200 {
			return Document{}, fmt.Errorf("%w: title too long", ErrInvalidInput)
		}
		doc.Title = title
	}
	if in.ClearFolder {
		doc.FolderID = nil
	} else if in.FolderID != nil {
		doc.FolderID = in.FolderID
	}
	if mutatesDoc {
		doc.UpdatedAt = time.Now().UTC()
		doc.Role = role
		if err := s.store.WithTx(ctx, func(tx *sql.Tx) error {
			return s.store.UpdateDocument(ctx, tx, doc)
		}); err != nil {
			return Document{}, err
		}
	}
	if in.IsFavorite != nil {
		if err := s.store.SetFavorite(ctx, in.ActorID, in.DocumentID, *in.IsFavorite); err != nil {
			return Document{}, err
		}
		doc.IsFavorite = *in.IsFavorite
	}
	doc.Role = role
	return doc, nil
}

func (s *Service) DeleteDocument(ctx context.Context, actorID, documentID string) error {
	if _, _, err := s.requireRole(ctx, documentID, actorID, Role.CanManage); err != nil {
		return err
	}
	now := time.Now().UTC()
	return s.store.WithTx(ctx, func(tx *sql.Tx) error {
		return s.store.SoftDeleteDocument(ctx, tx, documentID, now)
	})
}

func (s *Service) RestoreDocument(ctx context.Context, actorID, documentID string) (Document, error) {
	doc, role, err := s.requireRole(ctx, documentID, actorID, Role.CanManage)
	if err != nil {
		return Document{}, err
	}
	now := time.Now().UTC()
	if err := s.store.WithTx(ctx, func(tx *sql.Tx) error {
		return s.store.RestoreDocument(ctx, tx, documentID, now)
	}); err != nil {
		return Document{}, err
	}
	doc.DeletedAt = nil
	doc.UpdatedAt = now
	doc.Role = role
	return doc, nil
}

func (s *Service) ListBlocks(ctx context.Context, actorID, documentID string) ([]Block, error) {
	if _, _, err := s.requireRole(ctx, documentID, actorID, Role.CanRead); err != nil {
		return nil, err
	}
	return s.store.ListBlocks(ctx, documentID, false)
}

func (s *Service) ListBlockIndex(ctx context.Context, actorID, documentID string) (Document, []BlockMeta, error) {
	doc, _, err := s.requireRole(ctx, documentID, actorID, Role.CanRead)
	if err != nil {
		return Document{}, nil, err
	}
	index, err := s.store.ListBlockIndex(ctx, documentID)
	if err != nil {
		return Document{}, nil, err
	}
	if index == nil {
		index = []BlockMeta{}
	}
	return doc, index, nil
}

func (s *Service) ListTOC(ctx context.Context, actorID, documentID string) ([]TOCItem, error) {
	if _, _, err := s.requireRole(ctx, documentID, actorID, Role.CanRead); err != nil {
		return nil, err
	}
	blocks, indexes, err := s.store.ListHeadingBlocks(ctx, documentID)
	if err != nil {
		return nil, err
	}
	out := make([]TOCItem, 0, len(blocks))
	for i, b := range blocks {
		level := 1
		var attrs map[string]any
		_ = json.Unmarshal(b.Attrs, &attrs)
		if v, ok := attrs["level"].(float64); ok {
			level = int(v)
		}
		out = append(out, TOCItem{
			ID:       b.ID,
			OrderKey: b.OrderKey,
			Index:    indexes[i],
			Level:    level,
			Text:     extractHeadingText(b.Content),
		})
	}
	return out, nil
}

func (s *Service) ListBlocksByIDs(ctx context.Context, actorID, documentID string, ids []string) ([]Block, error) {
	if _, _, err := s.requireRole(ctx, documentID, actorID, Role.CanRead); err != nil {
		return nil, err
	}
	if len(ids) > 200 {
		ids = ids[:200]
	}
	return s.store.ListBlocksByIDs(ctx, documentID, ids)
}

func extractHeadingText(content json.RawMessage) string {
	var payload struct {
		Inlines []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"inlines"`
	}
	_ = json.Unmarshal(content, &payload)
	var b strings.Builder
	for _, n := range payload.Inlines {
		if n.Type == "text" {
			b.WriteString(n.Text)
		}
	}
	text := strings.TrimSpace(b.String())
	if text == "" {
		return "无标题"
	}
	return text
}

type BlocksWindowQuery struct {
	ActorID    string
	DocumentID string
	AfterKey   string
	Limit      int
	AnchorID   string
	Before     int
	After      int
}

func (s *Service) ListBlocksWindow(ctx context.Context, q BlocksWindowQuery) ([]Block, error) {
	if _, _, err := s.requireRole(ctx, q.DocumentID, q.ActorID, Role.CanRead); err != nil {
		return nil, err
	}
	if q.AnchorID != "" {
		return s.store.ListBlocksAround(ctx, q.DocumentID, q.AnchorID, q.Before, q.After)
	}
	return s.store.ListBlocksWindow(ctx, q.DocumentID, q.AfterKey, q.Limit)
}

func (s *Service) GetBlock(ctx context.Context, actorID, documentID, blockID string) (Block, error) {
	if _, _, err := s.requireRole(ctx, documentID, actorID, Role.CanRead); err != nil {
		return Block{}, err
	}
	b, err := s.store.GetBlock(ctx, documentID, blockID)
	if err != nil {
		return Block{}, err
	}
	if b.DeletedAt != nil {
		return Block{}, ErrNotFound
	}
	return b, nil
}

type CreateBlockInput struct {
	ActorID      string
	DocumentID   string
	BlockType    BlockType
	Content      json.RawMessage
	Attrs        json.RawMessage
	AfterBlockID *string
	BeforeBlockID *string
	OperationID  string
}

func (s *Service) CreateBlock(ctx context.Context, in CreateBlockInput) (Block, Document, error) {
	doc, _, err := s.requireRole(ctx, in.DocumentID, in.ActorID, Role.CanEdit)
	if err != nil {
		return Block{}, Document{}, err
	}
	if doc.DeletedAt != nil {
		return Block{}, Document{}, ErrNotFound
	}
	if !ValidBlockTypes[in.BlockType] {
		return Block{}, Document{}, fmt.Errorf("%w: invalid block_type", ErrInvalidInput)
	}
	content := SanitizeBlockContent(in.BlockType, normalizeJSON(in.Content))
	attrs := normalizeJSON(in.Attrs)
	now := time.Now().UTC()
	var created Block

	err = s.store.WithTx(ctx, func(tx *sql.Tx) error {
		orderKey, err := s.resolveInsertOrderKey(ctx, tx, in)
		if err != nil {
			return err
		}
		created = Block{
			ID:         uuid.NewString(),
			DocumentID: in.DocumentID,
			OrderKey:   orderKey,
			BlockType:  in.BlockType,
			Content:    content,
			Attrs:      attrs,
			Revision:   1,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if err := s.store.InsertBlock(ctx, tx, created); err != nil {
			return err
		}
		count, err := s.store.CountAliveBlocks(ctx, tx, in.DocumentID)
		if err != nil {
			return err
		}
		doc.BlockCount = count
		doc.Revision++
		doc.UpdatedAt = now
		if err := s.store.UpdateDocument(ctx, tx, doc); err != nil {
			return err
		}
		return s.appendOp(ctx, tx, in.DocumentID, in.ActorID, in.OperationID, "block.create", created, now)
	})
	if err != nil {
		return Block{}, Document{}, err
	}
	return created, doc, nil
}

type UpdateBlockInput struct {
	ActorID     string
	DocumentID  string
	BlockID     string
	BlockType   *BlockType
	Content     json.RawMessage
	Attrs       json.RawMessage
	BaseRevision *int64
	OperationID string
}

func (s *Service) UpdateBlock(ctx context.Context, in UpdateBlockInput) (Block, Document, error) {
	doc, _, err := s.requireRole(ctx, in.DocumentID, in.ActorID, Role.CanEdit)
	if err != nil {
		return Block{}, Document{}, err
	}
	if doc.DeletedAt != nil {
		return Block{}, Document{}, ErrNotFound
	}
	block, err := s.store.GetBlock(ctx, in.DocumentID, in.BlockID)
	if err != nil {
		return Block{}, Document{}, err
	}
	if block.DeletedAt != nil {
		return Block{}, Document{}, ErrNotFound
	}
	if in.BaseRevision != nil && *in.BaseRevision != block.Revision {
		return Block{}, Document{}, ErrVersionConflict
	}
	if in.BlockType != nil {
		if !ValidBlockTypes[*in.BlockType] {
			return Block{}, Document{}, fmt.Errorf("%w: invalid block_type", ErrInvalidInput)
		}
		block.BlockType = *in.BlockType
	}
	if in.Content != nil {
		block.Content = SanitizeBlockContent(block.BlockType, normalizeJSON(in.Content))
	}
	if in.Attrs != nil {
		block.Attrs = normalizeJSON(in.Attrs)
	}
	now := time.Now().UTC()
	block.Revision++
	block.UpdatedAt = now

	err = s.store.WithTx(ctx, func(tx *sql.Tx) error {
		if err := s.store.UpdateBlock(ctx, tx, block); err != nil {
			return err
		}
		doc.Revision++
		doc.UpdatedAt = now
		if err := s.store.UpdateDocument(ctx, tx, doc); err != nil {
			return err
		}
		return s.appendOp(ctx, tx, in.DocumentID, in.ActorID, in.OperationID, "block.update", block, now)
	})
	if err != nil {
		return Block{}, Document{}, err
	}
	return block, doc, nil
}

type DeleteBlockInput struct {
	ActorID     string
	DocumentID  string
	BlockID     string
	OperationID string
}

func (s *Service) DeleteBlock(ctx context.Context, in DeleteBlockInput) (Document, error) {
	doc, _, err := s.requireRole(ctx, in.DocumentID, in.ActorID, Role.CanEdit)
	if err != nil {
		return Document{}, err
	}
	now := time.Now().UTC()
	err = s.store.WithTx(ctx, func(tx *sql.Tx) error {
		if err := s.store.SoftDeleteBlock(ctx, tx, in.DocumentID, in.BlockID, now); err != nil {
			return err
		}
		count, err := s.store.CountAliveBlocks(ctx, tx, in.DocumentID)
		if err != nil {
			return err
		}
		doc.BlockCount = count
		doc.Revision++
		doc.UpdatedAt = now
		if err := s.store.UpdateDocument(ctx, tx, doc); err != nil {
			return err
		}
		payload, _ := json.Marshal(map[string]string{"block_id": in.BlockID})
		return s.store.AppendOperation(ctx, tx, Operation{
			DocumentID:    in.DocumentID,
			OperationID:   ensureOpID(in.OperationID),
			ActorID:       in.ActorID,
			OperationType: "block.delete",
			Payload:       payload,
			CreatedAt:     now,
		})
	})
	return doc, err
}

type MoveBlockInput struct {
	ActorID       string
	DocumentID    string
	BlockID       string
	AfterBlockID  *string
	BeforeBlockID *string
	OperationID   string
}

func (s *Service) MoveBlock(ctx context.Context, in MoveBlockInput) (Block, Document, error) {
	doc, _, err := s.requireRole(ctx, in.DocumentID, in.ActorID, Role.CanEdit)
	if err != nil {
		return Block{}, Document{}, err
	}
	block, err := s.store.GetBlock(ctx, in.DocumentID, in.BlockID)
	if err != nil {
		return Block{}, Document{}, err
	}
	if block.DeletedAt != nil {
		return Block{}, Document{}, ErrNotFound
	}
	now := time.Now().UTC()
	err = s.store.WithTx(ctx, func(tx *sql.Tx) error {
		orderKey, err := s.resolveMoveOrderKey(ctx, tx, in)
		if err != nil {
			return err
		}
		block.OrderKey = orderKey
		block.Revision++
		block.UpdatedAt = now
		if err := s.store.UpdateBlock(ctx, tx, block); err != nil {
			return err
		}
		doc.Revision++
		doc.UpdatedAt = now
		if err := s.store.UpdateDocument(ctx, tx, doc); err != nil {
			return err
		}
		return s.appendOp(ctx, tx, in.DocumentID, in.ActorID, in.OperationID, "block.move", block, now)
	})
	if err != nil {
		return Block{}, Document{}, err
	}
	return block, doc, nil
}

func (s *Service) ListChanges(ctx context.Context, actorID, documentID string, afterSeq int64) ([]Operation, error) {
	if _, _, err := s.requireRole(ctx, documentID, actorID, Role.CanRead); err != nil {
		return nil, err
	}
	return s.store.ListOperationsAfter(ctx, documentID, afterSeq, 200)
}

type ImportDocumentInput struct {
	ActorID  string
	Title    string
	Markdown string
}

func (s *Service) ImportMarkdown(ctx context.Context, in ImportDocumentInput, blocks []Block) (Document, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = "导入文档"
	}
	now := time.Now().UTC()
	doc := Document{
		ID:            uuid.NewString(),
		OwnerID:       in.ActorID,
		Title:         title,
		FormatVersion: FormatVersion,
		Revision:      1,
		BlockCount:    len(blocks),
		CreatedAt:     now,
		UpdatedAt:     now,
		Role:          RoleOwner,
	}
	err := s.store.WithTx(ctx, func(tx *sql.Tx) error {
		if err := s.store.CreateDocument(ctx, tx, doc); err != nil {
			return err
		}
		key := ""
		for i := range blocks {
			blocks[i].ID = uuid.NewString()
			blocks[i].DocumentID = doc.ID
			key = OrderKeyAfter(key)
			blocks[i].OrderKey = key
			blocks[i].Revision = 1
			blocks[i].CreatedAt = now
			blocks[i].UpdatedAt = now
			blocks[i].Content = SanitizeBlockContent(blocks[i].BlockType, normalizeJSON(blocks[i].Content))
			if err := s.store.InsertBlock(ctx, tx, blocks[i]); err != nil {
				return err
			}
		}
		payload, _ := json.Marshal(map[string]any{"block_count": len(blocks)})
		return s.store.AppendOperation(ctx, tx, Operation{
			DocumentID:    doc.ID,
			OperationID:   uuid.NewString(),
			ActorID:       in.ActorID,
			OperationType: "document.import",
			Payload:       payload,
			CreatedAt:     now,
		})
	})
	if err != nil {
		return Document{}, err
	}
	return doc, nil
}

func (s *Service) LoadBlocksForExport(ctx context.Context, actorID, documentID string) (Document, []Block, error) {
	doc, _, err := s.requireRole(ctx, documentID, actorID, Role.CanRead)
	if err != nil {
		return Document{}, nil, err
	}
	blocks, err := s.store.ListBlocks(ctx, documentID, false)
	if err != nil {
		return Document{}, nil, err
	}
	return doc, blocks, nil
}

type ContentBlockInput struct {
	ID        string          `json:"id"`
	BlockType BlockType       `json:"block_type"`
	Content   json.RawMessage `json:"content"`
	Attrs     json.RawMessage `json:"attrs"`
}

type SyncContentInput struct {
	ActorID      string
	DocumentID   string
	BaseRevision *int64
	OperationID  string
	Blocks       []ContentBlockInput
}

func (s *Service) SyncContent(ctx context.Context, in SyncContentInput) (Document, []Block, error) {
	doc, _, err := s.requireRole(ctx, in.DocumentID, in.ActorID, Role.CanEdit)
	if err != nil {
		return Document{}, nil, err
	}
	if doc.DeletedAt != nil {
		return Document{}, nil, ErrNotFound
	}
	if in.BaseRevision != nil && *in.BaseRevision != doc.Revision {
		return Document{}, nil, fmt.Errorf(
			"%w: client base_revision=%d server revision=%d",
			ErrVersionConflict, *in.BaseRevision, doc.Revision,
		)
	}
	if len(in.Blocks) > 100000 {
		return Document{}, nil, fmt.Errorf("%w: too many blocks", ErrInvalidInput)
	}

	now := time.Now().UTC()
	blocks := make([]Block, 0, len(in.Blocks))
	orderKey := ""
	for _, raw := range in.Blocks {
		if !ValidBlockTypes[raw.BlockType] {
			return Document{}, nil, fmt.Errorf("%w: invalid block_type %s", ErrInvalidInput, raw.BlockType)
		}
		id := strings.TrimSpace(raw.ID)
		if id == "" {
			id = uuid.NewString()
		}
		orderKey = OrderKeyAfter(orderKey)
		blocks = append(blocks, Block{
			ID:         id,
			DocumentID: in.DocumentID,
			OrderKey:   orderKey,
			BlockType:  raw.BlockType,
			Content:    SanitizeBlockContent(raw.BlockType, normalizeJSON(raw.Content)),
			Attrs:      normalizeJSON(raw.Attrs),
			Revision:   1,
			CreatedAt:  now,
			UpdatedAt:  now,
		})
	}

	err = s.store.WithTx(ctx, func(tx *sql.Tx) error {
		if err := s.store.ReplaceAllBlocks(ctx, tx, in.DocumentID, blocks, now); err != nil {
			return err
		}
		doc.BlockCount = len(blocks)
		doc.Revision++
		doc.UpdatedAt = now
		if err := s.store.UpdateDocument(ctx, tx, doc); err != nil {
			return err
		}
		payload, _ := json.Marshal(map[string]any{"block_count": len(blocks)})
		return s.store.AppendOperation(ctx, tx, Operation{
			DocumentID:    in.DocumentID,
			OperationID:   ensureOpID(in.OperationID),
			ActorID:       in.ActorID,
			OperationType: "document.sync_content",
			Payload:       payload,
			CreatedAt:     now,
		})
	})
	if err != nil {
		return Document{}, nil, err
	}
	// User-visible history: auto snapshot after successful materialize/sync.
	s.maybeAutoSnapshot(ctx, doc, in.ActorID, blocks)
	return doc, blocks, nil
}

func (s *Service) resolveInsertOrderKey(ctx context.Context, tx *sql.Tx, in CreateBlockInput) (string, error) {
	var prev, next string
	if in.AfterBlockID != nil && *in.AfterBlockID != "" {
		b, err := s.store.GetBlock(ctx, in.DocumentID, *in.AfterBlockID)
		if err != nil {
			return "", err
		}
		prev = b.OrderKey
		_, n, _ := s.store.NeighborOrderKeys(ctx, tx, in.DocumentID, prev)
		next = n
	} else if in.BeforeBlockID != nil && *in.BeforeBlockID != "" {
		b, err := s.store.GetBlock(ctx, in.DocumentID, *in.BeforeBlockID)
		if err != nil {
			return "", err
		}
		next = b.OrderKey
		p, _, _ := s.store.NeighborOrderKeys(ctx, tx, in.DocumentID, next)
		prev = p
	} else {
		last, err := s.store.LastOrderKey(ctx, tx, in.DocumentID)
		if err != nil {
			return "", err
		}
		prev = last
	}
	return OrderKeyBetween(prev, next)
}

func (s *Service) resolveMoveOrderKey(ctx context.Context, tx *sql.Tx, in MoveBlockInput) (string, error) {
	var prev, next string
	if in.AfterBlockID != nil && *in.AfterBlockID != "" {
		if *in.AfterBlockID == in.BlockID {
			return "", fmt.Errorf("%w: cannot move after self", ErrInvalidInput)
		}
		b, err := s.store.GetBlock(ctx, in.DocumentID, *in.AfterBlockID)
		if err != nil {
			return "", err
		}
		prev = b.OrderKey
		_, n, _ := s.store.NeighborOrderKeys(ctx, tx, in.DocumentID, prev)
		next = n
		if next == "" {
			// ok
		} else {
			// if next is the moving block itself, skip to its next
			moving, _ := s.store.GetBlock(ctx, in.DocumentID, in.BlockID)
			if next == moving.OrderKey {
				_, n2, _ := s.store.NeighborOrderKeys(ctx, tx, in.DocumentID, next)
				next = n2
			}
		}
	} else if in.BeforeBlockID != nil && *in.BeforeBlockID != "" {
		if *in.BeforeBlockID == in.BlockID {
			return "", fmt.Errorf("%w: cannot move before self", ErrInvalidInput)
		}
		b, err := s.store.GetBlock(ctx, in.DocumentID, *in.BeforeBlockID)
		if err != nil {
			return "", err
		}
		next = b.OrderKey
		p, _, _ := s.store.NeighborOrderKeys(ctx, tx, in.DocumentID, next)
		prev = p
		moving, _ := s.store.GetBlock(ctx, in.DocumentID, in.BlockID)
		if prev == moving.OrderKey {
			p2, _, _ := s.store.NeighborOrderKeys(ctx, tx, in.DocumentID, prev)
			prev = p2
		}
	} else {
		last, err := s.store.LastOrderKey(ctx, tx, in.DocumentID)
		if err != nil {
			return "", err
		}
		moving, _ := s.store.GetBlock(ctx, in.DocumentID, in.BlockID)
		if last == moving.OrderKey {
			p, _, _ := s.store.NeighborOrderKeys(ctx, tx, in.DocumentID, last)
			prev = p
			next = ""
			return OrderKeyBetween(prev, next)
		}
		prev = last
	}
	return OrderKeyBetween(prev, next)
}

func (s *Service) appendOp(ctx context.Context, tx *sql.Tx, docID, actorID, opID, opType string, block Block, at time.Time) error {
	payload, _ := json.Marshal(block)
	return s.store.AppendOperation(ctx, tx, Operation{
		DocumentID:    docID,
		OperationID:   ensureOpID(opID),
		ActorID:       actorID,
		OperationType: opType,
		Payload:       payload,
		CreatedAt:     at,
	})
}

func ensureOpID(id string) string {
	if strings.TrimSpace(id) == "" {
		return uuid.NewString()
	}
	return id
}

func normalizeJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	if !json.Valid(raw) {
		return json.RawMessage(`{}`)
	}
	return raw
}
