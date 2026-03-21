package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/ProjectBarks/subway-pcb/service/internal/middleware"
	"github.com/ProjectBarks/subway-pcb/service/internal/model"
	"github.com/ProjectBarks/subway-pcb/service/internal/mta"
	"github.com/ProjectBarks/subway-pcb/service/internal/plugin"
	"github.com/ProjectBarks/subway-pcb/service/internal/store"
	"github.com/ProjectBarks/subway-pcb/service/ui"
)

// ServerConfig holds all dependencies for the HTTP server.
type ServerConfig struct {
	Aggregator     *mta.Aggregator
	Store          store.Store
	PluginRegistry *plugin.Registry
	Boards         map[string]*BoardData
	AuthConfig     middleware.AuthConfig
	StaticDir      string // optional: directory to serve at /static/
	DevMode        bool   // enable dev-only routes (e.g. /landing)
}

// Server is the HTTP API server.
type Server struct {
	aggregator *mta.Aggregator
	store      store.Store
	plugins    *plugin.Registry
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
		plugins:    cfg.PluginRegistry,
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
	r.Get("/api/v1/device-state", s.handleDeviceState)
	r.Get("/api/v1/device-board", s.handleDeviceBoard)
	r.Get("/api/v1/device-script", s.handleDeviceScript)

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
				r.Get("/preview", s.handleBoardPreview)
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

// --- Board View ---

func (s *Server) handleBoardView(w http.ResponseWriter, r *http.Request) {
	mac := chi.URLParam(r, "mac")
	user := middleware.UserFromContext(r.Context())

	device, err := s.store.GetDevice(mac)
	if err != nil || device == nil {
		http.Error(w, "device not found", http.StatusNotFound)
		return
	}

	access, _ := s.store.ListAccessByDevice(mac)

	data := s.buildBoardData(user, device, mac, access)
	ui.BoardPage(data).Render(r.Context(), w)
}

// buildBoardCards creates board card data for dashboard display.
func (s *Server) buildBoardCards(user *model.User, boards []model.Device) []ui.BoardCard {
	// Prefetch installed plugins once for name resolution.
	var installedPlugins []model.Plugin
	if user != nil {
		own, _ := s.store.ListPluginsByAuthor(user.Email)
		inst, _ := s.store.ListInstalledPlugins(user.Email)
		seen := make(map[string]bool)
		for _, p := range own {
			seen[p.ID] = true
			installedPlugins = append(installedPlugins, p)
		}
		for _, p := range inst {
			if !seen[p.ID] {
				installedPlugins = append(installedPlugins, p)
			}
		}
	}

	cards := make([]ui.BoardCard, len(boards))
	for i, d := range boards {
		cards[i].Device = d
		if d.PresetID != "" {
			t, _ := s.store.GetPreset(d.PresetID)
			cards[i].Preset = t
		}
		cards[i].ActivePluginName = s.resolvePluginName(d.PluginName, installedPlugins)
		if board, ok := s.boards[BoardModelKey(d.BoardModelID)]; ok {
			cards[i].BoardModelName = board.Manifest.Name
		}

		// LED preview data
		luaSource, _ := s.resolveDeviceLua(d.MAC)
		cards[i].LuaSource = luaSource
		cards[i].BoardURL = BoardURLPath(d.BoardModelID)
		config := s.buildDeviceConfig(d.MAC, d.PluginName)
		configBytes, _ := json.Marshal(config)
		cards[i].ConfigJSON = string(configBytes)
	}
	return cards
}

// resolvePluginName returns the human-readable name for a plugin ID.
func (s *Server) resolvePluginName(id string, installedPlugins []model.Plugin) string {
	if id == "" {
		return ""
	}
	for _, p := range s.plugins.List() {
		if p.Name() == id {
			return p.Name()
		}
	}
	for _, p := range installedPlugins {
		if p.ID == id {
			return p.Name
		}
	}
	return id
}

// buildBoardData builds the typed data for the board view and controls.
func (s *Server) buildBoardData(user *model.User, device *model.Device, mac string, access []model.DeviceAccess) ui.BoardData {
	pluginName := device.PluginName
	if pluginName == "" {
		pluginName = "track"
	}

	// Get config fields for the active plugin (built-in or DB)
	configFields := s.getConfigFields(pluginName)
	configGroups := plugin.GroupedFields(configFields)

	// Build config values: defaults -> preset -> device overrides
	configValues := make(map[string]string)
	for _, f := range configFields {
		configValues[f.Key] = f.Default
	}
	if device.PresetID != "" {
		preset, _ := s.store.GetPreset(device.PresetID)
		if preset != nil {
			for k, v := range preset.Values {
				configValues[k] = v
			}
		}
	}
	for k, v := range device.PluginConfig {
		configValues[k] = v
	}

	// Get presets for this plugin: built-in + user's own
	allPresets, _ := s.store.ListPresets()
	var pluginPresets []model.Preset
	for _, t := range allPresets {
		if t.PluginName == pluginName && (t.IsBuiltIn || t.OwnerEmail == user.Email) {
			pluginPresets = append(pluginPresets, t)
		}
	}

	// Get user's own + installed plugins from DB for Browse tab
	ownPlugins, _ := s.store.ListPluginsByAuthor(user.Email)
	installedOnly, _ := s.store.ListInstalledPlugins(user.Email)
	// Merge: own plugins first, then installed (skip duplicates)
	seen := make(map[string]bool)
	var installedPlugins []model.Plugin
	for _, p := range ownPlugins {
		seen[p.ID] = true
		installedPlugins = append(installedPlugins, p)
	}
	for _, p := range installedOnly {
		if !seen[p.ID] {
			installedPlugins = append(installedPlugins, p)
		}
	}

	// Filter built-in plugins by board compatibility
	var compatiblePlugins []plugin.Plugin
	boardData := s.boards[BoardModelKey(device.BoardModelID)]
	var boardFeatures []string
	if boardData != nil {
		boardFeatures = boardData.Manifest.Features
	}
	for _, p := range s.plugins.List() {
		if plugin.IsPluginCompatible(p.RequiredFeatures(), boardFeatures) {
			compatiblePlugins = append(compatiblePlugins, p)
		}
	}

	// Resolve active plugin Lua source
	var activeLuaSource string
	if p, ok := s.plugins.Get(pluginName); ok {
		activeLuaSource = p.LuaSource()
	} else {
		dbPlugin, _ := s.store.GetPlugin(pluginName)
		if dbPlugin != nil {
			activeLuaSource = dbPlugin.LuaSource
		}
	}

	return ui.BoardData{
		User:             user,
		Device:           device,
		Presets:          pluginPresets,
		Access:           access,
		Plugins:          compatiblePlugins,
		InstalledPlugins: installedPlugins,
		ActiveMAC:        mac,
		ConfigGroups:     configGroups,
		ConfigValues:     configValues,
		BoardURL:         BoardURLPath(device.BoardModelID),
		ActiveLuaSource:  activeLuaSource,
	}
}

func (s *Server) handleSetPlugin(w http.ResponseWriter, r *http.Request) {
	mac := chi.URLParam(r, "mac")
	r.ParseForm()
	pluginName := r.FormValue("plugin")

	// Accept built-in plugins by name or DB plugins by ID
	var requiredFeatures []string
	if builtIn, ok := s.plugins.Get(pluginName); ok {
		requiredFeatures = builtIn.RequiredFeatures()
	} else {
		dbPlugin, err := s.store.GetPlugin(pluginName)
		if err != nil || dbPlugin == nil {
			http.Error(w, "unknown plugin", http.StatusBadRequest)
			return
		}
	}

	device, err := s.store.GetDevice(mac)
	if err != nil || device == nil {
		http.Error(w, "device not found", http.StatusNotFound)
		return
	}

	// Check board compatibility
	if len(requiredFeatures) > 0 {
		if board, ok := s.boards[BoardModelKey(device.BoardModelID)]; ok {
			if !plugin.IsPluginCompatible(requiredFeatures, board.Manifest.Features) {
				http.Error(w, "plugin incompatible with this board", http.StatusBadRequest)
				return
			}
		}
	}

	device.PluginName = pluginName
	if err := s.store.UpsertDevice(device); err != nil {
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}

	s.renderControls(w, r, mac)
}

func (s *Server) handleSetPreset(w http.ResponseWriter, r *http.Request) {
	mac := chi.URLParam(r, "mac")
	r.ParseForm()
	presetID := r.FormValue("preset_id")

	if _, err := s.store.GetPreset(presetID); err != nil {
		http.Error(w, "preset not found", http.StatusBadRequest)
		return
	}

	device, err := s.store.GetDevice(mac)
	if err != nil || device == nil {
		http.Error(w, "device not found", http.StatusNotFound)
		return
	}

	device.PresetID = presetID
	if err := s.store.UpsertDevice(device); err != nil {
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}

	s.renderControls(w, r, mac)
}

func (s *Server) handleSetName(w http.ResponseWriter, r *http.Request) {
	mac := chi.URLParam(r, "mac")
	r.ParseForm()
	name := r.FormValue("name")

	device, err := s.store.GetDevice(mac)
	if err != nil || device == nil {
		http.Error(w, "device not found", http.StatusNotFound)
		return
	}

	device.Name = name
	if err := s.store.UpsertDevice(device); err != nil {
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}

	s.renderControls(w, r, mac)
}

func (s *Server) handleSetPluginConfig(w http.ResponseWriter, r *http.Request) {
	mac := chi.URLParam(r, "mac")
	r.ParseForm()

	device, err := s.store.GetDevice(mac)
	if err != nil || device == nil {
		http.Error(w, "device not found", http.StatusNotFound)
		return
	}

	// Collect all form values as config
	config := make(map[string]string)
	for key, vals := range r.Form {
		if len(vals) > 0 && key != "" {
			config[key] = vals[0]
		}
	}

	device.PluginConfig = config
	if err := s.store.UpsertDevice(device); err != nil {
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}

	s.renderControls(w, r, mac)
}

func (s *Server) handleBoardPreview(w http.ResponseWriter, r *http.Request) {
	mac := chi.URLParam(r, "mac")
	device, _ := s.store.GetDevice(mac)
	if device == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<div class="text-sm text-gray-400">Preview for %s</div>`, device.Name)
}

// --- Access Management ---

func (s *Server) handleGrantAccess(w http.ResponseWriter, r *http.Request) {
	mac := chi.URLParam(r, "mac")
	user := middleware.UserFromContext(r.Context())
	r.ParseForm()
	email := r.FormValue("email")

	if email == "" {
		http.Error(w, "email required", http.StatusBadRequest)
		return
	}

	access := &model.DeviceAccess{
		MAC:       mac,
		UserEmail: email,
		GrantedBy: user.Email,
		GrantedAt: time.Now(),
	}

	if err := s.store.GrantAccess(access); err != nil {
		http.Error(w, "grant failed", http.StatusInternalServerError)
		return
	}

	s.renderAccessPanel(w, r, mac)
}

func (s *Server) handleRevokeAccess(w http.ResponseWriter, r *http.Request) {
	mac := chi.URLParam(r, "mac")
	email := chi.URLParam(r, "email")

	if err := s.store.RevokeAccess(mac, email); err != nil {
		http.Error(w, "revoke failed", http.StatusInternalServerError)
		return
	}

	s.renderAccessPanel(w, r, mac)
}

func (s *Server) renderAccessPanel(w http.ResponseWriter, r *http.Request, mac string) {
	access, _ := s.store.ListAccessByDevice(mac)
	device, _ := s.store.GetDevice(mac)
	ui.DeviceAccess(device, access).Render(r.Context(), w)
}

func (s *Server) renderControls(w http.ResponseWriter, r *http.Request, mac string) {
	user := middleware.UserFromContext(r.Context())
	device, _ := s.store.GetDevice(mac)
	access, _ := s.store.ListAccessByDevice(mac)

	data := s.buildBoardData(user, device, mac, access)
	ui.BoardControls(data).Render(r.Context(), w)
}

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

	existing, err := s.store.GetPreset(id)
	if err != nil || existing == nil {
		jsonError(w, "preset not found", http.StatusNotFound)
		return
	}

	if existing.IsBuiltIn {
		jsonError(w, "cannot modify built-in preset", http.StatusForbidden)
		return
	}

	if existing.OwnerEmail != user.Email && !user.IsAdmin() {
		jsonError(w, "access denied", http.StatusForbidden)
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

	existing, err := s.store.GetPreset(id)
	if err != nil || existing == nil {
		jsonError(w, "preset not found", http.StatusNotFound)
		return
	}

	if existing.IsBuiltIn {
		jsonError(w, "cannot delete built-in preset", http.StatusForbidden)
		return
	}

	if existing.OwnerEmail != user.Email && !user.IsAdmin() {
		jsonError(w, "access denied", http.StatusForbidden)
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

// --- Plugin API ---

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

	p, err := s.store.GetPlugin(id)
	if err != nil || p == nil {
		jsonError(w, "plugin not found", http.StatusNotFound)
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

	existing, err := s.store.GetPlugin(id)
	if err != nil || existing == nil {
		jsonError(w, "plugin not found", http.StatusNotFound)
		return
	}

	if existing.AuthorEmail != user.Email && !user.IsAdmin() {
		jsonError(w, "access denied", http.StatusForbidden)
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

	existing, err := s.store.GetPlugin(id)
	if err != nil || existing == nil {
		jsonError(w, "plugin not found", http.StatusNotFound)
		return
	}

	if existing.AuthorEmail != user.Email && !user.IsAdmin() {
		jsonError(w, "access denied", http.StatusForbidden)
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

	existing, err := s.store.GetPlugin(id)
	if err != nil || existing == nil {
		jsonError(w, "plugin not found", http.StatusNotFound)
		return
	}

	if existing.AuthorEmail != user.Email && !user.IsAdmin() {
		jsonError(w, "access denied", http.StatusForbidden)
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

func (s *Server) handleInstallPlugin(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user := middleware.UserFromContext(r.Context())

	if err := s.store.InstallPlugin(user.Email, id); err != nil {
		jsonError(w, "install failed", http.StatusInternalServerError)
		return
	}
	s.store.IncrementPluginInstalls(id)

	if isHTMX(r) {
		fmt.Fprintf(w, `<button hx-delete="/api/v1/plugins/%s/install" hx-swap="outerHTML" class="flex-1 h-10 border border-border-subtle text-text-secondary rounded-full transition-all font-medium hover:border-status-error hover:text-status-error">Uninstall</button>`, id)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleUninstallPlugin(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user := middleware.UserFromContext(r.Context())

	if err := s.store.UninstallPlugin(user.Email, id); err != nil {
		jsonError(w, "uninstall failed", http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		fmt.Fprintf(w, `<button hx-post="/api/v1/plugins/%s/install" hx-swap="outerHTML" class="flex-1 h-10 bg-white text-black rounded-full transition-all font-medium">Install</button>`, id)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Users (Admin) ---

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers()
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	jsonResponse(w, users)
}

// --- State & Health (unchanged core logic) ---

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stations := s.aggregator.GetStations()
	ts := s.aggregator.GetTimestamp()

	type stateJSONTrain struct {
		Route  string `json:"route"`
		Status string `json:"status"`
	}
	type stateJSONStation struct {
		StopID string           `json:"stop_id"`
		Trains []stateJSONTrain `json:"trains"`
	}

	stationList := make([]stateJSONStation, 0, len(stations))
	for _, st := range stations {
		trains := make([]stateJSONTrain, 0, len(st.Trains))
		for _, t := range st.Trains {
			trains = append(trains, stateJSONTrain{
				Route:  t.Route,
				Status: t.Status.String(),
			})
		}
		stationList = append(stationList, stateJSONStation{
			StopID: st.StopId,
			Trains: trains,
		})
	}

	resp := struct {
		Timestamp uint64             `json:"timestamp"`
		Stations  []stateJSONStation `json:"stations"`
	}{
		Timestamp: ts,
		Stations:  stationList,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

type healthResponse struct {
	Status        string  `json:"status"`
	Uptime        string  `json:"uptime"`
	UptimeSeconds float64 `json:"uptime_seconds"`
	LastUpdate    string  `json:"last_update"`
	StationCount  int     `json:"station_count"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	uptime := time.Since(s.startTime)
	lastUpdate := s.aggregator.LastUpdate()

	lastUpdateStr := "never"
	if !lastUpdate.IsZero() {
		lastUpdateStr = fmt.Sprintf("%ds ago", int(time.Since(lastUpdate).Seconds()))
	}

	resp := healthResponse{
		Status:        "ok",
		Uptime:        uptime.Round(time.Second).String(),
		UptimeSeconds: uptime.Seconds(),
		LastUpdate:    lastUpdateStr,
		StationCount:  s.aggregator.StationCount(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// --- Helpers ---

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
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
