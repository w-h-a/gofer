package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/w-h-a/gofer/internal/client/event_publisher/sse"
	"github.com/w-h-a/gofer/internal/client/repo"
	"github.com/w-h-a/gofer/internal/client/repo/sqlite"
	"github.com/w-h-a/gofer/internal/service"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	repoLocation := os.Getenv("REPO_LOCATION")
	if repoLocation == "" {
		repoLocation = "gofer.db"
	}

	r, err := sqlite.NewRepo(repo.WithLocation(repoLocation))
	if err != nil {
		slog.Error("failed to create repo", "error", err)
		os.Exit(1)
	}

	p, err := sse.NewEventPublisher()
	if err != nil {
		slog.Error("failed to create event publisher", "error", err)
		os.Exit(1)
	}

	svc := service.NewService(r, p)
	h := newHandler(svc)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: h.routes(),
	}

	go func() {
		slog.Info("starting server", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}
