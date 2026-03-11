package main

import (
	"net/http"
	"time"

	"github.com/w-h-a/gofer/internal/service"
)

type handler struct {
	svc        *service.Service
	defaultTTL time.Duration
	version    string
}

func (h *handler) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.handleHealthz)
	mux.HandleFunc("POST /api/bins", h.handleCreateBin)
	mux.HandleFunc("GET /api/bins/{slug}/sse", h.handleSubscribeToBin)
	mux.HandleFunc("/gofer/{slug}/{path...}", h.handleCaptureRequest)
	mux.HandleFunc("GET /api/bins/{slug}", h.handleViewBin)
	mux.HandleFunc("GET /api/requests/{id}", h.handleViewCapturedRequest)
	return mux
}

func newHandler(svc *service.Service, defaultTTL time.Duration, version string) *handler {
	return &handler{svc: svc, defaultTTL: defaultTTL, version: version}
}
