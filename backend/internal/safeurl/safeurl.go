package safeurl

import (
	"net/url"
	"strings"
)

// Link returns href if it uses an allowed scheme (http, https, mailto), else "".
func Link(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	// Relative paths / fragments are safe for same-origin navigation.
	if strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "#") || strings.HasPrefix(raw, "./") || strings.HasPrefix(raw, "../") {
		if strings.ContainsAny(raw, "\r\n\x00") {
			return ""
		}
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" {
		return ""
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https", "mailto":
		return raw
	default:
		return ""
	}
}

// Image returns src if it is http(s) or a relative path, else "".
func Image(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "./") || strings.HasPrefix(raw, "../") {
		if strings.ContainsAny(raw, "\r\n\x00") {
			return ""
		}
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" {
		return ""
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return raw
	default:
		return ""
	}
}
