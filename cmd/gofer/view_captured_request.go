package main

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/w-h-a/gofer/internal/client/repo"
	"github.com/w-h-a/gofer/internal/service"
)

type viewCapturedRequestResponse struct {
	ID          string              `json:"id"`
	BinID       string              `json:"bin_id"`
	SequenceNum int                 `json:"sequence_num"`
	Method      string              `json:"method"`
	Path        string              `json:"path"`
	Headers     map[string][]string `json:"headers"`
	QueryParams map[string][]string `json:"query_params"`
	ContentType string              `json:"content_type"`
	RemoteAddr  string              `json:"remote_addr"`
	BodySize    int                 `json:"body_size"`
	CapturedAt  string              `json:"captured_at"`
	Body        string              `json:"body"`
}

func (h *handler) handleViewCapturedRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if _, err := uuid.Parse(id); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request id"})
		return
	}

	out, err := h.svc.ViewCapturedRequest(r.Context(), service.ViewCapturedRequestInput{
		ID: id,
	})
	if err != nil {
		switch {
		case errors.Is(err, repo.ErrNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "request not found"})
		default:
			slog.Error("failed to view captured request", "error", err)
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, viewCapturedRequestResponse{
		ID:          out.ID.String(),
		BinID:       out.BinID.String(),
		SequenceNum: out.SequenceNum,
		Method:      out.Method,
		Path:        out.Path,
		Headers:     out.Headers,
		QueryParams: out.QueryParams,
		ContentType: out.ContentType,
		RemoteAddr:  out.RemoteAddr,
		BodySize:    out.BodySize,
		CapturedAt:  out.CapturedAt.UTC().Format(time.RFC3339),
		Body:        string(out.Body),
	})
}
