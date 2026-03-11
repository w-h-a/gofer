package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/w-h-a/gofer/internal/service"
)

type createBinRequest struct {
	TTL string `json:"ttl"`
}

type createBinResponse struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
}

func (h *handler) handleCreateBin(w http.ResponseWriter, r *http.Request) {
	ttl := h.defaultTTL

	var req createBinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	if req.TTL != "" {
		parsed, err := time.ParseDuration(req.TTL)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid ttl"})
			return
		}
		ttl = parsed
	}

	out, err := h.svc.CreateBin(r.Context(), service.CreateBinInput{TTL: ttl})
	if err != nil {
		slog.Error("failed to create bin", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, createBinResponse{
		ID:        out.ID.String(),
		Slug:      out.Slug,
		CreatedAt: out.CreatedAt.UTC().Format(time.RFC3339),
		ExpiresAt: out.ExpiresAt.UTC().Format(time.RFC3339),
	})
}
