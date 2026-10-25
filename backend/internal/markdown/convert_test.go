package markdown

import (
	"strings"
	"testing"
)

func TestRoundTripBasicMarkdown(t *testing.T) {
	input := strings.TrimSpace(`
# Title

Hello **world** and *italic*.

- item one
- item two

`+"```go\nfmt.Println(\"hi\")\n```"+`

> quoted

---

| A | B |
| --- | --- |
| 1 | 2 |
`) + "\n"

	blocks, err := Parse(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(blocks) < 6 {
		t.Fatalf("expected multiple blocks, got %d", len(blocks))
	}

	out, err := Export(blocks)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if !strings.Contains(out, "# Title") {
		t.Fatalf("missing title in export: %s", out)
	}
	if !strings.Contains(out, "**world**") {
		t.Fatalf("missing bold in export: %s", out)
	}
	if !strings.Contains(out, "item one") {
		t.Fatalf("missing list item in export: %s", out)
	}
	if !strings.Contains(out, "```go") {
		t.Fatalf("missing code fence: %s", out)
	}
	if !strings.Contains(out, "| A |") {
		t.Fatalf("missing table: %s", out)
	}

	blocks2, err := Parse(out)
	if err != nil {
		t.Fatalf("reparse: %v", err)
	}
	if len(blocks2) != len(blocks) {
		t.Fatalf("block count changed: %d -> %d\nexport:\n%s", len(blocks), len(blocks2), out)
	}
}

func TestParseEmptyCreatesParagraph(t *testing.T) {
	blocks, err := Parse("")
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 1 {
		t.Fatalf("got %d blocks", len(blocks))
	}
}

func TestTightListItems(t *testing.T) {
	blocks, err := Parse("- a\n- b\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 1 || blocks[0].BlockType != "bullet_list" {
		t.Fatalf("unexpected blocks: %+v", blocks)
	}
	out, err := Export(blocks)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "- a") || !strings.Contains(out, "- b") {
		t.Fatalf("export missing items: %s", out)
	}
}
