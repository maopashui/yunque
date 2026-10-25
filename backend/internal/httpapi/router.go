package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"gitee.com/yunque/yunque/backend/internal/auth"
	"gitee.com/yunque/yunque/backend/internal/collab"
	"gitee.com/yunque/yunque/backend/internal/config"
	"gitee.com/yunque/yunque/backend/internal/docs"
)

const (
	defaultJSONBodyLimit = 2 << 20 // 2 MiB
	importJSONBodyLimit  = 5 << 20 // 5 MiB
	authRateLimit        = 10
	authRateWindow       = time.Minute
)

func NewRouter(cfg config.Config, authSvc *auth.Service, docsSvc *docs.Service, collabHub *collab.Hub, logger *slog.Logger) http.Handler {
	h := &Handlers{
		Auth:    authSvc,
		Docs:    docsSvc,
		Collab:  collabHub,
		Config:  cfg,
		Version: cfg.Version,
	}

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(RequestLogger(logger))
	r.Use(chimw.Recoverer)
	r.Use(SecurityHeaders)
	r.Use(CORS(cfg.AllowedOrigin))

	r.Route("/api", func(r chi.Router) {
		r.Use(MaxBodyBytes(defaultJSONBodyLimit))
		r.Get("/health", h.Health)

		r.Route("/auth", func(r chi.Router) {
			r.Get("/registration", h.RegistrationStatus)
			authLimit := RateLimit(authRateLimit, authRateWindow)
			r.With(authLimit).Post("/register", h.Register)
			r.With(authLimit).Post("/login", h.Login)
			r.Post("/logout", h.Logout)
			r.With(RequireAuth(authSvc, cfg)).Get("/me", h.Me)
		})

		r.Group(func(r chi.Router) {
			r.Use(RequireAuth(authSvc, cfg))

			r.Route("/admin", func(r chi.Router) {
				r.Use(RequireAdmin)
				r.Get("/settings", h.GetAdminSettings)
				r.Patch("/settings", h.PatchAdminSettings)
				r.Get("/users", h.ListAdminUsers)
				r.Patch("/users/{userId}", h.PatchAdminUser)
				r.Delete("/users/{userId}", h.DeleteAdminUser)
			})

			r.Get("/documents", h.ListDocuments)
			r.Post("/documents", h.CreateDocument)
			r.With(MaxBodyBytes(importJSONBodyLimit)).Post("/documents/import", h.ImportDocument)

			r.Route("/documents/{id}", func(r chi.Router) {
				r.Get("/", h.GetDocument)
				r.Patch("/", h.PatchDocument)
				r.Delete("/", h.DeleteDocument)
				r.Post("/restore", h.RestoreDocument)
				r.Get("/export.md", h.ExportDocument)
				r.Get("/changes", h.ListChanges)

				r.Get("/snapshots", h.ListSnapshots)
				r.Post("/snapshots", h.CreateSnapshot)
				r.Post("/snapshots/cleanup", h.CleanupSnapshots)
				r.Get("/snapshots/{snapshotId}", h.GetSnapshot)
				r.Get("/snapshots/{snapshotId}/diff", h.CompareSnapshot)
				r.Post("/snapshots/{snapshotId}/restore", h.RestoreSnapshot)

				r.With(MaxBodyBytes(importJSONBodyLimit)).Put("/content", h.SyncContent)
				r.Get("/collab", h.CollabWS)
				r.Post("/assets", h.UploadDocumentAsset)
				r.Get("/assets/{assetId}", h.GetDocumentAsset)
				r.Get("/members", h.ListMembers)
				r.Post("/members", h.AddMember)
				r.Patch("/members/{userId}", h.UpdateMember)
				r.Delete("/members/{userId}", h.RemoveMember)
				r.Get("/blocks", h.ListBlocks)
				r.Get("/blocks/index", h.ListBlockIndex)
				r.Get("/blocks/window", h.ListBlocksWindow)
				r.Post("/blocks/batch", h.ListBlocksBatch)
				r.Get("/toc", h.ListTOC)
				r.Post("/blocks", h.CreateBlock)
				r.Get("/blocks/{blockId}", h.GetBlock)
				r.Put("/blocks/{blockId}", h.UpdateBlock)
				r.Delete("/blocks/{blockId}", h.DeleteBlock)
				r.Post("/blocks/{blockId}/move", h.MoveBlock)
			})
		})
	})

	if cfg.StaticDir != "" {
		spa := SPAFileServer(cfg.StaticDir)
		r.NotFound(spa.ServeHTTP)
		r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet || r.Method == http.MethodHead {
				spa.ServeHTTP(w, r)
				return
			}
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		})
	}

	return r
}
