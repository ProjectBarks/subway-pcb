package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ProjectBarks/subway-pcb/service/internal/middleware"
	"github.com/ProjectBarks/subway-pcb/service/ui"
)

func (s *Server) handleInstallPlugin(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user := middleware.UserFromContext(r.Context())

	if err := s.store.InstallPlugin(user.Email, id); err != nil {
		jsonError(w, "install failed", http.StatusInternalServerError)
		return
	}
	s.store.IncrementPluginInstalls(id)

	ui.PluginInstallButton(id, false).Render(r.Context(), w)
}

func (s *Server) handleUninstallPlugin(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user := middleware.UserFromContext(r.Context())

	if err := s.store.UninstallPlugin(user.Email, id); err != nil {
		jsonError(w, "uninstall failed", http.StatusInternalServerError)
		return
	}

	ui.PluginInstallButton(id, true).Render(r.Context(), w)
}
