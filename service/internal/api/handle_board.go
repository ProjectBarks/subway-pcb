package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/sync/errgroup"

	"github.com/ProjectBarks/subway-pcb/service/internal/middleware"
	"github.com/ProjectBarks/subway-pcb/service/internal/model"
	"github.com/ProjectBarks/subway-pcb/service/internal/plugin"
	"github.com/ProjectBarks/subway-pcb/service/internal/utils"
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

	data, err := s.buildBoardData(r.Context(), user, device, mac, access)
	if err != nil {
		log.Printf("api: build board data error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	ui.BoardPage(data).Render(r.Context(), w)
}

// buildBoardCards creates board card data for dashboard display.
func (s *Server) buildBoardCards(ctx context.Context, user *model.User, boards []model.Device) ([]ui.BoardCard, error) {
	// Prefetch installed plugins once for name resolution.
	var installedPlugins []model.Plugin
	if user != nil {
		var own, inst []model.Plugin
		g, _ := errgroup.WithContext(ctx)
		g.Go(func() error {
			var err error
			own, err = s.store.ListPluginsByAuthor(user.Email)
			return err
		})
		g.Go(func() error {
			var err error
			inst, err = s.store.ListInstalledPlugins(user.Email)
			return err
		})
		if err := g.Wait(); err != nil {
			return nil, err
		}

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
	g, _ := errgroup.WithContext(ctx)
	for i, d := range boards {
		g.Go(func() error {
			cards[i].Device = d
			if d.PresetID != "" {
				t, err := s.store.GetPreset(d.PresetID)
				if err != nil {
					return err
				}
				cards[i].Preset = t
			}
			cards[i].ActivePluginName = s.resolvePluginName(d.PluginName, installedPlugins)
			if board, ok := s.boards[BoardModelKey(d.BoardModelID)]; ok {
				cards[i].BoardModelName = board.Manifest.Name
			}

			luaSource, _, err := s.resolveDeviceLua(d.MAC)
			if err != nil {
				return err
			}
			cards[i].LuaSource = luaSource
			cards[i].BoardURL = BoardURLPath(d.BoardModelID)
			config, err := s.buildDeviceConfig(ctx, d.MAC, d.PluginName)
			if err != nil {
				return err
			}
			configBytes, _ := json.Marshal(config)
			cards[i].ConfigJSON = string(configBytes)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return cards, nil
}

// resolvePluginName returns the human-readable name for a plugin ID.
func (s *Server) resolvePluginName(id string, installedPlugins []model.Plugin) string {
	if id == "" {
		return ""
	}
	// Check DB (covers built-in and community plugins)
	if dbPlugin, _ := s.store.GetPlugin(id); dbPlugin != nil {
		return dbPlugin.Name
	}
	for _, p := range installedPlugins {
		if p.ID == id {
			return p.Name
		}
	}
	return id
}

// buildBoardData builds the typed data for the board view and controls.
func (s *Server) buildBoardData(ctx context.Context, user *model.User, device *model.Device, mac string, access []model.DeviceAccess) (ui.BoardData, error) {
	pluginName := device.PluginName

	// Fetch all independent data concurrently.
	var (
		dbPlugin         *model.Plugin
		preset           *model.Preset
		allPresets       []model.Preset
		ownPlugins       []model.Plugin
		installedOnly    []model.Plugin
		publishedPlugins []model.Plugin
	)

	g, _ := errgroup.WithContext(ctx)
	g.Go(utils.Bind1(&dbPlugin, s.store.GetPlugin, pluginName))
	g.Go(utils.Bind0(&allPresets, s.store.ListPresets))
	g.Go(utils.Bind1(&ownPlugins, s.store.ListPluginsByAuthor, user.Email))
	g.Go(utils.Bind1(&installedOnly, s.store.ListInstalledPlugins, user.Email))
	g.Go(utils.Bind0(&publishedPlugins, s.store.ListPublishedPlugins))
	if device.PresetID != "" {
		g.Go(utils.Bind1(&preset, s.store.GetPreset, device.PresetID))
	}
	if err := g.Wait(); err != nil {
		return ui.BoardData{}, err
	}

	// Config fields + values from the fetched plugin
	var configFields []plugin.ConfigField
	if dbPlugin != nil && len(dbPlugin.ConfigFields) > 0 {
		json.Unmarshal(dbPlugin.ConfigFields, &configFields)
	}
	configGroups := plugin.GroupedFields(configFields)

	configValues := make(map[string]string)
	for _, f := range configFields {
		configValues[f.Key] = f.Default
	}
	if preset != nil {
		for k, v := range preset.Values {
			configValues[k] = v
		}
	}
	for k, v := range device.PluginConfig {
		configValues[k] = v
	}

	// Filter presets for this plugin
	var pluginPresets []model.Preset
	for _, t := range allPresets {
		if t.PluginName == pluginName && (t.IsBuiltIn || t.OwnerEmail == user.Email) {
			pluginPresets = append(pluginPresets, t)
		}
	}

	// Merge own + installed plugins (dedup)
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

	// Filter published plugins by board compatibility
	boardData := s.boards[BoardModelKey(device.BoardModelID)]
	var boardFeatures []string
	if boardData != nil {
		boardFeatures = boardData.Manifest.Features
	}
	var compatiblePlugins []model.Plugin
	for _, p := range publishedPlugins {
		if plugin.IsPluginCompatible(p.RequiredFeatures, boardFeatures) {
			compatiblePlugins = append(compatiblePlugins, p)
		}
	}

	var activeLuaSource string
	if dbPlugin != nil {
		activeLuaSource = dbPlugin.LuaSource
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
	}, nil
}

func (s *Server) handleSetPlugin(w http.ResponseWriter, r *http.Request) {
	mac := chi.URLParam(r, "mac")
	r.ParseForm()
	pluginName := r.FormValue("plugin")

	// Look up plugin from DB
	dbPlugin, ok := getOrError(w, func() (*model.Plugin, error) { return s.store.GetPlugin(pluginName) }, "unknown plugin", http.StatusBadRequest)
	if !ok {
		return
	}
	requiredFeatures := dbPlugin.RequiredFeatures

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

	data, err := s.buildBoardData(r.Context(), user, device, mac, access)
	if err != nil {
		log.Printf("api: build board data error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	ui.BoardControls(data).Render(r.Context(), w)
}
