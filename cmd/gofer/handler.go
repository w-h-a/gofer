package main

import (
	"encoding/json"
	"net/http"

	"github.com/w-h-a/gofer/internal/service"
)

type handler struct {
	svc *service.Service
}

func (h *handler) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.handleHealthz)
	mux.HandleFunc("POST /api/bins", h.handleCreateBin)
	return mux
}

func newHandler(svc *service.Service) *handler {
	return &handler{svc: svc}
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
