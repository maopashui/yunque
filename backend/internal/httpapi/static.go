package httpapi

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// SPAFileServer serves files from root and falls back to index.html for
// client-side routes. Paths under /api are never handled here.
func SPAFileServer(root string) http.Handler {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		absRoot = root
	}
	fileServer := http.FileServer(http.Dir(absRoot))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		rel := strings.TrimPrefix(filepath.Clean(r.URL.Path), "/")
		if rel == "." || rel == ".." {
			rel = ""
		}
		target := filepath.Join(absRoot, rel)
		if relPath, err := filepath.Rel(absRoot, target); err != nil || strings.HasPrefix(relPath, "..") {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		info, err := os.Stat(target)
		if err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		index := filepath.Join(absRoot, "index.html")
		if _, err := os.Stat(index); err != nil {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, index)
	})
}
