package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/ProjectBarks/subway-pcb/service/internal/middleware"
	"github.com/ProjectBarks/subway-pcb/service/internal/model"
	"github.com/ProjectBarks/subway-pcb/service/internal/mta"
	"github.com/ProjectBarks/subway-pcb/service/internal/store"
)

// ServerConfig holds all dependencies for the HTTP server.
type ServerConfig struct {
	Aggregator *mta.Aggregator
	Store      store.Store
	Boards     map[string]*BoardData
	AuthConfig middleware.AuthConfig
	StaticDir  string // optional: directory to serve at /static/
	DevMode    bool   // enable dev-only routes (e.g. /landing)
}

// Server is the HTTP API server.
type Server struct {
	aggregator *mta.Aggregator
	store      store.Store
	boards     map[string]*BoardData
	authConfig middleware.AuthConfig
	staticDir  string
	devMode    bool
	startTime  time.Time
	router     chi.Router
}

// NewServer creates a new API server with all dependencies.
func NewServer(cfg ServerConfig) *Server {
	s := &Server{
		aggregator: cfg.Aggregator,
		store:      cfg.Store,
		boards:     cfg.Boards,
		authConfig: cfg.AuthConfig,
		staticDir:  cfg.StaticDir,
		devMode:    cfg.DevMode,
		startTime:  time.Now(),
	}
	s.buildRouter()
	return s
}

func (s *Server) buildRouter() {
	r := chi.NewRouter()
	r.Use(chimw.Logger, chimw.Recoverer)

	// Dev-only routes
	if s.devMode {
		r.Get("/landing", s.handleLanding)
	}

	// Device routes — accessible on all hosts including RESTRICTED_HOST
	r.Group(func(r chi.Router) {
		// Build board defaults map for device auto-registration middleware
		boardDefaults := make(map[string]middleware.BoardDefaults, len(s.boards))
		for key, b := range s.boards {
			boardDefaults[key] = middleware.BoardDefaults{
				DefaultPlugin: b.Manifest.DefaultPlugin,
				DefaultPreset: b.Manifest.DefaultPreset,
			}
		}
		r.Use(middleware.DeviceAutoRegister(s.store, boardDefaults))

		r.Get("/api/v1/device-state", s.handleDeviceState)
		r.Get("/api/v1/device-board", s.handleDeviceBoard)
		r.Get("/api/v1/device-script", s.handleDeviceScript)
	})

	// App routes — restricted to ALLOWED_HOSTS
	r.Group(func(r chi.Router) {
		r.Use(middleware.HostRestriction(s.authConfig.AllowedHosts))

		r.Get("/api/v1/state", s.handleState)
		r.Get("/health", s.handleHealth)

		// Serve frontend static assets (JS/CSS bundles) when static-dir is set
		if s.staticDir != "" {
			fileServer := http.FileServer(http.Dir(s.staticDir))
			r.Handle("/static/*", http.StripPrefix("/static/", fileServer))
		}

		// Authenticated routes
		authMW := middleware.Auth(s.store, s.authConfig)
		requireBoardAccess := middleware.RequireBoardAccess(s.store)

		r.Group(func(r chi.Router) {
			r.Use(authMW)

			r.Get("/", s.handleRootRedirect)
			r.Get("/boards", s.handleDashboard)
			r.Get("/community", s.handleCommunity)
			r.Get("/community/search", s.handleCommunitySearch)
			r.Get("/editor", s.handleEditor)
			r.Get("/partials/board-list", s.handleBoardListPartial)

			// Board view (per-user access check — admins bypass)
			r.Route("/boards/{mac}", func(r chi.Router) {
				r.Use(requireBoardAccess)
				r.Get("/", s.handleBoardView)
				r.Put("/plugin", s.handleSetPlugin)
				r.Put("/preset", s.handleSetPreset)
				r.Put("/name", s.handleSetName)
				r.Put("/config", s.handleSetPluginConfig)
				// Access management
				r.Post("/access", s.handleGrantAccess)
				r.Delete("/access/{email}", s.handleRevokeAccess)
			})

			// Preset API
			r.Get("/api/v1/presets", s.handleListPresets)
			r.Post("/api/v1/presets", s.handleCreatePreset)
			r.Put("/api/v1/presets/{id}", s.handleUpdatePreset)
			r.Delete("/api/v1/presets/{id}", s.handleDeletePreset)

			// Plugin API
			r.Get("/api/v1/plugins", s.handleListPlugins)
			r.Post("/api/v1/plugins", s.handleCreatePlugin)
			r.Get("/api/v1/plugins/{id}", s.handleGetPlugin)
			r.Put("/api/v1/plugins/{id}", s.handleUpdatePlugin)
			r.Delete("/api/v1/plugins/{id}", s.handleDeletePlugin)
			r.Post("/api/v1/plugins/{id}/publish", s.handlePublishPlugin)
			r.Post("/api/v1/plugins/{id}/install", s.handleInstallPlugin)
			r.Delete("/api/v1/plugins/{id}/install", s.handleUninstallPlugin)

			// Admin-only
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireAdmin)
				r.Get("/api/v1/users", s.handleListUsers)
			})
		})
	})

	s.router = r
}

// Handler returns the http.Handler for this server.
func (s *Server) Handler() http.Handler {
	return s.router
}

// --- Helpers ---

// getOrError looks up a resource and writes a JSON error if not found.
// Returns the resource and true on success, or nil and false on failure.
func getOrError[T any](w http.ResponseWriter, fn func() (*T, error), msg string, status int) (*T, bool) {
	result, err := fn()
	if err != nil || result == nil {
		jsonError(w, msg, status)
		return nil, false
	}
	return result, true
}

// requireOwnerOrAdmin checks that the user owns the resource or is an admin.
// Returns true if access is granted, false if a 403 was written.
func requireOwnerOrAdmin(w http.ResponseWriter, user *model.User, ownerEmail string) bool {
	if ownerEmail != user.Email && !user.IsAdmin() {
		jsonError(w, "access denied", http.StatusForbidden)
		return false
	}
	return true
}

func jsonResponse(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"data": data})
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]any{"error": msg})
}

// parseValuesFromForm collects all form fields prefixed with "val_" as theme values.
func parseValuesFromForm(r *http.Request) map[string]string {
	values := make(map[string]string)
	for key, vals := range r.Form {
		if len(key) > 4 && key[:4] == "val_" && len(vals) > 0 {
			values[key[4:]] = vals[0]
		}
	}
	return values
}
