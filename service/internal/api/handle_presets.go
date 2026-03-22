package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ProjectBarks/subway-pcb/service/internal/middleware"
	"github.com/ProjectBarks/subway-pcb/service/internal/model"
)

func (s *Server) handleListPresets(w http.ResponseWriter, r *http.Request) {
	presets, err := s.store.ListPresets()
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	jsonResponse(w, presets)
}

func (s *Server) handleCreatePreset(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	r.ParseForm()

	preset := &model.Preset{
		ID:         fmt.Sprintf("preset-%d", time.Now().UnixMilli()),
		Name:       r.FormValue("name"),
		PluginName: r.FormValue("plugin_name"),
		OwnerEmail: user.Email,
		Values:     parseValuesFromForm(r),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.store.CreatePreset(preset); err != nil {
		http.Error(w, "create failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "presetCreated")
	fmt.Fprintf(w, `<div class="text-green-400 text-sm">Preset "%s" saved</div>`, preset.Name)
}

func (s *Server) handleUpdatePreset(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user := middleware.UserFromContext(r.Context())

	existing, ok := getOrError(w, func() (*model.Preset, error) { return s.store.GetPreset(id) }, "preset not found", http.StatusNotFound)
	if !ok {
		return
	}

	if existing.IsBuiltIn {
		http.Error(w, "cannot modify built-in preset", http.StatusForbidden)
		return
	}

	if !requireOwnerOrAdmin(w, user, existing.OwnerEmail) {
		return
	}

	r.ParseForm()
	if name := r.FormValue("name"); name != "" {
		existing.Name = name
	}
	existing.Values = parseValuesFromForm(r)
	existing.UpdatedAt = time.Now()

	if err := s.store.UpdatePreset(existing); err != nil {
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "presetUpdated")
	fmt.Fprintf(w, `<div class="text-green-400 text-sm">Preset updated</div>`)
}

func (s *Server) handleDeletePreset(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user := middleware.UserFromContext(r.Context())

	existing, ok := getOrError(w, func() (*model.Preset, error) { return s.store.GetPreset(id) }, "preset not found", http.StatusNotFound)
	if !ok {
		return
	}

	if existing.IsBuiltIn {
		http.Error(w, "cannot delete built-in preset", http.StatusForbidden)
		return
	}

	if !requireOwnerOrAdmin(w, user, existing.OwnerEmail) {
		return
	}

	if err := s.store.DeletePreset(id); err != nil {
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "presetDeleted")
	w.WriteHeader(http.StatusOK)
}
