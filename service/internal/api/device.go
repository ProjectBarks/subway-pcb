package api

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	pb "github.com/ProjectBarks/subway-pcb/service/gen/subwaypb"
	"github.com/ProjectBarks/subway-pcb/service/internal/model"
	"google.golang.org/protobuf/proto"
)

// handleDeviceState serves the realtime DeviceState protobuf.
// GET /api/v1/device-state
func (s *Server) handleDeviceState(w http.ResponseWriter, r *http.Request) {
	mac := r.Header.Get("X-Device-ID")
	hardware := r.Header.Get("X-Hardware")

	if mac != "" {
		s.autoRegisterDevice(mac, r)
	}

	// Resolve board
	boardKey := BoardModelKey(hardware)
	board := s.boards[boardKey]

	// Resolve active plugin + Lua source
	luaSource, pluginName := s.resolveDeviceLua(mac)
	scriptHash := sha256Hex(luaSource)

	// Compute board hash
	var boardHash string
	if board != nil {
		boardHash = computeBoardHash(board)
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
	hardware := r.Header.Get("X-Hardware")
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
		Hash:       computeBoardHash(board),
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
	mac := r.Header.Get("X-Device-ID")
	luaSource, pluginName := s.resolveDeviceLua(mac)

	// Get plugin description
	var description string
	if p, ok := s.plugins.Get(pluginName); ok {
		description = p.Description()
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
	pluginName = "track"

	if mac != "" {
		device, _ := s.store.GetDevice(mac)
		if device != nil && device.PluginName != "" {
			pluginName = device.PluginName
		}
	}

	// Check built-in plugins first
	if p, ok := s.plugins.Get(pluginName); ok {
		return p.LuaSource(), pluginName
	}

	// Check DB plugins
	if mac != "" {
		dbPlugin, _ := s.store.GetPlugin(pluginName)
		if dbPlugin != nil && dbPlugin.LuaSource != "" {
			return dbPlugin.LuaSource, pluginName
		}
	}

	// Fallback to track
	if p, ok := s.plugins.Get("track"); ok {
		return p.LuaSource(), "track"
	}

	return "", pluginName
}

// buildDeviceConfig returns the merged config for a device's active plugin.
func (s *Server) buildDeviceConfig(mac, pluginName string) map[string]string {
	config := make(map[string]string)

	// Field defaults
	if p, ok := s.plugins.Get(pluginName); ok {
		for _, f := range p.ConfigFields() {
			config[f.Key] = f.Default
		}
	} else {
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

// computeBoardHash returns a SHA256 hash of the board's led_map + strip sizes.
func computeBoardHash(board *BoardData) string {
	h := sha256.New()
	fmt.Fprintf(h, "id=%s;leds=%d;", board.Manifest.ID, board.Manifest.LEDCount)
	for i, sz := range board.Manifest.Strips {
		fmt.Fprintf(h, "s%d=%d;", i, sz)
	}
	for i, sid := range board.StationIDs {
		if sid != "" {
			fmt.Fprintf(h, "%d=%s;", i, sid)
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// sha256Hex returns the hex-encoded SHA256 of a string.
func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:])
}

// autoRegisterDevice creates or updates a device record when it connects.
func (s *Server) autoRegisterDevice(mac string, r *http.Request) {
	existing, _ := s.store.GetDevice(mac)
	if existing != nil {
		s.store.UpdateDeviceLastSeen(mac)
		return
	}

	boardModelID := r.Header.Get("X-Hardware")
	if boardModelID == "" {
		boardModelID = "nyc-subway/v1"
	}

	// Look up board manifest for default plugin/preset
	defaultPlugin := "track"
	defaultPreset := "track-classic-mta"
	if board, ok := s.boards[boardModelID]; ok {
		if board.Manifest.DefaultPlugin != "" {
			defaultPlugin = board.Manifest.DefaultPlugin
		}
		if board.Manifest.DefaultPreset != "" {
			defaultPreset = board.Manifest.DefaultPreset
		}
	}

	device := &model.Device{
		MAC:          mac,
		BoardModelID: boardModelID,
		PluginName:   defaultPlugin,
		PresetID:     defaultPreset,
		LastSeen:     time.Now(),
		CreatedAt:    time.Now(),
	}
	if fwVer := r.Header.Get("X-Firmware-Version"); fwVer != "" {
		device.FirmwareVer = fwVer
	}

	if err := s.store.UpsertDevice(device); err != nil {
		log.Printf("api: failed to auto-register device %s: %v", mac, err)
	} else {
		log.Printf("api: auto-registered new device %s (board=%s)", mac, boardModelID)
	}
}
