package collab

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"sync"
)

var ErrForbidden = errors.New("forbidden")

type room struct {
	mu      sync.Mutex
	clients map[*Client]struct{}
}

type Client struct {
	send    chan Message
	userID  string
	canEdit bool
	closed  chan struct{}
}

func (c *Client) Send(msg Message) error {
	select {
	case <-c.closed:
		return context.Canceled
	case c.send <- msg:
		return nil
	default:
		select {
		case c.send <- msg:
			return nil
		default:
			return nil
		}
	}
}

func (c *Client) UserID() string          { return c.userID }
func (c *Client) CanEdit() bool           { return c.canEdit }
func (c *Client) Outgoing() <-chan Message { return c.send }
func (c *Client) Done() <-chan struct{}   { return c.closed }

type Hub struct {
	store  *Store
	logger *slog.Logger

	mu    sync.RWMutex
	rooms map[string]*room
}

func NewHub(store *Store, logger *slog.Logger) *Hub {
	if logger == nil {
		logger = slog.Default()
	}
	return &Hub{
		store:  store,
		logger: logger,
		rooms:  make(map[string]*room),
	}
}

func (h *Hub) Store() *Store { return h.store }

func (h *Hub) getOrCreateRoom(docID string) *room {
	h.mu.Lock()
	defer h.mu.Unlock()
	r, ok := h.rooms[docID]
	if !ok {
		r = &room{clients: make(map[*Client]struct{})}
		h.rooms[docID] = r
	}
	return r
}

func (h *Hub) join(docID string, c *Client) {
	r := h.getOrCreateRoom(docID)
	r.mu.Lock()
	r.clients[c] = struct{}{}
	r.mu.Unlock()
}

func (h *Hub) leave(docID string, c *Client) {
	h.mu.Lock()
	r, ok := h.rooms[docID]
	if !ok {
		h.mu.Unlock()
		return
	}
	r.mu.Lock()
	delete(r.clients, c)
	empty := len(r.clients) == 0
	r.mu.Unlock()
	if empty {
		delete(h.rooms, docID)
	}
	h.mu.Unlock()
	select {
	case <-c.closed:
	default:
		close(c.closed)
	}
}

func (h *Hub) RoomSize(docID string) int {
	h.mu.RLock()
	r := h.rooms[docID]
	h.mu.RUnlock()
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.clients)
}

func (h *Hub) broadcast(docID string, msg Message, except *Client) {
	h.mu.RLock()
	r := h.rooms[docID]
	h.mu.RUnlock()
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for c := range r.clients {
		if c == except {
			continue
		}
		_ = c.Send(msg)
	}
}

func (h *Hub) BuildSyncMessage(ctx context.Context, documentID string) (Message, bool, error) {
	snap, updates, empty, err := h.store.LoadSyncState(ctx, documentID)
	if err != nil {
		return Message{}, false, err
	}
	msg := Message{Type: TypeSync, Updates: make([]UpdateItem, 0, len(updates))}
	if snap != nil && len(snap.Blob) > 0 {
		msg.Snapshot = base64.StdEncoding.EncodeToString(snap.Blob)
		msg.UpdateSeq = snap.UpdateSeq
	}
	for _, u := range updates {
		msg.Updates = append(msg.Updates, UpdateItem{
			Seq:         u.Seq,
			OperationID: u.OperationID,
			Update:      base64.StdEncoding.EncodeToString(u.Blob),
		})
	}
	return msg, empty, nil
}

func (h *Hub) HandleUpdate(ctx context.Context, documentID, actorID string, msg Message, from *Client) error {
	if !from.CanEdit() {
		return ErrForbidden
	}
	blob, err := base64.StdEncoding.DecodeString(msg.Update)
	if err != nil {
		return err
	}
	stored, err := h.store.AppendUpdate(ctx, documentID, msg.OperationID, actorID, blob)
	if err != nil {
		if errors.Is(err, ErrDuplicateOperation) {
			return nil
		}
		return err
	}
	out := Message{
		Type:        TypeUpdate,
		OperationID: stored.OperationID,
		Update:      msg.Update,
		UpdateSeq:   stored.Seq,
	}
	h.broadcast(documentID, out, from)
	return nil
}

func (h *Hub) HandleAwareness(documentID string, msg Message, from *Client) {
	h.broadcast(documentID, Message{
		Type:    TypeAwareness,
		Payload: msg.Payload,
	}, from)
}

func (h *Hub) HandleSnapshot(ctx context.Context, documentID string, msg Message, from *Client) error {
	if !from.CanEdit() {
		return ErrForbidden
	}
	blob, err := base64.StdEncoding.DecodeString(msg.Snapshot)
	if err != nil {
		return err
	}
	seq := msg.UpdateSeq
	if seq == 0 {
		seq, err = h.store.MaxSeq(ctx, documentID)
		if err != nil {
			return err
		}
	}
	return h.store.UpsertSnapshot(ctx, documentID, blob, seq)
}

// AttachClient registers a client for broadcasting; caller owns the send loop.
func (h *Hub) AttachClient(docID, userID string, canEdit bool) *Client {
	c := &Client{
		send:    make(chan Message, 64),
		userID:  userID,
		canEdit: canEdit,
		closed:  make(chan struct{}),
	}
	h.join(docID, c)
	return c
}

func (h *Hub) DetachClient(docID string, c *Client) {
	h.leave(docID, c)
}
