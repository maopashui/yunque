package docs

import (
	"encoding/json"
	"time"
)

const FormatVersion = 1

type Role string

const (
	RoleOwner  Role = "owner"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

func (r Role) CanRead() bool {
	return r == RoleOwner || r == RoleEditor || r == RoleViewer
}

func (r Role) CanEdit() bool {
	return r == RoleOwner || r == RoleEditor
}

func (r Role) CanManage() bool {
	return r == RoleOwner
}

type Document struct {
	ID            string     `json:"id"`
	OwnerID       string     `json:"owner_id"`
	OwnerName     string     `json:"owner_name,omitempty"`
	FolderID      *string    `json:"folder_id,omitempty"`
	Title         string     `json:"title"`
	FormatVersion int        `json:"format_version"`
	Revision      int64      `json:"revision"`
	BlockCount    int        `json:"block_count"`
	IsFavorite    bool       `json:"is_favorite"`
	Role          Role       `json:"role,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
}

type BlockType string

const (
	BlockHeading        BlockType = "heading"
	BlockParagraph      BlockType = "paragraph"
	BlockBulletList     BlockType = "bullet_list"
	BlockOrderedList    BlockType = "ordered_list"
	BlockCodeBlock      BlockType = "code_block"
	BlockTable          BlockType = "table"
	BlockBlockquote     BlockType = "blockquote"
	BlockImage          BlockType = "image"
	BlockThematicBreak  BlockType = "thematic_break"
	BlockRawMarkdown    BlockType = "raw_markdown"
)

var ValidBlockTypes = map[BlockType]bool{
	BlockHeading:       true,
	BlockParagraph:     true,
	BlockBulletList:    true,
	BlockOrderedList:   true,
	BlockCodeBlock:     true,
	BlockTable:         true,
	BlockBlockquote:    true,
	BlockImage:         true,
	BlockThematicBreak: true,
	BlockRawMarkdown:   true,
}

type Block struct {
	ID         string          `json:"id"`
	DocumentID string          `json:"document_id"`
	OrderKey   string          `json:"order_key"`
	BlockType  BlockType       `json:"block_type"`
	Content    json.RawMessage `json:"content"`
	Attrs      json.RawMessage `json:"attrs"`
	Revision   int64           `json:"revision"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	DeletedAt  *time.Time      `json:"deleted_at,omitempty"`
}

type Operation struct {
	Seq           int64           `json:"seq"`
	DocumentID    string          `json:"document_id"`
	OperationID   string          `json:"operation_id"`
	ActorID       string          `json:"actor_id"`
	OperationType string          `json:"operation_type"`
	Payload       json.RawMessage `json:"payload"`
	CreatedAt     time.Time       `json:"created_at"`
}

type ListDocumentsQuery struct {
	UserID   string
	Query    string
	Sort     string // updated_at | title | created_at
	Order    string // asc | desc
	Trash    bool
	Favorite bool
	Limit    int
	Offset   int
}

// BlockMeta is a lightweight index entry without content payload.
type BlockMeta struct {
	ID        string          `json:"id"`
	OrderKey  string          `json:"order_key"`
	BlockType BlockType       `json:"block_type"`
	Attrs     json.RawMessage `json:"attrs"`
	Index     int             `json:"index"`
}

type TOCItem struct {
	ID       string `json:"id"`
	OrderKey string `json:"order_key"`
	Index    int    `json:"index"`
	Level    int    `json:"level"`
	Text     string `json:"text"`
}
