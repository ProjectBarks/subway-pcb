package api

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	pb "github.com/ProjectBarks/subway-pcb/service/gen/subwaypb"
	"github.com/ProjectBarks/subway-pcb/service/internal/middleware"
	"google.golang.org/protobuf/proto"
)

// handleDeviceState serves the realtime DeviceState protobuf.
// GET /api/v1/device-state
func (s *Server) handleDeviceState(w http.ResponseWriter, r *http.Request) {
	mac := r.Header.Get(HeaderDeviceID)
	hardware := middleware.HardwareFromContext(r.Context())

	// Resolve board
	boardKey := BoardModelKey(hardware)
	board := s.boards[boardKey]

	// Resolve active plugin + Lua source
	luaSource, pluginName := s.resolveDeviceLua(mac)
	scriptHash := sha256Hex(luaSource)

	// Compute board hash
	var boardHash string
	if board != nil {
		boardHash = board.Hash()
	}

	// Get MTA stations
	stations := s.aggregator.GetStations()

	// Build merged config
	config := s.buildDeviceConfig(mac, pluginName)

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
	hardware := middleware.HardwareFromContext(r.Context())
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
	mac := r.Header.Get(HeaderDeviceID)
	luaSource, pluginName := s.resolveDeviceLua(mac)

	// Get plugin description from DB
	var description string
	if dbPlugin, _ := s.store.GetPlugin(pluginName); dbPlugin != nil {
		description = dbPlugin.Description
	}

	// Build config defaults
	config := s.buildDeviceConfig(mac, pluginName)

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
func (s *Server) resolveDeviceLua(mac string) (luaSource, pluginName string) {
	if mac != "" {
		device, _ := s.store.GetDevice(mac)
		if device != nil && device.PluginName != "" {
			pluginName = device.PluginName
		}
	}

	if pluginName == "" {
		return "", ""
	}

	dbPlugin, _ := s.store.GetPlugin(pluginName)
	if dbPlugin != nil && dbPlugin.LuaSource != "" {
		return dbPlugin.LuaSource, pluginName
	}

	return "", pluginName
}

// buildDeviceConfig returns the merged config for a device's active plugin.
func (s *Server) buildDeviceConfig(mac, pluginName string) map[string]string {
	config := make(map[string]string)

	// Field defaults from DB plugin
	dbPlugin, _ := s.store.GetPlugin(pluginName)
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

	if mac == "" {
		return config
	}

	// Preset overrides
	device, _ := s.store.GetDevice(mac)
	if device != nil {
		if device.PresetID != "" {
			preset, _ := s.store.GetPreset(device.PresetID)
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
	}

	return config
}

// sha256Hex returns the hex-encoded SHA256 of a string.
func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:])
}

