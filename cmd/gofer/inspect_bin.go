package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/w-h-a/gofer/internal/client/repo"
	"github.com/w-h-a/gofer/internal/domain"
	"github.com/w-h-a/gofer/internal/service"
)

type inspectBinData struct {
	Slug        string
	CreatedAt   string
	ExpiresAt   string
	CaptureURL  string
	CurlExample string
	Requests    []inspectBinRequest
	SSEEndpoint string
}

type inspectBinRequest struct {
	ID          string
	SequenceNum int
	Method      string
	Path        string
	ContentType string
	BodySize    int
	CapturedAt  string
}

func (h *handler) handleInspectBin(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	out, err := h.svc.ViewBin(r.Context(), service.ViewBinInput{
		Slug: slug,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidSlug):
			http.Error(w, "invalid slug", http.StatusBadRequest)
		case errors.Is(err, repo.ErrNotFound):
			http.Error(w, "bin not found", http.StatusNotFound)
		case errors.Is(err, service.ErrBinExpired):
			http.Error(w, "bin is expired", http.StatusGone)
		default:
			slog.ErrorContext(r.Context(), "failed to inspect bin", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	requests := make([]inspectBinRequest, len(out.Requests))
	for i, req := range out.Requests {
		requests[len(requests)-1-i] = inspectBinRequest{
			ID:          req.ID.String(),
			SequenceNum: req.SequenceNum,
			Method:      req.Method,
			Path:        req.Path,
			ContentType: req.ContentType,
			BodySize:    req.BodySize,
			CapturedAt:  req.CapturedAt.UTC().Format(time.RFC3339),
		}
	}

	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	captureURL := fmt.Sprintf("%s://%s/gofer/%s/", scheme, r.Host, out.Slug)

	data := inspectBinData{
		Slug:        out.Slug,
		CreatedAt:   out.CreatedAt.UTC().Format(time.RFC3339),
		ExpiresAt:   out.ExpiresAt.UTC().Format(time.RFC3339),
		CaptureURL:  captureURL,
		CurlExample: fmt.Sprintf("curl -X POST %s \\\n  -H \"Content-Type: application/json\" \\\n  -d '{\"hello\":\"world\"}'", captureURL),
		Requests:    requests,
		SSEEndpoint: fmt.Sprintf("/api/bins/%s/sse", out.Slug),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "bin.html", data); err != nil {
		slog.ErrorContext(r.Context(), "failed to render bin", "error", err)
	}
}
