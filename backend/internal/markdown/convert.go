package markdown

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"

	"gitee.com/yunque/yunque/backend/internal/docs"
	"gitee.com/yunque/yunque/backend/internal/safeurl"
)

func parserEngine() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	)
}

type inlineNode struct {
	Type  string         `json:"type"`
	Text  string         `json:"text,omitempty"`
	Marks []markNode     `json:"marks,omitempty"`
	Attrs map[string]any `json:"attrs,omitempty"`
}

type markNode struct {
	Type  string         `json:"type"`
	Attrs map[string]any `json:"attrs,omitempty"`
}

type paragraphContent struct {
	Inlines []inlineNode `json:"inlines"`
}

type headingContent struct {
	Inlines []inlineNode `json:"inlines"`
}

type listContent struct {
	Items []listItem `json:"items"`
}

type listItem struct {
	Inlines  []inlineNode `json:"inlines"`
	Children []listItem   `json:"children,omitempty"`
}

type codeContent struct {
	Text string `json:"text"`
}

type quoteContent struct {
	Blocks []map[string]any `json:"blocks"`
}

type imageContent struct {
	Src string `json:"src"`
	Alt string `json:"alt"`
}

type tableContent struct {
	Rows [][][]inlineNode `json:"rows"`
}

type rawContent struct {
	Markdown string `json:"markdown"`
}

func Parse(md string) ([]docs.Block, error) {
	source := []byte(md)
	p := parserEngine().Parser()
	node := p.Parse(text.NewReader(source))

	var blocks []docs.Block
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		bs, err := convertNode(child, source)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, bs...)
	}
	if len(blocks) == 0 {
		content, _ := json.Marshal(paragraphContent{Inlines: []inlineNode{}})
		blocks = append(blocks, docs.Block{
			BlockType: docs.BlockParagraph,
			Content:   content,
			Attrs:     json.RawMessage(`{}`),
		})
	}
	return blocks, nil
}

func Export(blocks []docs.Block) (string, error) {
	var b strings.Builder
	for i, block := range blocks {
		part, err := exportBlock(block)
		if err != nil {
			return "", err
		}
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(part)
	}
	if b.Len() > 0 {
		b.WriteString("\n")
	}
	return b.String(), nil
}

func convertNode(node ast.Node, source []byte) ([]docs.Block, error) {
	switch n := node.(type) {
	case *ast.Heading:
		inlines := collectInlines(n, source)
		content, _ := json.Marshal(headingContent{Inlines: inlines})
		attrs, _ := json.Marshal(map[string]any{"level": n.Level})
		return []docs.Block{{
			BlockType: docs.BlockHeading,
			Content:   content,
			Attrs:     attrs,
		}}, nil
	case *ast.Paragraph:
		inlines := collectInlines(n, source)
		if len(inlines) == 1 && inlines[0].Type == "image" {
			src, _ := inlines[0].Attrs["src"].(string)
			alt, _ := inlines[0].Attrs["alt"].(string)
			content, _ := json.Marshal(imageContent{Src: src, Alt: alt})
			return []docs.Block{{
				BlockType: docs.BlockImage,
				Content:   content,
				Attrs:     json.RawMessage(`{}`),
			}}, nil
		}
		content, _ := json.Marshal(paragraphContent{Inlines: inlines})
		return []docs.Block{{
			BlockType: docs.BlockParagraph,
			Content:   content,
			Attrs:     json.RawMessage(`{}`),
		}}, nil
	case *ast.List:
		items := collectListItems(n, source)
		content, _ := json.Marshal(listContent{Items: items})
		bt := docs.BlockBulletList
		if n.IsOrdered() {
			bt = docs.BlockOrderedList
		}
		return []docs.Block{{
			BlockType: bt,
			Content:   content,
			Attrs:     json.RawMessage(`{}`),
		}}, nil
	case *ast.FencedCodeBlock:
		var buf bytes.Buffer
		for i := 0; i < n.Lines().Len(); i++ {
			line := n.Lines().At(i)
			buf.Write(line.Value(source))
		}
		lang := string(n.Language(source))
		content, _ := json.Marshal(codeContent{Text: buf.String()})
		attrs, _ := json.Marshal(map[string]any{"language": lang})
		return []docs.Block{{
			BlockType: docs.BlockCodeBlock,
			Content:   content,
			Attrs:     attrs,
		}}, nil
	case *ast.CodeBlock:
		var buf bytes.Buffer
		for i := 0; i < n.Lines().Len(); i++ {
			line := n.Lines().At(i)
			buf.Write(line.Value(source))
		}
		content, _ := json.Marshal(codeContent{Text: buf.String()})
		return []docs.Block{{
			BlockType: docs.BlockCodeBlock,
			Content:   content,
			Attrs:     json.RawMessage(`{"language":""}`),
		}}, nil
	case *ast.Blockquote:
		var inner []map[string]any
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			bs, err := convertNode(c, source)
			if err != nil {
				return nil, err
			}
			for _, b := range bs {
				var content any
				_ = json.Unmarshal(b.Content, &content)
				var attrs any
				_ = json.Unmarshal(b.Attrs, &attrs)
				inner = append(inner, map[string]any{
					"block_type": b.BlockType,
					"content":    content,
					"attrs":      attrs,
				})
			}
		}
		content, _ := json.Marshal(quoteContent{Blocks: inner})
		return []docs.Block{{
			BlockType: docs.BlockBlockquote,
			Content:   content,
			Attrs:     json.RawMessage(`{}`),
		}}, nil
	case *ast.ThematicBreak:
		return []docs.Block{{
			BlockType: docs.BlockThematicBreak,
			Content:   json.RawMessage(`{}`),
			Attrs:     json.RawMessage(`{}`),
		}}, nil
	case *ast.HTMLBlock:
		raw := extractSegment(n, source)
		content, _ := json.Marshal(rawContent{Markdown: raw})
		return []docs.Block{{
			BlockType: docs.BlockRawMarkdown,
			Content:   content,
			Attrs:     json.RawMessage(`{"reason":"html_block"}`),
		}}, nil
	case *east.Table:
		rows := collectTableRows(n, source)
		content, _ := json.Marshal(tableContent{Rows: rows})
		return []docs.Block{{
			BlockType: docs.BlockTable,
			Content:   content,
			Attrs:     json.RawMessage(`{}`),
		}}, nil
	default:
		raw := extractSegment(node, source)
		if strings.TrimSpace(raw) == "" {
			return nil, nil
		}
		content, _ := json.Marshal(rawContent{Markdown: raw})
		return []docs.Block{{
			BlockType: docs.BlockRawMarkdown,
			Content:   content,
			Attrs:     json.RawMessage(fmt.Sprintf(`{"reason":"unknown_node","type":%q}`, node.Kind().String())),
		}}, nil
	}
}

func collectInlines(node ast.Node, source []byte) []inlineNode {
	var out []inlineNode
	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := n.(type) {
		case *ast.Text:
			txt := string(t.Segment.Value(source))
			if txt == "" {
				return ast.WalkContinue, nil
			}
			out = append(out, inlineNode{Type: "text", Text: txt, Marks: collectMarks(t)})
			return ast.WalkContinue, nil
		case *ast.String:
			txt := string(t.Value)
			if txt == "" {
				return ast.WalkContinue, nil
			}
			out = append(out, inlineNode{Type: "text", Text: txt, Marks: collectMarks(t)})
			return ast.WalkContinue, nil
		case *ast.CodeSpan:
			out = append(out, inlineNode{
				Type:  "text",
				Text:  collectPlainText(t, source),
				Marks: []markNode{{Type: "code"}},
			})
			return ast.WalkSkipChildren, nil
		case *ast.Link:
			label := collectPlainText(t, source)
			href := safeurl.Link(string(t.Destination))
			marks := []markNode{}
			if href != "" {
				marks = append(marks, markNode{
					Type:  "link",
					Attrs: map[string]any{"href": href},
				})
			}
			out = append(out, inlineNode{
				Type:  "text",
				Text:  label,
				Marks: marks,
			})
			return ast.WalkSkipChildren, nil
		case *ast.Image:
			src := safeurl.Image(string(t.Destination))
			out = append(out, inlineNode{
				Type: "image",
				Attrs: map[string]any{
					"src": src,
					"alt": collectPlainText(t, source),
				},
			})
			return ast.WalkSkipChildren, nil
		case *ast.AutoLink:
			dest := safeurl.Link(string(t.URL(source)))
			marks := []markNode{}
			if dest != "" {
				marks = append(marks, markNode{
					Type:  "link",
					Attrs: map[string]any{"href": dest},
				})
			}
			out = append(out, inlineNode{
				Type:  "text",
				Text:  string(t.URL(source)),
				Marks: marks,
			})
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	return out
}

func collectMarks(n ast.Node) []markNode {
	var marks []markNode
	for p := n.Parent(); p != nil; p = p.Parent() {
		switch t := p.(type) {
		case *ast.Emphasis:
			if t.Level >= 2 {
				marks = append(marks, markNode{Type: "bold"})
			} else {
				marks = append(marks, markNode{Type: "italic"})
			}
		case *east.Strikethrough:
			marks = append(marks, markNode{Type: "strike"})
		}
	}
	return marks
}

func collectPlainText(node ast.Node, source []byte) string {
	var b strings.Builder
	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			switch t := n.(type) {
			case *ast.Text:
				b.Write(t.Segment.Value(source))
			case *ast.String:
				b.Write(t.Value)
			}
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}

func collectListItems(list *ast.List, source []byte) []listItem {
	var items []listItem
	for c := list.FirstChild(); c != nil; c = c.NextSibling() {
		li, ok := c.(*ast.ListItem)
		if !ok {
			continue
		}
		item := listItem{}
		for child := li.FirstChild(); child != nil; child = child.NextSibling() {
			switch n := child.(type) {
			case *ast.Paragraph, *ast.TextBlock:
				item.Inlines = append(item.Inlines, collectInlines(n, source)...)
			case *ast.List:
				nested := collectListItems(n, source)
				item.Children = append(item.Children, nested...)
			}
		}
		items = append(items, item)
	}
	return items
}

func collectTableRows(table *east.Table, source []byte) [][][]inlineNode {
	var rows [][][]inlineNode
	for c := table.FirstChild(); c != nil; c = c.NextSibling() {
		var cellsParent ast.Node
		switch section := c.(type) {
		case *east.TableHeader:
			cellsParent = section
		case *east.TableRow:
			cellsParent = section
		default:
			continue
		}
		var row [][]inlineNode
		for cell := cellsParent.FirstChild(); cell != nil; cell = cell.NextSibling() {
			if _, ok := cell.(*east.TableCell); ok {
				row = append(row, collectInlines(cell, source))
			}
		}
		if len(row) > 0 {
			rows = append(rows, row)
		}
	}
	return rows
}

func extractSegment(node ast.Node, source []byte) string {
	if node.Lines().Len() == 0 {
		return ""
	}
	start := node.Lines().At(0).Start
	last := node.Lines().At(node.Lines().Len() - 1)
	end := last.Stop
	if start < 0 || end > len(source) || start >= end {
		return ""
	}
	return string(source[start:end])
}

func exportBlock(block docs.Block) (string, error) {
	switch block.BlockType {
	case docs.BlockHeading:
		var c headingContent
		_ = json.Unmarshal(block.Content, &c)
		level := 1
		var attrs map[string]any
		_ = json.Unmarshal(block.Attrs, &attrs)
		if v, ok := attrs["level"].(float64); ok {
			level = int(v)
		}
		if level < 1 {
			level = 1
		}
		if level > 6 {
			level = 6
		}
		return strings.Repeat("#", level) + " " + exportInlines(c.Inlines), nil
	case docs.BlockParagraph:
		var c paragraphContent
		_ = json.Unmarshal(block.Content, &c)
		return exportInlines(c.Inlines), nil
	case docs.BlockBulletList:
		var c listContent
		_ = json.Unmarshal(block.Content, &c)
		return exportList(c.Items, false, 0), nil
	case docs.BlockOrderedList:
		var c listContent
		_ = json.Unmarshal(block.Content, &c)
		return exportList(c.Items, true, 0), nil
	case docs.BlockCodeBlock:
		var c codeContent
		_ = json.Unmarshal(block.Content, &c)
		var attrs map[string]any
		_ = json.Unmarshal(block.Attrs, &attrs)
		lang, _ := attrs["language"].(string)
		txt := strings.TrimRight(c.Text, "\n")
		return fmt.Sprintf("```%s\n%s\n```", lang, txt), nil
	case docs.BlockBlockquote:
		var c quoteContent
		_ = json.Unmarshal(block.Content, &c)
		var parts []string
		for _, inner := range c.Blocks {
			sub, err := nestedBlock(inner)
			if err != nil {
				return "", err
			}
			part, err := exportBlock(sub)
			if err != nil {
				return "", err
			}
			for _, line := range strings.Split(part, "\n") {
				parts = append(parts, "> "+line)
			}
		}
		return strings.Join(parts, "\n"), nil
	case docs.BlockThematicBreak:
		return "---", nil
	case docs.BlockImage:
		var c imageContent
		_ = json.Unmarshal(block.Content, &c)
		return fmt.Sprintf("![%s](%s)", c.Alt, c.Src), nil
	case docs.BlockTable:
		var c tableContent
		_ = json.Unmarshal(block.Content, &c)
		return exportTable(c.Rows), nil
	case docs.BlockRawMarkdown:
		var c rawContent
		_ = json.Unmarshal(block.Content, &c)
		return c.Markdown, nil
	default:
		return "", fmt.Errorf("unsupported block type: %s", block.BlockType)
	}
}

func nestedBlock(inner map[string]any) (docs.Block, error) {
	bt, _ := inner["block_type"].(string)
	contentBytes, err := json.Marshal(inner["content"])
	if err != nil {
		return docs.Block{}, err
	}
	attrsBytes, err := json.Marshal(inner["attrs"])
	if err != nil {
		return docs.Block{}, err
	}
	if string(attrsBytes) == "null" {
		attrsBytes = []byte(`{}`)
	}
	return docs.Block{
		BlockType: docs.BlockType(bt),
		Content:   contentBytes,
		Attrs:     attrsBytes,
	}, nil
}

func exportInlines(inlines []inlineNode) string {
	var b strings.Builder
	for _, n := range inlines {
		if n.Type == "image" {
			src, _ := n.Attrs["src"].(string)
			alt, _ := n.Attrs["alt"].(string)
			b.WriteString(fmt.Sprintf("![%s](%s)", alt, src))
			continue
		}
		txt := n.Text
		for _, m := range n.Marks {
			switch m.Type {
			case "bold":
				txt = "**" + txt + "**"
			case "italic":
				txt = "*" + txt + "*"
			case "strike":
				txt = "~~" + txt + "~~"
			case "code":
				txt = "`" + txt + "`"
			case "link":
				href, _ := m.Attrs["href"].(string)
				txt = fmt.Sprintf("[%s](%s)", txt, href)
			}
		}
		b.WriteString(txt)
	}
	return b.String()
}

func exportList(items []listItem, ordered bool, depth int) string {
	var b strings.Builder
	for i, item := range items {
		indent := strings.Repeat("  ", depth)
		prefix := "- "
		if ordered {
			prefix = fmt.Sprintf("%d. ", i+1)
		}
		b.WriteString(indent + prefix + exportInlines(item.Inlines))
		if len(item.Children) > 0 {
			b.WriteString("\n")
			b.WriteString(exportList(item.Children, ordered, depth+1))
		}
		if i < len(items)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func exportTable(rows [][][]inlineNode) string {
	if len(rows) == 0 {
		return ""
	}
	var b strings.Builder
	for i, row := range rows {
		b.WriteString("|")
		for _, cell := range row {
			b.WriteString(" ")
			b.WriteString(exportInlines(cell))
			b.WriteString(" |")
		}
		b.WriteString("\n")
		if i == 0 {
			b.WriteString("|")
			for range row {
				b.WriteString(" --- |")
			}
			b.WriteString("\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}
