package safeurl

import "testing"

func TestLink(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"https://example.com", "https://example.com"},
		{"http://example.com/a", "http://example.com/a"},
		{"mailto:a@b.com", "mailto:a@b.com"},
		{"/docs/1", "/docs/1"},
		{"#section", "#section"},
		{"javascript:alert(1)", ""},
		{"data:text/html,hi", ""},
		{"vbscript:msg", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := Link(c.in); got != c.want {
			t.Fatalf("Link(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

func TestImage(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"https://cdn.example/a.png", "https://cdn.example/a.png"},
		{"/api/documents/x/assets/y", "/api/documents/x/assets/y"},
		{"javascript:alert(1)", ""},
		{"data:image/png;base64,xx", ""},
		{"mailto:a@b.com", ""},
	}
	for _, c := range cases {
		if got := Image(c.in); got != c.want {
			t.Fatalf("Image(%q)=%q want %q", c.in, got, c.want)
		}
	}
}
