package api

import (
	"log"
	"net/http"

	"github.com/ProjectBarks/subway-pcb/service/internal/middleware"
	"github.com/ProjectBarks/subway-pcb/service/internal/model"
	"github.com/ProjectBarks/subway-pcb/service/ui"
)

// --- Landing (dev only) ---

func (s *Server) handleLanding(w http.ResponseWriter, r *http.Request) {
	ui.Landing().Render(r.Context(), w)
}

// --- Root Redirect ---

func (s *Server) handleRootRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/boards", http.StatusFound)
}

// --- Dashboard ---

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	var boards []model.Device
	var err error
	if user.IsAdmin() {
		boards, err = s.store.ListDevices()
	} else {
		boards, err = s.store.ListDevicesByUser(user.Email)
	}
	if err != nil {
		log.Printf("api: dashboard error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	cards := s.buildBoardCards(user, boards)
	ui.DashboardPage(user, boards, cards).Render(r.Context(), w)
}

func (s *Server) handleBoardListPartial(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	var boards []model.Device
	if user.IsAdmin() {
		boards, _ = s.store.ListDevices()
	} else {
		boards, _ = s.store.ListDevicesByUser(user.Email)
	}

	cards := s.buildBoardCards(user, boards)
	ui.BoardGrid(cards).Render(r.Context(), w)
}

// --- Community ---

func (s *Server) handleCommunity(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	plugins, err := s.store.ListPublishedPlugins()
	if err != nil {
		log.Printf("api: community error: %v", err)
		plugins = nil
	}

	installed, _ := s.store.ListInstalledPlugins(user.Email)
	installedSet := make(map[string]bool)
	for _, p := range installed {
		installedSet[p.ID] = true
	}

	ui.CommunityPage(user, plugins, installedSet).Render(r.Context(), w)
}

func (s *Server) handleCommunitySearch(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	q := r.URL.Query().Get("q")
	sort := r.URL.Query().Get("sort")

	plugins, err := s.store.SearchPlugins(q, sort)
	if err != nil {
		log.Printf("api: community search error: %v", err)
		plugins = nil
	}

	installed, _ := s.store.ListInstalledPlugins(user.Email)
	installedSet := make(map[string]bool)
	for _, p := range installed {
		installedSet[p.ID] = true
	}

	ui.CommunityPluginGrid(plugins, installedSet).Render(r.Context(), w)
}

// --- Editor ---

func (s *Server) handleEditor(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	ui.EditorPage(user).Render(r.Context(), w)
}
