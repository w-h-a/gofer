package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/w-h-a/gofer/internal/client/event_publisher/sse"
	"github.com/w-h-a/gofer/internal/client/repo"
	"github.com/w-h-a/gofer/internal/client/repo/sqlite"
	"github.com/w-h-a/gofer/internal/service"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	version := envOr("VERSION", "dev")

	res, err := newResource(ctx, version)
	if err != nil {
		slog.Error("failed to create resource", "error", err)
		os.Exit(1)
	}

	shutdownTracer, err := initTracer(ctx, res)
	if err != nil {
		slog.Error("failed to init tracer", "error", err)
		os.Exit(1)
	}

	shutdownLogger, err := initLogger(ctx, res)
	if err != nil {
		slog.Error("failed to init logger", "error", err)
		os.Exit(1)
	}

	repoLocation := envOr("REPO_LOCATION", "gofer.db")

	port := envOr("PORT", "8080")
	if _, err := strconv.Atoi(port); err != nil {
		slog.Error("invalid PORT", "value", port, "error", err)
		os.Exit(1)
	}

	ttl, err := time.ParseDuration(envOr("TTL", "48h"))
	if err != nil {
		slog.Error("invalid TTL", "error", err)
		os.Exit(1)
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
	h := newHandler(svc, ttl, version)

	addr := fmt.Sprintf(":%s", port)

	srv := &http.Server{
		Addr: addr,
		Handler: otelhttp.NewHandler(h.routes(), "gofer",
			otelhttp.WithFilter(func(r *http.Request) bool {
				if r.URL.Path == "/healthz" {
					return false
				}
				if strings.HasPrefix(r.URL.Path, "/api/bins/") && strings.HasSuffix(r.URL.Path, "/sse") {
					return false
				}
				return true
			}),
		),
	}

	go runCleanup(ctx, svc)

	go func() {
		slog.Info("starting server", "addr", srv.Addr, "version", version)
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
	}

	slog.Info("server stopped")

	if err := shutdownTracer(shutdownCtx); err != nil {
		slog.Error("tracer shutdown error", "error", err)
	}

	if err := shutdownLogger(shutdownCtx); err != nil {
		slog.Error("logger shutdown error", "error", err)
	}
}

func runCleanup(ctx context.Context, svc *service.Service) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			out, err := svc.CleanupExpiredBins(ctx)
			if err != nil {
				slog.ErrorContext(ctx, "cleanup failed", "error", err)
				continue
			}
			if out.Deleted > 0 {
				slog.InfoContext(ctx, "cleanup completed", "deleted", out.Deleted)
			}
		}
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
