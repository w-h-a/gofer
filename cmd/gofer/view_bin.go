package main

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/w-h-a/gofer/internal/client/repo"
	"github.com/w-h-a/gofer/internal/domain"
	"github.com/w-h-a/gofer/internal/service"
)

type viewBinResponse struct {
	ID        string                   `json:"id"`
	Slug      string                   `json:"slug"`
	CreatedAt string                   `json:"created_at"`
	ExpiresAt string                   `json:"expires_at"`
	Requests  []capturedRequestSummary `json:"requests"`
}

type capturedRequestSummary struct {
	ID          string `json:"id"`
	SequenceNum int    `json:"sequence_num"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	ContentType string `json:"content_type"`
	BodySize    int    `json:"body_size"`
	CapturedAt  string `json:"captured_at"`
}

func (h *handler) handleViewBin(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	out, err := h.svc.ViewBin(r.Context(), service.ViewBinInput{
		Slug: slug,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidSlug):
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid slug"})
		case errors.Is(err, repo.ErrNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "bin not found"})
		case errors.Is(err, service.ErrBinExpired):
			writeJSON(w, http.StatusGone, errorResponse{Error: "bin is expired"})
		default:
			slog.Error("failed to view bin", "error", err)
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})
		}
		return
	}

	requests := make([]capturedRequestSummary, len(out.Requests))
	for i, req := range out.Requests {
		requests[i] = capturedRequestSummary{
			ID:          req.ID.String(),
			SequenceNum: req.SequenceNum,
			Method:      req.Method,
			Path:        req.Path,
			ContentType: req.ContentType,
			BodySize:    req.BodySize,
			CapturedAt:  req.CapturedAt.UTC().Format(time.RFC3339),
		}
	}

	writeJSON(w, http.StatusOK, viewBinResponse{
		ID:        out.ID.String(),
		Slug:      out.Slug,
		CreatedAt: out.CreatedAt.UTC().Format(time.RFC3339),
		ExpiresAt: out.ExpiresAt.UTC().Format(time.RFC3339),
		Requests:  requests,
	})
}
