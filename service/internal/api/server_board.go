package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ProjectBarks/subway-pcb/service/internal/middleware"
	"github.com/ProjectBarks/subway-pcb/service/internal/model"
	"github.com/ProjectBarks/subway-pcb/service/internal/plugin"
	"github.com/ProjectBarks/subway-pcb/service/ui"
)

// --- Board View ---

func (s *Server) handleBoardView(w http.ResponseWriter, r *http.Request) {
	mac := chi.URLParam(r, "mac")
	user := middleware.UserFromContext(r.Context())

	device, ok := getOrError(w, func() (*model.Device, error) { return s.store.GetDevice(mac) }, "device not found", http.StatusNotFound)
	if !ok {
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
		dbPlugin, ok := getOrError(w, func() (*model.Plugin, error) { return s.store.GetPlugin(pluginName) }, "unknown plugin", http.StatusBadRequest)
		if !ok {
			return
		}
		_ = dbPlugin
	}

	device, ok := getOrError(w, func() (*model.Device, error) { return s.store.GetDevice(mac) }, "device not found", http.StatusNotFound)
	if !ok {
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

	device, ok := getOrError(w, func() (*model.Device, error) { return s.store.GetDevice(mac) }, "device not found", http.StatusNotFound)
	if !ok {
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

	device, ok := getOrError(w, func() (*model.Device, error) { return s.store.GetDevice(mac) }, "device not found", http.StatusNotFound)
	if !ok {
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

	device, ok := getOrError(w, func() (*model.Device, error) { return s.store.GetDevice(mac) }, "device not found", http.StatusNotFound)
	if !ok {
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
	access, err := s.store.ListAccessByDevice(mac)
	if err != nil {
		log.Printf("api: list access error for %s: %v", mac, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	device, ok := getOrError(w, func() (*model.Device, error) { return s.store.GetDevice(mac) }, "device not found", http.StatusNotFound)
	if !ok {
		return
	}

	ui.DeviceAccess(device, access).Render(r.Context(), w)
}

func (s *Server) renderControls(w http.ResponseWriter, r *http.Request, mac string) {
	user := middleware.UserFromContext(r.Context())

	device, err := s.store.GetDevice(mac)
	if err != nil || device == nil {
		log.Printf("api: render controls error for %s: %v", mac, err)
		http.Error(w, "device not found", http.StatusNotFound)
		return
	}

	access, err := s.store.ListAccessByDevice(mac)
	if err != nil {
		log.Printf("api: list access error for %s: %v", mac, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	data := s.buildBoardData(user, device, mac, access)
	ui.BoardControls(data).Render(r.Context(), w)
}
