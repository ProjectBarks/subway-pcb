package api

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	pb "github.com/ProjectBarks/subway-pcb/service/gen/subwaypb"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"

	"github.com/ProjectBarks/subway-pcb/service/internal/middleware"
	"github.com/ProjectBarks/subway-pcb/service/internal/model"
)

// handleDeviceState serves the realtime DeviceState protobuf.
// GET /api/v1/device-state
func (s *Server) handleDeviceState(w http.ResponseWriter, r *http.Request) {
	mac := r.Header.Get(middleware.HeaderDeviceID)
	hardware := r.Header.Get(middleware.HeaderHardware)

	// Resolve board
	boardKey := BoardModelKey(hardware)
	board := s.boards[boardKey]

	// Resolve active plugin + Lua source
	luaSource, pluginName, err := s.resolveDeviceLua(mac)
	if err != nil {
		log.Printf("api: resolve device lua error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	scriptHash := sha256Hex(luaSource)

	// Compute board hash
	var boardHash string
	if board != nil {
		boardHash = board.Hash()
	}

	// Get MTA stations
	stations := s.aggregator.GetStations()

	// Build merged config
	config, err := s.buildDeviceConfig(r.Context(), mac, pluginName)
	if err != nil {
		log.Printf("api: device config error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	state := &pb.DeviceState{
		ScriptHash: scriptHash,
		BoardHash:  boardHash,
		Timestamp:  uint64(time.Now().Unix()),
		Stations:   stations,
		Config:     config,
	}

	data, err := proto.Marshal(state)
	if err != nil {
		log.Printf("api: device state marshal error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-protobuf")
	w.Write(data)
}

// handleDeviceBoard serves the static DeviceBoard protobuf.
// GET /api/v1/device-board
func (s *Server) handleDeviceBoard(w http.ResponseWriter, r *http.Request) {
	hardware := r.Header.Get(middleware.HeaderHardware)
	boardKey := BoardModelKey(hardware)
	board := s.boards[boardKey]
	if board == nil {
		http.Error(w, "unknown board model", http.StatusNotFound)
		return
	}

	ledMap := make(map[uint32]string)
	for i, sid := range board.StationIDs {
		if sid != "" {
			ledMap[uint32(i)] = sid
		}
	}

	stripSizes := make([]uint32, len(board.Manifest.Strips))
	for i, sz := range board.Manifest.Strips {
		stripSizes[i] = uint32(sz)
	}

	msg := &pb.DeviceBoard{
		Hash:       board.Hash(),
		BoardId:    board.Manifest.ID,
		LedCount:   uint32(board.Manifest.LEDCount),
		StripSizes: stripSizes,
		LedMap:     ledMap,
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		log.Printf("api: device board marshal error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-protobuf")
	w.Write(data)
}

// handleDeviceScript serves the Lua script protobuf.
// GET /api/v1/device-script
func (s *Server) handleDeviceScript(w http.ResponseWriter, r *http.Request) {
	mac := r.Header.Get(middleware.HeaderDeviceID)
	luaSource, pluginName, err := s.resolveDeviceLua(mac)
	if err != nil {
		log.Printf("api: resolve device lua error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Get plugin description from DB
	var description string
	if dbPlugin, _ := s.store.GetPlugin(pluginName); dbPlugin != nil {
		description = dbPlugin.Description
	}

	// Build config defaults
	config, err := s.buildDeviceConfig(r.Context(), mac, pluginName)
	if err != nil {
		log.Printf("api: device config error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	msg := &pb.DeviceScript{
		Hash:              sha256Hex(luaSource),
		LuaSource:         luaSource,
		PluginName:        pluginName,
		PluginDescription: description,
		Config:            config,
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		log.Printf("api: device script marshal error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-protobuf")
	w.Write(data)
}

// resolveDeviceLua returns the Lua source and plugin name for a device.
func (s *Server) resolveDeviceLua(mac string) (luaSource, pluginName string, err error) {
	if mac != "" {
		device, err := s.store.GetDevice(mac)
		if err != nil {
			return "", "", err
		}
		if device != nil && device.PluginName != "" {
			pluginName = device.PluginName
		}
	}

	if pluginName == "" {
		return "", "", nil
	}

	dbPlugin, err := s.store.GetPlugin(pluginName)
	if err != nil {
		return "", pluginName, err
	}
	if dbPlugin != nil && dbPlugin.LuaSource != "" {
		return dbPlugin.LuaSource, pluginName, nil
	}

	return "", pluginName, nil
}

// buildDeviceConfig returns the merged config for a device's active plugin.
func (s *Server) buildDeviceConfig(ctx context.Context, mac, pluginName string) (map[string]string, error) {
	config := make(map[string]string)

	// Fetch plugin and device concurrently (independent queries).
	var dbPlugin *model.Plugin
	var device *model.Device

	g, _ := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		dbPlugin, err = s.store.GetPlugin(pluginName)
		return err
	})
	if mac != "" {
		g.Go(func() error {
			var err error
			device, err = s.store.GetDevice(mac)
			return err
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Field defaults from DB plugin
	if dbPlugin != nil && len(dbPlugin.ConfigFields) > 0 {
		var fields []struct {
			Key     string `json:"key"`
			Default string `json:"default"`
		}
		json.Unmarshal(dbPlugin.ConfigFields, &fields)
		for _, f := range fields {
			config[f.Key] = f.Default
		}
	}

	if device == nil {
		return config, nil
	}

	// Preset overrides
	if device.PresetID != "" {
		preset, err := s.store.GetPreset(device.PresetID)
		if err != nil {
			return nil, err
		}
		if preset != nil {
			for k, v := range preset.Values {
				config[k] = v
			}
		}
	}
	// Device overrides
	for k, v := range device.PluginConfig {
		config[k] = v
	}

	return config, nil
}

// sha256Hex returns the hex-encoded SHA256 of a string.
func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:])
}

