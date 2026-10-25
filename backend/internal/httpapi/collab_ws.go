package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"

	"gitee.com/yunque/yunque/backend/internal/collab"
)

func (h *Handlers) CollabWS(w http.ResponseWriter, r *http.Request) {
	if h.Collab == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "collab not configured")
		return
	}
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	docID := chi.URLParam(r, "id")
	doc, err := h.Docs.GetDocument(r.Context(), user.ID, docID)
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	if !doc.Role.CanRead() {
		writeError(w, http.StatusForbidden, "forbidden", "forbidden")
		return
	}
	canEdit := doc.Role.CanEdit()

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: websocketOriginPatterns(h.Config.AllowedOrigin, r),
	})
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	client := h.Collab.AttachClient(docID, user.ID, canEdit)
	defer h.Collab.DetachClient(docID, client)

	ctx := r.Context()
	hello := collab.Message{
		Type:    collab.TypeHello,
		DocID:   docID,
		CanEdit: canEdit,
		User: &collab.UserInfo{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
		},
	}
	syncMsg, empty, err := h.Collab.BuildSyncMessage(ctx, docID)
	if err != nil {
		_ = writeWSJSON(ctx, conn, collab.Message{Type: collab.TypeError, Code: "sync_failed", Message: err.Error()})
		return
	}
	hello.Empty = empty
	if err := writeWSJSON(ctx, conn, hello); err != nil {
		return
	}
	if err := writeWSJSON(ctx, conn, syncMsg); err != nil {
		return
	}

	errCh := make(chan error, 1)
	go func() {
		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case <-client.Done():
				errCh <- nil
				return
			case msg, ok := <-client.Outgoing():
				if !ok {
					errCh <- nil
					return
				}
				if err := writeWSJSON(ctx, conn, msg); err != nil {
					errCh <- err
					return
				}
			}
		}
	}()

	readDone := make(chan error, 1)
	go func() {
		for {
			_, data, err := conn.Read(ctx)
			if err != nil {
				readDone <- err
				return
			}
			msg, err := collab.DecodeMessage(data)
			if err != nil {
				_ = writeWSJSON(ctx, conn, collab.Message{
					Type: collab.TypeError, Code: "bad_message", Message: "invalid JSON",
				})
				continue
			}
			switch msg.Type {
			case collab.TypeSyncDone:
				// client ready
			case collab.TypeUpdate:
				if err := h.Collab.HandleUpdate(ctx, docID, user.ID, msg, client); err != nil {
					code := "update_failed"
					if errors.Is(err, collab.ErrForbidden) {
						code = "forbidden"
					}
					_ = writeWSJSON(ctx, conn, collab.Message{
						Type: collab.TypeError, Code: code, Message: err.Error(),
					})
				}
			case collab.TypeAwareness:
				h.Collab.HandleAwareness(docID, msg, client)
			case collab.TypeSnapshot:
				if err := h.Collab.HandleSnapshot(ctx, docID, msg, client); err != nil {
					_ = writeWSJSON(ctx, conn, collab.Message{
						Type: collab.TypeError, Code: "snapshot_failed", Message: err.Error(),
					})
				}
			default:
				_ = writeWSJSON(ctx, conn, collab.Message{
					Type: collab.TypeError, Code: "unknown_type", Message: msg.Type,
				})
			}
		}
	}()

	select {
	case <-readDone:
	case <-errCh:
	case <-ctx.Done():
	}
}

func writeWSJSON(ctx context.Context, conn *websocket.Conn, msg collab.Message) error {
	data, err := collab.EncodeMessage(msg)
	if err != nil {
		return err
	}
	writeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return conn.Write(writeCtx, websocket.MessageText, data)
}

// websocketOriginPatterns restricts browser WebSocket Origin checks.
// When AllowedOrigin is set, only that origin (and its host pattern) are accepted.
// When empty (same-origin Docker/production), the request Host is allowed.
func websocketOriginPatterns(allowedOrigin string, r *http.Request) []string {
	var patterns []string
	if allowedOrigin != "" {
		if u, err := url.Parse(allowedOrigin); err == nil && u.Host != "" {
			patterns = append(patterns, u.Host, allowedOrigin)
		} else {
			patterns = append(patterns, allowedOrigin)
		}
	}
	host := r.Host
	if host != "" {
		patterns = append(patterns, host)
		if !strings.Contains(host, "://") {
			patterns = append(patterns, "http://"+host, "https://"+host)
		}
	}
	if len(patterns) == 0 {
		return []string{"localhost", "127.0.0.1"}
	}
	return patterns
}
