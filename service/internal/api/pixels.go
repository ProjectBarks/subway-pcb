package api

import (
	"fmt"
	"log"
	"net/http"
	"time"

	pb "github.com/ProjectBarks/subway-pcb/service/gen/subwaypb"
	"github.com/ProjectBarks/subway-pcb/service/internal/model"
	"github.com/ProjectBarks/subway-pcb/service/internal/mta"
	"github.com/ProjectBarks/subway-pcb/service/internal/plugin"
	"github.com/ProjectBarks/subway-pcb/service/internal/store"
	"google.golang.org/protobuf/proto"
)

// PixelRenderer generates PixelFrame protobuf from aggregator state.
type PixelRenderer struct {
	boards  map[string]*BoardData
	store   store.Store
	plugins *plugin.Registry
	seq     uint32
}

// NewPixelRenderer creates a renderer with the loaded board data.
func NewPixelRenderer(boards map[string]*BoardData) *PixelRenderer {
	return &PixelRenderer{boards: boards}
}

// SetDeps sets the store and plugin registry.
func (pr *PixelRenderer) SetDeps(s store.Store, p *plugin.Registry) {
	pr.store = s
	pr.plugins = p
}

// RenderFrame renders a pixel frame for the given device.
func (pr *PixelRenderer) RenderFrame(agg *mta.Aggregator, mac string) ([]byte, error) {
	device, err := pr.store.GetDevice(mac)
	if err != nil || device == nil {
		return nil, fmt.Errorf("device not found: %s", mac)
	}

	board := pr.boards[BoardModelKey(device.BoardModelID)]
	if board == nil {
		return nil, fmt.Errorf("unknown board model: %s", device.BoardModelID)
	}

	pluginName := device.PluginName
	if pluginName == "" {
		pluginName = board.Manifest.DefaultPlugin
		if pluginName == "" {
			pluginName = "track"
		}
	}

	p, ok := pr.plugins.Get(pluginName)
	if !ok {
		return nil, fmt.Errorf("unknown plugin: %s", pluginName)
	}

	// Build config: field defaults -> preset values -> device overrides
	config := make(map[string]string)
	for _, f := range p.ConfigFields() {
		config[f.Key] = f.Default
	}
	if device.PresetID != "" {
		preset, _ := pr.store.GetPreset(device.PresetID)
		if preset != nil {
			for k, v := range preset.Values {
				config[k] = v
			}
		}
	}
	for k, v := range device.PluginConfig {
		config[k] = v
	}

	pixels, err := p.Render(plugin.RenderContext{
		Aggregator: agg,
		StationIDs: board.StationIDs,
		Device:     device,
		Config:     config,
		TotalLEDs:  board.Manifest.LEDCount,
		Strips:     board.Manifest.Strips,
	})
	if err != nil {
		return nil, err
	}

	pr.seq++
	frame := &pb.PixelFrame{
		Timestamp: uint64(time.Now().Unix()),
		Sequence:  pr.seq,
		LedCount:  uint32(board.Manifest.LEDCount),
		Pixels:    pixels,
	}
	return proto.Marshal(frame)
}

// handlePixels serves the pre-rendered PixelFrame.
func (s *Server) handlePixels(w http.ResponseWriter, r *http.Request) {
	if s.pixelRenderer == nil {
		http.Error(w, "pixel renderer not configured", http.StatusServiceUnavailable)
		return
	}

	mac := r.URL.Query().Get("device")
	if mac == "" {
		mac = r.Header.Get("X-Device-ID")
	}

	if mac != "" {
		s.autoRegisterDevice(mac, r)
	}

	if mac == "" {
		http.Error(w, "missing device ID", http.StatusBadRequest)
		return
	}

	data, err := s.pixelRenderer.RenderFrame(s.aggregator, mac)
	if err != nil {
		log.Printf("api: pixel render error: %v", err)
		http.Error(w, "render error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-protobuf")
	w.Write(data)
}

// autoRegisterDevice creates or updates a device record when it connects.
func (s *Server) autoRegisterDevice(mac string, r *http.Request) {
	existing, _ := s.store.GetDevice(mac)
	if existing != nil {
		s.store.UpdateDeviceLastSeen(mac)
		return
	}

	boardModelID := r.Header.Get("X-Board-Model")
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
