package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ProjectBarks/subway-pcb/service/internal/middleware"
	"github.com/ProjectBarks/subway-pcb/service/internal/model"
	"github.com/ProjectBarks/subway-pcb/service/internal/plugin"
)

// --- Preset API ---

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

	var preset model.Preset
	if isHTMX(r) {
		r.ParseForm()
		preset.Name = r.FormValue("name")
		preset.PluginName = r.FormValue("plugin_name")
		preset.Values = parseValuesFromForm(r)
	} else {
		if err := json.NewDecoder(r.Body).Decode(&preset); err != nil {
			jsonError(w, "invalid body", http.StatusBadRequest)
			return
		}
	}

	preset.ID = fmt.Sprintf("preset-%d", time.Now().UnixMilli())
	preset.OwnerEmail = user.Email
	preset.CreatedAt = time.Now()
	preset.UpdatedAt = time.Now()

	if err := s.store.CreatePreset(&preset); err != nil {
		jsonError(w, "create failed", http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		w.Header().Set("HX-Trigger", "presetCreated")
		fmt.Fprintf(w, `<div class="text-green-400 text-sm">Preset "%s" saved</div>`, preset.Name)
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonResponse(w, preset)
}

func (s *Server) handleUpdatePreset(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user := middleware.UserFromContext(r.Context())

	existing, ok := getOrError(w, func() (*model.Preset, error) { return s.store.GetPreset(id) }, "preset not found", http.StatusNotFound)
	if !ok {
		return
	}

	if existing.IsBuiltIn {
		jsonError(w, "cannot modify built-in preset", http.StatusForbidden)
		return
	}

	if !requireOwnerOrAdmin(w, user, existing.OwnerEmail) {
		return
	}

	if isHTMX(r) {
		r.ParseForm()
		if name := r.FormValue("name"); name != "" {
			existing.Name = name
		}
		existing.Values = parseValuesFromForm(r)
	} else {
		var update model.Preset
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			jsonError(w, "invalid body", http.StatusBadRequest)
			return
		}
		if update.Name != "" {
			existing.Name = update.Name
		}
		if update.Values != nil {
			existing.Values = update.Values
		}
	}

	existing.UpdatedAt = time.Now()
	if err := s.store.UpdatePreset(existing); err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		w.Header().Set("HX-Trigger", "presetUpdated")
		fmt.Fprintf(w, `<div class="text-green-400 text-sm">Preset updated</div>`)
		return
	}
	jsonResponse(w, existing)
}

func (s *Server) handleDeletePreset(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user := middleware.UserFromContext(r.Context())

	existing, ok := getOrError(w, func() (*model.Preset, error) { return s.store.GetPreset(id) }, "preset not found", http.StatusNotFound)
	if !ok {
		return
	}

	if existing.IsBuiltIn {
		jsonError(w, "cannot delete built-in preset", http.StatusForbidden)
		return
	}

	if !requireOwnerOrAdmin(w, user, existing.OwnerEmail) {
		return
	}

	if err := s.store.DeletePreset(id); err != nil {
		jsonError(w, "delete failed", http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		w.Header().Set("HX-Trigger", "presetDeleted")
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// getConfigFields returns config fields for a plugin, whether built-in or DB-stored.
func (s *Server) getConfigFields(pluginName string) []plugin.ConfigField {
	if p, ok := s.plugins.Get(pluginName); ok {
		return p.ConfigFields()
	}
	dbPlugin, _ := s.store.GetPlugin(pluginName)
	if dbPlugin == nil || len(dbPlugin.ConfigFields) == 0 {
		return nil
	}
	var fields []plugin.ConfigField
	json.Unmarshal(dbPlugin.ConfigFields, &fields)
	return fields
}
