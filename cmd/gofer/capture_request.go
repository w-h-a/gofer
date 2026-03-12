package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/w-h-a/gofer/internal/client/repo"
	"github.com/w-h-a/gofer/internal/domain"
	"github.com/w-h-a/gofer/internal/service"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	maxBodySize = 1 << 20 // 1MB
)

type captureRequestResponse struct {
	ID          string `json:"id"`
	BinID       string `json:"bin_id"`
	SequenceNum int    `json:"sequence_num"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	ContentType string `json:"content_type"`
	BodySize    int    `json:"body_size"`
	CapturedAt  string `json:"captured_at"`
}

func (h *handler) handleSubscribeToBin(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	out, err := h.svc.SubscribeToBin(r.Context(), service.SubscribeToBinInput{
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
			slog.ErrorContext(r.Context(), "failed to subscribe to bin", "error", err)
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})
		}
		return
	}

	slog.InfoContext(r.Context(), "sse client connected", "slug", slug, "bin_id", out.BinID)
	defer slog.InfoContext(r.Context(), "sse client disconnected", "slug", slug, "bin_id", out.BinID)

	defer h.svc.UnsubscribeFromBin(context.Background(), service.UnsubscribeFromBinInput{
		BinID:   out.BinID,
		Channel: out.Channel,
	})

	flusher, ok := w.(http.Flusher)
	if !ok {
		slog.ErrorContext(r.Context(), "failed to subscribe to bin", "error", "response writer does not support flushing")
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case req, ok := <-out.Channel:
			if !ok {
				return
			}

			data, err := json.Marshal(captureRequestResponse{
				ID:          req.ID().String(),
				BinID:       req.BinID().String(),
				SequenceNum: req.SequenceNum(),
				Method:      req.Method(),
				Path:        req.Path(),
				ContentType: req.ContentType(),
				BodySize:    req.BodySize(),
				CapturedAt:  req.CapturedAt().UTC().Format(time.RFC3339),
			})
			if err != nil {
				slog.ErrorContext(r.Context(), "failed to marshal sse event", "error", err)
				return
			}

			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func (h *handler) handleCaptureRequest(w http.ResponseWriter, r *http.Request) {
	span := trace.SpanFromContext(r.Context())

	slug := r.PathValue("slug")
	path := "/" + r.PathValue("path")

	span.SetAttributes(
		attribute.String("bin.slug", slug),
		attribute.String("http.path", path),
	)

	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			span.SetAttributes(attribute.Int("outcome", http.StatusRequestEntityTooLarge))
			writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{Error: "request body too large"})
			return
		}
		span.SetAttributes(attribute.Int("outcome", http.StatusBadRequest))
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "failed to read request body"})
		return
	}

	span.SetAttributes(
		attribute.Int("headers.count", len(r.Header)),
		attribute.Int("body.size", len(body)),
		attribute.String("content_type", r.Header.Get("Content-Type")),
	)

	out, err := h.svc.CaptureRequest(r.Context(), service.CaptureRequestInput{
		Slug:        slug,
		Method:      r.Method,
		Path:        path,
		Headers:     r.Header,
		QueryParams: r.URL.Query(),
		Body:        body,
		ContentType: r.Header.Get("Content-Type"),
		RemoteAddr:  r.RemoteAddr,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidSlug):
			span.SetAttributes(attribute.Int("outcome", http.StatusBadRequest))
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid slug"})
		case errors.Is(err, repo.ErrNotFound):
			span.SetAttributes(attribute.Int("outcome", http.StatusNotFound))
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "bin not found"})
		case errors.Is(err, service.ErrBinExpired):
			span.SetAttributes(attribute.Int("outcome", http.StatusGone))
			writeJSON(w, http.StatusGone, errorResponse{Error: "bin is expired"})
		default:
			span.RecordError(err)
			span.SetStatus(codes.Error, "capture failed")
			span.SetAttributes(attribute.Int("outcome", http.StatusInternalServerError))
			slog.ErrorContext(r.Context(), "failed to capture request", "error", err)
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})
		}
		return
	}

	span.SetAttributes(
		attribute.Int("sequence_num", out.SequenceNum),
		attribute.Int("outcome", http.StatusCreated),
	)

	writeJSON(w, http.StatusCreated, captureRequestResponse{
		ID:          out.ID.String(),
		BinID:       out.BinID.String(),
		SequenceNum: out.SequenceNum,
		Method:      out.Method,
		Path:        out.Path,
		ContentType: out.ContentType,
		BodySize:    out.BodySize,
		CapturedAt:  out.CapturedAt.UTC().Format(time.RFC3339),
	})
}
