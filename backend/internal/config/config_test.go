package config

import "testing"

func TestNormalizeBasePath(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"/", ""},
		{"  /  ", ""},
		{"yunque", "/yunque"},
		{"/yunque", "/yunque"},
		{"/yunque/", "/yunque"},
		{" /yunque/ ", "/yunque"},
		{"/a/b/", "/a/b"},
	}
	for _, tc := range cases {
		got := NormalizeBasePath(tc.in)
		if got != tc.want {
			t.Errorf("NormalizeBasePath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestCookiePath(t *testing.T) {
	if (Config{}.CookiePath()) != "/" {
		t.Fatal("empty base cookie path")
	}
	if (Config{BasePath: "/yunque"}).CookiePath() != "/yunque" {
		t.Fatal("base cookie path")
	}
}
