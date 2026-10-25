package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"gitee.com/yunque/yunque/backend/internal/auth"
	"gitee.com/yunque/yunque/backend/internal/docs"
	"gitee.com/yunque/yunque/backend/internal/markdown"
)

type createDocumentRequest struct {
	Title    string  `json:"title"`
	FolderID *string `json:"folder_id"`
}

type patchDocumentRequest struct {
	Title       *string `json:"title"`
	IsFavorite  *bool   `json:"is_favorite"`
	FolderID    *string `json:"folder_id"`
	ClearFolder *bool   `json:"clear_folder"`
}

type createBlockRequest struct {
	BlockType     docs.BlockType   `json:"block_type"`
	Content       json.RawMessage  `json:"content"`
	Attrs         json.RawMessage  `json:"attrs"`
	AfterBlockID  *string          `json:"after_block_id"`
	BeforeBlockID *string          `json:"before_block_id"`
	OperationID   string           `json:"operation_id"`
}

type updateBlockRequest struct {
	BlockType    *docs.BlockType  `json:"block_type"`
	Content      json.RawMessage  `json:"content"`
	Attrs        json.RawMessage  `json:"attrs"`
	BaseRevision *int64           `json:"base_revision"`
	OperationID  string           `json:"operation_id"`
}

type moveBlockRequest struct {
	AfterBlockID  *string `json:"after_block_id"`
	BeforeBlockID *string `json:"before_block_id"`
	OperationID   string  `json:"operation_id"`
}

type importDocumentRequest struct {
	Title    string `json:"title"`
	Markdown string `json:"markdown"`
}

func (h *Handlers) ListDocuments(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	q := docs.ListDocumentsQuery{
		UserID:   user.ID,
		Query:    r.URL.Query().Get("q"),
		Sort:     r.URL.Query().Get("sort"),
		Order:    r.URL.Query().Get("order"),
		Trash:    r.URL.Query().Get("trash") == "1" || r.URL.Query().Get("trash") == "true",
		Favorite: r.URL.Query().Get("favorite") == "1" || r.URL.Query().Get("favorite") == "true",
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			q.Limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			q.Offset = n
		}
	}
	result, err := h.Docs.ListDocuments(r.Context(), q)
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"documents": result.Documents,
		"total":     result.Total,
		"limit":     result.Limit,
		"offset":    result.Offset,
	})
}

func (h *Handlers) CreateDocument(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	var req createDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	doc, err := h.Docs.CreateDocument(r.Context(), docs.CreateDocumentInput{
		ActorID:  user.ID,
		Title:    req.Title,
		FolderID: req.FolderID,
	})
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"document": doc})
}

func (h *Handlers) GetDocument(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}
	doc, err := h.Docs.GetDocument(r.Context(), user.ID, docID)
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"document": doc})
}

func (h *Handlers) PatchDocument(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}
	var req patchDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	doc, err := h.Docs.PatchDocument(r.Context(), docs.PatchDocumentInput{
		ActorID:     user.ID,
		DocumentID:  docID,
		Title:       req.Title,
		IsFavorite:  req.IsFavorite,
		FolderID:    req.FolderID,
		ClearFolder: req.ClearFolder != nil && *req.ClearFolder,
	})
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"document": doc})
}

func (h *Handlers) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}
	if err := h.Docs.DeleteDocument(r.Context(), user.ID, docID); err != nil {
		h.writeDocsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handlers) RestoreDocument(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}
	doc, err := h.Docs.RestoreDocument(r.Context(), user.ID, docID)
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"document": doc})
}

func (h *Handlers) ImportDocument(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	var req importDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if strings.TrimSpace(req.Markdown) == "" {
		writeError(w, http.StatusBadRequest, "invalid_input", "markdown required")
		return
	}
	blocks, err := markdown.Parse(req.Markdown)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_markdown", err.Error())
		return
	}
	doc, err := h.Docs.ImportMarkdown(r.Context(), docs.ImportDocumentInput{
		ActorID:  user.ID,
		Title:    req.Title,
		Markdown: req.Markdown,
	}, blocks)
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"document": doc})
}

func (h *Handlers) ExportDocument(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}
	doc, blocks, err := h.Docs.LoadBlocksForExport(r.Context(), user.ID, docID)
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	md, err := markdown.Export(blocks)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "export_failed", err.Error())
		return
	}
	filename := sanitizeFilename(doc.Title) + ".md"
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(md))
}

func (h *Handlers) SyncContent(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}
	var req struct {
		BaseRevision *int64                   `json:"base_revision"`
		OperationID  string                   `json:"operation_id"`
		Blocks       []docs.ContentBlockInput `json:"blocks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if req.Blocks == nil {
		req.Blocks = []docs.ContentBlockInput{}
	}
	doc, blocks, err := h.Docs.SyncContent(r.Context(), docs.SyncContentInput{
		ActorID:      user.ID,
		DocumentID:   docID,
		BaseRevision: req.BaseRevision,
		OperationID:  req.OperationID,
		Blocks:       req.Blocks,
	})
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"document": doc,
		"blocks":   blocks,
	})
}

func (h *Handlers) ListBlocks(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}
	blocks, err := h.Docs.ListBlocks(r.Context(), user.ID, docID)
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	if blocks == nil {
		blocks = []docs.Block{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"blocks": blocks})
}

func (h *Handlers) ListBlockIndex(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}
	doc, index, err := h.Docs.ListBlockIndex(r.Context(), user.ID, docID)
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"document": doc,
		"revision": doc.Revision,
		"blocks":   index,
	})
}

func (h *Handlers) ListTOC(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}
	toc, err := h.Docs.ListTOC(r.Context(), user.ID, docID)
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	if toc == nil {
		toc = []docs.TOCItem{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"toc": toc})
}

func (h *Handlers) ListBlocksBatch(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if len(req.IDs) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"blocks": []docs.Block{}})
		return
	}
	blocks, err := h.Docs.ListBlocksByIDs(r.Context(), user.ID, docID, req.IDs)
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	if blocks == nil {
		blocks = []docs.Block{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"blocks": blocks})
}

func (h *Handlers) ListBlocksWindow(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}
	q := docs.BlocksWindowQuery{
		ActorID:    user.ID,
		DocumentID: docID,
		AfterKey:   r.URL.Query().Get("after_key"),
		AnchorID:   r.URL.Query().Get("anchor_id"),
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			q.Limit = n
		}
	}
	if v := r.URL.Query().Get("before"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			q.Before = n
		}
	}
	if v := r.URL.Query().Get("after"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			q.After = n
		}
	}
	blocks, err := h.Docs.ListBlocksWindow(r.Context(), q)
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	if blocks == nil {
		blocks = []docs.Block{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"blocks": blocks})
}

func (h *Handlers) GetBlock(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}
	blockID := chi.URLParam(r, "blockId")
	block, err := h.Docs.GetBlock(r.Context(), user.ID, docID, blockID)
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"block": block})
}

func (h *Handlers) CreateBlock(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}
	var req createBlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	block, doc, err := h.Docs.CreateBlock(r.Context(), docs.CreateBlockInput{
		ActorID:       user.ID,
		DocumentID:    docID,
		BlockType:     req.BlockType,
		Content:       req.Content,
		Attrs:         req.Attrs,
		AfterBlockID:  req.AfterBlockID,
		BeforeBlockID: req.BeforeBlockID,
		OperationID:   req.OperationID,
	})
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"block": block, "document": doc})
}

func (h *Handlers) UpdateBlock(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}
	blockID := chi.URLParam(r, "blockId")
	var req updateBlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	block, doc, err := h.Docs.UpdateBlock(r.Context(), docs.UpdateBlockInput{
		ActorID:      user.ID,
		DocumentID:   docID,
		BlockID:      blockID,
		BlockType:    req.BlockType,
		Content:      req.Content,
		Attrs:        req.Attrs,
		BaseRevision: req.BaseRevision,
		OperationID:  req.OperationID,
	})
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"block": block, "document": doc})
}

func (h *Handlers) DeleteBlock(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}
	blockID := chi.URLParam(r, "blockId")
	var req struct {
		OperationID string `json:"operation_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	doc, err := h.Docs.DeleteBlock(r.Context(), docs.DeleteBlockInput{
		ActorID:     user.ID,
		DocumentID:  docID,
		BlockID:     blockID,
		OperationID: req.OperationID,
	})
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"document": doc})
}

func (h *Handlers) MoveBlock(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}
	blockID := chi.URLParam(r, "blockId")
	var req moveBlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	block, doc, err := h.Docs.MoveBlock(r.Context(), docs.MoveBlockInput{
		ActorID:       user.ID,
		DocumentID:    docID,
		BlockID:       blockID,
		AfterBlockID:  req.AfterBlockID,
		BeforeBlockID: req.BeforeBlockID,
		OperationID:   req.OperationID,
	})
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"block": block, "document": doc})
}

func (h *Handlers) ListChanges(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}
	var afterSeq int64
	if v := r.URL.Query().Get("after_seq"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_input", "invalid after_seq")
			return
		}
		afterSeq = n
	}
	ops, err := h.Docs.ListChanges(r.Context(), user.ID, docID, afterSeq)
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	if ops == nil {
		ops = []docs.Operation{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"changes": ops})
}

func (h *Handlers) actorDoc(w http.ResponseWriter, r *http.Request) (auth.User, string, bool) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return auth.User{}, "", false
	}
	docID := chi.URLParam(r, "id")
	if docID == "" {
		writeError(w, http.StatusBadRequest, "invalid_input", "document id required")
		return auth.User{}, "", false
	}
	return user, docID, true
}

func (h *Handlers) writeDocsError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, docs.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
	case errors.Is(err, docs.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "permission denied")
	case errors.Is(err, docs.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, docs.ErrConflict):
		writeError(w, http.StatusConflict, "conflict", "resource conflict")
	case errors.Is(err, docs.ErrVersionConflict):
		writeError(w, http.StatusConflict, "version_conflict", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func sanitizeFilename(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return "document"
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "", "?", "", "\"", "", "<", "", ">", "", "|", "")
	out := replacer.Replace(title)
	if out == "" {
		return "document"
	}
	return out
}
