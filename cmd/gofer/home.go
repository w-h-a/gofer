package main

import (
	"log/slog"
	"net/http"
)

func (h *handler) handleHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "home.html", nil); err != nil {
		slog.ErrorContext(r.Context(), "failed to render home", "error", err)
	}
}
