package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/ProjectBarks/subway-pcb/service/internal/middleware"
	"github.com/ProjectBarks/subway-pcb/service/internal/model"
)

func (s *Server) handleListPlugins(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	author := r.URL.Query().Get("author")

	var plugins []model.Plugin
	var err error
	if author == "me" {
		plugins, err = s.store.ListPluginsByAuthor(user.Email)
	} else {
		plugins, err = s.store.ListPublishedPlugins()
	}
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	jsonResponse(w, plugins)
}

func (s *Server) handleCreatePlugin(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	var p model.Plugin
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}

	p.ID = uuid.New().String()
	p.AuthorEmail = user.Email
	p.Type = "lua"
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()

	if err := s.store.CreatePlugin(&p); err != nil {
		jsonError(w, "create failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonResponse(w, p)
}

func (s *Server) handleGetPlugin(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user := middleware.UserFromContext(r.Context())

	p, ok := getOrError(w, func() (*model.Plugin, error) { return s.store.GetPlugin(id) }, "plugin not found", http.StatusNotFound)
	if !ok {
		return
	}

	// Access check: published, or owner, or admin
	if !p.IsPublished && p.AuthorEmail != user.Email && !user.IsAdmin() {
		jsonError(w, "access denied", http.StatusForbidden)
		return
	}

	jsonResponse(w, p)
}

func (s *Server) handleUpdatePlugin(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user := middleware.UserFromContext(r.Context())

	existing, ok := getOrError(w, func() (*model.Plugin, error) { return s.store.GetPlugin(id) }, "plugin not found", http.StatusNotFound)
	if !ok {
		return
	}

	if !requireOwnerOrAdmin(w, user, existing.AuthorEmail) {
		return
	}

	var update model.Plugin
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}

	if update.Name != "" {
		existing.Name = update.Name
	}
	if update.LuaSource != "" {
		existing.LuaSource = update.LuaSource
	}
	if update.Description != "" {
		existing.Description = update.Description
	}
	if update.Category != "" {
		existing.Category = update.Category
	}
	if update.ConfigFields != nil {
		existing.ConfigFields = update.ConfigFields
	}
	existing.UpdatedAt = time.Now()

	if err := s.store.UpdatePlugin(existing); err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, existing)
}

func (s *Server) handleDeletePlugin(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user := middleware.UserFromContext(r.Context())

	existing, ok := getOrError(w, func() (*model.Plugin, error) { return s.store.GetPlugin(id) }, "plugin not found", http.StatusNotFound)
	if !ok {
		return
	}

	if !requireOwnerOrAdmin(w, user, existing.AuthorEmail) {
		return
	}

	if err := s.store.DeletePlugin(id); err != nil {
		jsonError(w, "delete failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePublishPlugin(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user := middleware.UserFromContext(r.Context())

	existing, ok := getOrError(w, func() (*model.Plugin, error) { return s.store.GetPlugin(id) }, "plugin not found", http.StatusNotFound)
	if !ok {
		return
	}

	if !requireOwnerOrAdmin(w, user, existing.AuthorEmail) {
		return
	}

	existing.IsPublished = !existing.IsPublished
	existing.UpdatedAt = time.Now()

	if err := s.store.UpdatePlugin(existing); err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, existing)
}
