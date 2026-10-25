package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"

	"gitee.com/yunque/yunque/backend/internal/auth"
	"gitee.com/yunque/yunque/backend/internal/config"
	"gitee.com/yunque/yunque/backend/internal/db"
	"gitee.com/yunque/yunque/backend/internal/docs"
)

func main() {
	blocks := flag.Int("blocks", 5000, "number of semantic blocks to generate")
	username := flag.String("user", "demo_user", "existing username that owns the document")
	title := flag.String("title", "大文档压测样例", "document title")
	flag.Parse()

	cfg := config.Load()
	sqlDB, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.Migrate(sqlDB, cfg.MigrationsDir); err != nil {
		log.Fatal(err)
	}

	userID, err := lookupUserID(sqlDB, *username)
	if err != nil {
		log.Fatalf("user %q not found; register via UI/API first: %v", *username, err)
	}

	store := docs.NewStore(sqlDB)
	svc := docs.NewService(store, nil)
	ctx := context.Background()
	doc, err := svc.CreateDocument(ctx, docs.CreateDocumentInput{
		ActorID: userID,
		Title:   *title,
	})
	if err != nil {
		log.Fatal(err)
	}

	now := time.Now().UTC()
	orderKey := ""
	err = store.WithTx(ctx, func(tx *sql.Tx) error {
		for i := 0; i < *blocks; i++ {
			orderKey = docs.OrderKeyAfter(orderKey)
			block, err := makeBlock(doc.ID, orderKey, i, now)
			if err != nil {
				return err
			}
			if err := store.InsertBlock(ctx, tx, block); err != nil {
				return err
			}
		}
		doc.BlockCount = *blocks
		doc.Revision = 1
		doc.UpdatedAt = now
		return store.UpdateDocument(ctx, tx, doc)
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("seeded document %s with %d blocks for user %s\n", doc.ID, *blocks, *username)
}

func lookupUserID(sqlDB *sql.DB, username string) (string, error) {
	store := auth.NewStore(sqlDB)
	user, err := store.GetUserByUsername(context.Background(), username)
	if err != nil {
		return "", err
	}
	return user.ID, nil
}

func makeBlock(docID, orderKey string, i int, now time.Time) (docs.Block, error) {
	var blockType docs.BlockType
	var content any
	var attrs any = map[string]any{}

	switch {
	case i%50 == 0:
		blockType = docs.BlockHeading
		level := 1 + (i/50)%3
		attrs = map[string]any{"level": level}
		content = map[string]any{
			"inlines": []map[string]any{{"type": "text", "text": fmt.Sprintf("章节 %d", i/50+1)}},
		}
	case i%37 == 0:
		blockType = docs.BlockCodeBlock
		attrs = map[string]any{"language": "go"}
		lines := make([]string, 80)
		for j := range lines {
			lines[j] = fmt.Sprintf("// line %d of block %d", j+1, i)
		}
		content = map[string]any{"text": strings.Join(lines, "\n") + "\n"}
	case i%41 == 0:
		blockType = docs.BlockTable
		rows := make([][][]map[string]any, 40)
		for r := range rows {
			rows[r] = [][]map[string]any{
				{{"type": "text", "text": fmt.Sprintf("r%d-c1", r)}},
				{{"type": "text", "text": fmt.Sprintf("r%d-c2", r)}},
				{{"type": "text", "text": fmt.Sprintf("r%d-c3", r)}},
			}
		}
		content = map[string]any{"rows": rows}
	default:
		blockType = docs.BlockParagraph
		content = map[string]any{
			"inlines": []map[string]any{{
				"type": "text",
				"text": fmt.Sprintf("段落 #%d — 云雀虚拟滚动测试内容，用于验证按需加载与视口渲染。", i+1),
			}},
		}
	}

	c, err := json.Marshal(content)
	if err != nil {
		return docs.Block{}, err
	}
	a, err := json.Marshal(attrs)
	if err != nil {
		return docs.Block{}, err
	}
	return docs.Block{
		ID:         uuid.NewString(),
		DocumentID: docID,
		OrderKey:   orderKey,
		BlockType:  blockType,
		Content:    c,
		Attrs:      a,
		Revision:   1,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}
