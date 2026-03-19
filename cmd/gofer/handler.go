package main

import (
	"html/template"
	"net/http"
	"time"

	"github.com/w-h-a/gofer/cmd/gofer/web"
	"github.com/w-h-a/gofer/internal/service"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type handler struct {
	svc        *service.Service
	tmpl       *template.Template
	defaultTTL time.Duration
	version    string
	tracer     trace.Tracer
}

func (h *handler) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", h.handleHome)
	mux.HandleFunc("GET /bins/{slug}", h.handleInspectBin)
	mux.HandleFunc("POST /api/bins", h.handleCreateBin)
	mux.HandleFunc("GET /api/bins/{slug}", h.handleViewBin)
	mux.HandleFunc("GET /api/requests/{id}", h.handleViewCapturedRequest)
	mux.HandleFunc("GET /api/bins/{slug}/sse", h.handleSubscribeToBin)
	mux.HandleFunc("/gofer/{slug}/{path...}", h.handleCaptureRequest)
	mux.HandleFunc("GET /healthz", h.handleHealthz)
	return mux
}

func newHandler(svc *service.Service, defaultTTL time.Duration, version string) *handler {
	tmpl := template.Must(template.ParseFS(web.Templates, "templates/*.html"))
	return &handler{
		svc:        svc,
		tmpl:       tmpl,
		defaultTTL: defaultTTL,
		version:    version,
		tracer:     otel.Tracer("github.com/w-h-a/gofer/cmd/gofer"),
	}
}
