// Package api serves the HTTP API and, optionally, the built frontend.
package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/LBess/doti-projects/server/internal/project"
	"github.com/LBess/doti-projects/server/internal/store"
)

type Store interface {
	List(ctx context.Context, status string) ([]project.Project, error)
	Ping(ctx context.Context) error
}

// NewHandler routes the API. If staticDir is non-empty, everything that isn't
// an API route is served from it (the built frontend).
func NewHandler(store Store, log *slog.Logger, staticDir string) http.Handler {
	handler := &handler{store: store, log: log}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/projects", handler.projects)
	mux.HandleFunc("GET /healthz", handler.health)
	if staticDir != "" {
		mux.Handle("/", http.FileServer(http.Dir(staticDir)))
	}
	return mux
}

type handler struct {
	store Store
	log   *slog.Logger
}

func (handler *handler) projects(writer http.ResponseWriter, request *http.Request) {
	status := request.URL.Query().Get("status")
	if status == "" {
		status = store.StatusActive
	}
	if status != store.StatusActive && status != store.StatusCompleted {
		http.Error(writer, "status must be 'active' or 'completed'", http.StatusBadRequest)
		return
	}

	projects, err := handler.store.List(request.Context(), status)
	if err != nil {
		handler.log.Error("list projects", "error", err)
		http.Error(writer, "internal error", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(writer).Encode(projects); err != nil {
		handler.log.Error("encode projects", "error", err)
	}
}

func (handler *handler) health(writer http.ResponseWriter, request *http.Request) {
	if err := handler.store.Ping(request.Context()); err != nil {
		handler.log.Error("health check", "error", err)
		http.Error(writer, "database unavailable", http.StatusServiceUnavailable)
		return
	}
	writer.WriteHeader(http.StatusOK)
}
