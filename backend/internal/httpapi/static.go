package httpapi

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// SPAFileServer serves files from root and falls back to index.html for
// client-side routes. Paths under {basePath}/api are never handled here.
// basePath is the public mount prefix (e.g. "/yunque") or empty for root.
func SPAFileServer(root, basePath string) http.Handler {
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

		urlPath := r.URL.Path
		if basePath != "" {
			switch {
			case urlPath == basePath:
				urlPath = "/"
			case strings.HasPrefix(urlPath, basePath+"/"):
				urlPath = urlPath[len(basePath):]
			default:
				http.NotFound(w, r)
				return
			}
		}

		rel := strings.TrimPrefix(filepath.Clean(urlPath), "/")
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
			r2 := r.Clone(r.Context())
			u := *r.URL
			u.Path = urlPath
			r2.URL = &u
			fileServer.ServeHTTP(w, r2)
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
