package main

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/w-h-a/gofer/internal/client/repo"
	"github.com/w-h-a/gofer/internal/service"
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

func (h *handler) handleCaptureRequest(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	path := "/" + r.PathValue("path")

	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{Error: "request body too large"})
			return
		}
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "failed to read request body"})
		return
	}

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
		case errors.Is(err, repo.ErrNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "bin not found"})
		case errors.Is(err, service.ErrBinExpired):
			writeJSON(w, http.StatusGone, errorResponse{Error: "bin is expired"})
		default:
			slog.Error("failed to capture request", "error", err)
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, captureRequestResponse{
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
