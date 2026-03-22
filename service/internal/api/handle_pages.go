package api

import (
	"log"
	"net/http"

	"golang.org/x/sync/errgroup"

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

	cards, err := s.buildBoardCards(r.Context(), user, boards)
	if err != nil {
		log.Printf("api: build board cards error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
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

	cards, err := s.buildBoardCards(r.Context(), user, boards)
	if err != nil {
		log.Printf("api: build board cards error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	ui.BoardGrid(cards).Render(r.Context(), w)
}

// --- Community ---

func (s *Server) handleCommunity(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	var plugins []model.Plugin
	var installed []model.Plugin

	g, _ := errgroup.WithContext(r.Context())
	g.Go(func() error {
		var err error
		plugins, err = s.store.ListPublishedPlugins()
		return err
	})
	g.Go(func() error {
		var err error
		installed, err = s.store.ListInstalledPlugins(user.Email)
		return err
	})
	if err := g.Wait(); err != nil {
		log.Printf("api: community error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

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

	var plugins []model.Plugin
	var installed []model.Plugin

	g, _ := errgroup.WithContext(r.Context())
	g.Go(func() error {
		var err error
		plugins, err = s.store.SearchPlugins(q, sort)
		return err
	})
	g.Go(func() error {
		var err error
		installed, err = s.store.ListInstalledPlugins(user.Email)
		return err
	})
	if err := g.Wait(); err != nil {
		log.Printf("api: community search error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

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
