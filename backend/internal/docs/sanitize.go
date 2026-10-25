package docs

import (
	"encoding/json"

	"gitee.com/yunque/yunque/backend/internal/safeurl"
)

// SanitizeBlockContent removes dangerous link/image protocols from block JSON.
func SanitizeBlockContent(blockType BlockType, content json.RawMessage) json.RawMessage {
	if len(content) == 0 || string(content) == "null" {
		return content
	}
	var raw any
	if err := json.Unmarshal(content, &raw); err != nil {
		return content
	}
	cleaned := sanitizeValue(raw)
	if blockType == BlockImage {
		cleaned = sanitizeImageObject(cleaned)
	}
	out, err := json.Marshal(cleaned)
	if err != nil {
		return content
	}
	return out
}

func sanitizeImageObject(v any) any {
	m, ok := v.(map[string]any)
	if !ok {
		return v
	}
	if src, ok := m["src"].(string); ok {
		m["src"] = safeurl.Image(src)
	}
	return m
}

func sanitizeValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		if markType, _ := t["type"].(string); markType == "link" {
			if attrs, ok := t["attrs"].(map[string]any); ok {
				if href, ok := attrs["href"].(string); ok {
					attrs["href"] = safeurl.Link(href)
				}
			}
		}
		if nodeType, _ := t["type"].(string); nodeType == "image" {
			if attrs, ok := t["attrs"].(map[string]any); ok {
				if src, ok := attrs["src"].(string); ok {
					attrs["src"] = safeurl.Image(src)
				}
			}
			if src, ok := t["src"].(string); ok {
				t["src"] = safeurl.Image(src)
			}
		}
		if href, ok := t["href"].(string); ok {
			t["href"] = safeurl.Link(href)
		}
		if src, ok := t["src"].(string); ok && t["type"] != "text" {
			// Top-level image content uses src without type.
			if _, hasInlines := t["inlines"]; !hasInlines {
				if _, hasText := t["text"]; !hasText {
					t["src"] = safeurl.Image(src)
				}
			}
		}
		for k, child := range t {
			t[k] = sanitizeValue(child)
		}
		return t
	case []any:
		for i, child := range t {
			t[i] = sanitizeValue(child)
		}
		return t
	default:
		return v
	}
}
