package collab

import "encoding/json"

// Wire protocol: JSON envelopes; binary Yjs/awareness payloads are base64.

const (
	TypeHello      = "hello"
	TypeSync       = "sync"
	TypeSyncDone   = "sync_done"
	TypeUpdate     = "update"
	TypeAwareness  = "awareness"
	TypeSnapshot   = "snapshot"
	TypeError      = "error"
)

type UserInfo struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

type UpdateItem struct {
	Seq         int64  `json:"seq"`
	OperationID string `json:"operation_id"`
	Update      string `json:"update"` // base64
}

type Message struct {
	Type string `json:"type"`

	DocID   string    `json:"doc_id,omitempty"`
	User    *UserInfo `json:"user,omitempty"`
	Empty   bool      `json:"empty,omitempty"`
	CanEdit bool      `json:"can_edit,omitempty"`

	Snapshot  string       `json:"snapshot,omitempty"` // base64
	Updates   []UpdateItem `json:"updates,omitempty"`
	UpdateSeq int64        `json:"update_seq,omitempty"`

	OperationID string `json:"operation_id,omitempty"`
	Update      string `json:"update,omitempty"` // base64
	Payload     string `json:"payload,omitempty"` // base64 awareness

	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func EncodeMessage(msg Message) ([]byte, error) {
	return json.Marshal(msg)
}

func DecodeMessage(data []byte) (Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return msg, err
}
