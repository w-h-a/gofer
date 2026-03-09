package main

import (
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
	mux.HandleFunc("GET /api/bins/{slug}", h.handleViewBin)
	mux.HandleFunc("GET /api/requests/{id}", h.handleViewCapturedRequest)
	mux.HandleFunc("/gofer/{slug}/{path...}", h.handleCaptureRequest)
	return mux
}

func newHandler(svc *service.Service) *handler {
	return &handler{svc: svc}
}
