package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gitee.com/yunque/yunque/backend/internal/auth"
	"gitee.com/yunque/yunque/backend/internal/collab"
	"gitee.com/yunque/yunque/backend/internal/config"
	"gitee.com/yunque/yunque/backend/internal/db"
	"gitee.com/yunque/yunque/backend/internal/docs"
	"gitee.com/yunque/yunque/backend/internal/httpapi"
	"gitee.com/yunque/yunque/backend/internal/userlookup"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := config.Load()

	sqlDB, err := db.Open(cfg.DBPath)
	if err != nil {
		logger.Error("open database", "error", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	if err := db.Migrate(sqlDB, cfg.MigrationsDir); err != nil {
		logger.Error("migrate database", "error", err)
		os.Exit(1)
	}

	authStore := auth.NewStore(sqlDB)
	authSvc := auth.NewService(authStore, cfg.SessionTTL)
	docsSvc := docs.NewService(docs.NewStore(sqlDB), userlookup.FromAuthStore(authStore))
	docsSvc.ConfigureAssets(cfg.AssetsDir, cfg.MaxAssetBytes)
	collabHub := collab.NewHub(collab.NewStore(sqlDB), logger)
	router := httpapi.NewRouter(cfg, authSvc, docsSvc, collabHub, logger)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("server starting", "addr", cfg.Addr, "version", cfg.Version)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server shutdown", "error", err)
		os.Exit(1)
	}
	logger.Info("server stopped")
}
