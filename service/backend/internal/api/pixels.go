package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	pb "github.com/ProjectBarks/subway-pcb/server/gen/subwaypb"
	"github.com/ProjectBarks/subway-pcb/server/internal/mode"
	"github.com/ProjectBarks/subway-pcb/server/internal/model"
	"github.com/ProjectBarks/subway-pcb/server/internal/mta"
	"github.com/ProjectBarks/subway-pcb/server/internal/store"
	"google.golang.org/protobuf/proto"
)

var stripSizes = [9]int{97, 102, 55, 81, 70, 21, 22, 19, 11}

const totalLEDs = 478

// LEDMap maps flat LED index -> station ID.
type LEDMap struct {
	stationIDs []string
}

// StationIDs returns the flat station ID mapping for use by the mode system.
func (m *LEDMap) StationIDs() []string {
	return m.stationIDs
}

// LoadLEDMap loads the led_map.json file.
func LoadLEDMap(path string) (*LEDMap, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read led_map.json: %w", err)
	}

	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse led_map.json: %w", err)
	}

	ids := make([]string, totalLEDs)
	offset := 0
	for strip := 0; strip < 9; strip++ {
		for pixel := 0; pixel < stripSizes[strip]; pixel++ {
			key := fmt.Sprintf("%d,%d", strip, pixel)
			if sid, ok := raw[key]; ok {
				ids[offset] = sid
			}
			offset++
		}
	}

	log.Printf("ledmap: loaded %d entries from %s", len(raw), path)
	return &LEDMap{stationIDs: ids}, nil
}

// PixelRenderer generates PixelFrame protobuf from aggregator state.
type PixelRenderer struct {
	ledMap *LEDMap
	store  store.Store
	modes  *mode.Registry
	seq    uint32
}

// NewPixelRenderer creates a renderer with the given LED map.
func NewPixelRenderer(ledMap *LEDMap) *PixelRenderer {
	return &PixelRenderer{ledMap: ledMap}
}

// SetDeps sets the store and mode registry.
func (pr *PixelRenderer) SetDeps(s store.Store, m *mode.Registry) {
	pr.store = s
	pr.modes = m
}

// RenderFrame renders a pixel frame for the given device.
func (pr *PixelRenderer) RenderFrame(agg *mta.Aggregator, mac string) ([]byte, error) {
	device, err := pr.store.GetDevice(mac)
	if err != nil || device == nil {
		return nil, fmt.Errorf("device not found: %s", mac)
	}

	modeName := device.Mode
	if modeName == "" {
		modeName = "track"
	}

	m, ok := pr.modes.Get(modeName)
	if !ok {
		return nil, fmt.Errorf("unknown mode: %s", modeName)
	}

	// Build config: field defaults -> theme values -> device overrides
	config := make(map[string]string)
	for _, f := range m.ConfigFields() {
		config[f.Key] = f.Default
	}
	if device.ThemeID != "" {
		theme, _ := pr.store.GetTheme(device.ThemeID)
		if theme != nil {
			for k, v := range theme.Values {
				config[k] = v
			}
		}
	}
	for k, v := range device.ModeConfig {
		config[k] = v
	}

	pixels, err := m.Render(mode.RenderContext{
		Aggregator: agg,
		StationIDs: pr.ledMap.stationIDs,
		Device:     device,
		Config:     config,
		TotalLEDs:  totalLEDs,
	})
	if err != nil {
		return nil, err
	}

	pr.seq++
	frame := &pb.PixelFrame{
		Timestamp: uint64(time.Now().Unix()),
		Sequence:  pr.seq,
		LedCount:  totalLEDs,
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

	device := &model.Device{
		MAC:       mac,
		Mode:      "track",
		ThemeID:   "track-classic-mta",
		LastSeen:  time.Now(),
		CreatedAt: time.Now(),
	}
	if fwVer := r.Header.Get("X-Firmware-Version"); fwVer != "" {
		device.FirmwareVer = fwVer
	}

	if err := s.store.UpsertDevice(device); err != nil {
		log.Printf("api: failed to auto-register device %s: %v", mac, err)
	} else {
		log.Printf("api: auto-registered new device %s", mac)
	}
}
